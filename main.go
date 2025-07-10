package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"example.com/v1/CF/get_config" // Assuming this is your local module for config
)

// DNS_REC represents a Cloudflare DNS record.
type DNS_REC struct {
	ZoneID  interface{} `json:"zone_id"`
	ID      string      `json:"id"`
	Name    interface{} `json:"name"`
	Typpe   interface{} `json:"type"` // Changed to 'Typpe' to match your original
	Proxied interface{} `json:"proxied"`
	Comment interface{} `json:"comment"`
	Tags    interface{} `json:"tags"`
	Ttl     interface{} `json:"ttl"`
	Content string      `json:"content"`
}

// String provides a formatted string representation of a DNS_REC.
func (d DNS_REC) String() string {
	return fmt.Sprintf("Name: %v\nType: %v\nProxied: %v\nComment: %v\nTags: %v\nTTL: %v\nContent: %v\n", d.Name, d.Typpe, d.Proxied, d.Comment, d.Tags, d.Ttl, d.Content)
}

// DNS_REC_LIST holds a slice of DNS_REC.
type DNS_REC_LIST struct {
	list []DNS_REC
}

// Z (Zone) struct, currently unused but kept for completeness.
type Z struct {
	ID     string
	Name   string
	Status string
}

func main() {
	// Get Cloudflare account email and API key from config.
	email, key := get_config.Get_account_info()
	// Create common HTTP headers for Cloudflare API authentication.
	header := createAuthHeader(email, key)

	// Get all zone IDs associated with the account.
	zoneIDs, err := getZoneIDs(header)
	if err != nil {
		log.Fatalf("Error getting zone IDs: %v", err)
	}

	// Get the current public IP address of the machine.
	localIP, err := getCurrentPublicIP()
	if err != nil {
		log.Fatalf("Error getting local IP: %v", err)
	}
	fmt.Println("local IP:", localIP)
	log.Println("local IP:", localIP)

	// Process each zone: fetch DNS records, check for updates, and update if necessary.
	for _, zoneID := range zoneIDs {
		processZone(zoneID, header, localIP)
	}
}

// createAuthHeader creates and returns standard HTTP headers for Cloudflare API authentication.
func createAuthHeader(email, key string) http.Header {
	header := make(http.Header)
	header.Add("X-Auth-Email", email)
	header.Add("Content-Type", "application/json")
	header.Add("X-Auth-Key", key)
	return header
}

// getZoneIDs fetches all zone IDs associated with the Cloudflare account.
func getZoneIDs(header http.Header) ([]string, error) {
	var zonesRequest = http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "https",
			Host:   "api.cloudflare.com",
			Path:   "/client/v4/zones",
		},
		Header: header,
	}

	netClient := &http.Client{}
	response, err := netClient.Do(&zonesRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to make zones request: %w", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read zones response body: %w", err)
	}

	var data struct {
		Result []struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal zones response: %w", err)
	}

	var zoneIDs []string
	for _, id := range data.Result {
		zoneIDs = append(zoneIDs, id.ID)
	}
	return zoneIDs, nil
}

// getCurrentPublicIP fetches the current public IP address of the machine from ipify.org.
func getCurrentPublicIP() (string, error) {
	resp, err := http.Get("https://api.ipify.org")
	if err != nil {
		return "", fmt.Errorf("failed to get public IP: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read public IP response body: %w", err)
	}
	return strings.TrimSpace(string(body)), nil // Trim whitespace
}

// processZone handles the logic for a single zone, including fetching DNS records,
// checking for new records from config, comparing IPs, and updating records.
func processZone(zoneID string, header http.Header, localIP string) {
	dnsRecords, err := getDNSRecords(zoneID, header)
	if err != nil {
		log.Printf("Error getting DNS records for zone %s: %v", zoneID, err)
		return
	}

	aRecords := filterARecords(dnsRecords)

	// Check for new records from create.yaml and add them.
	// This assumes 'get_config.Read_yaml()' returns a struct with fields matching DNS_REC for creation.
	newRecordConfig, err := get_config.Read_yaml()
	if err == nil { // Only proceed if the file exists and is readable.
		newDNS := DNS_REC{
			Content: newRecordConfig.Content,
			Name:    newRecordConfig.Name,
			Typpe:   newRecordConfig.Typpe,
			Proxied: newRecordConfig.Proxied,
			Comment: newRecordConfig.Comment,
			Tags:    newRecordConfig.Tags,
			Ttl:     newRecordConfig.Ttl,
		}
		aRecords.list = append(aRecords.list, newDNS)
		slices.Reverse(aRecords.list) // Reversing might be intended for priority.
	}

	// Check if the IP address has changed. If not, log and return.
	if len(aRecords.list) > 0 && localIP == aRecords.list[0].Content {
		log.Printf("IP address for zone %s has not changed", zoneID)
		fmt.Println("\033[0;31m IP address has not changed \033[0m")
		return
	}

	log.Printf("Updating A records for zone %s with IP: %s", zoneID, localIP)
	for _, record := range aRecords.list {
		if record.Typpe == "A" { // Ensure we only try to update A records.
			if err := updateDNSRecord(zoneID, record, localIP, header); err != nil {
				log.Printf("Error updating DNS record %s for zone %s: %v", record.Name, zoneID, err)
			}
		}
	}
}

// getDNSRecords retrieves all DNS records for a given zone ID.
func getDNSRecords(zoneID string, header http.Header) (DNS_REC_LIST, error) {
	var listRequest = http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "https",
			Host:   "api.cloudflare.com",
			Path:   fmt.Sprintf("/client/v4/zones/%s/dns_records", zoneID),
		},
		Header: header,
	}

	netClient := &http.Client{}
	response, err := netClient.Do(&listRequest)
	if err != nil {
		return DNS_REC_LIST{}, fmt.Errorf("failed to make DNS records request: %w", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return DNS_REC_LIST{}, fmt.Errorf("failed to read DNS records response body: %w", err)
	}

	var data struct {
		Result []map[string]interface{} `json:"result"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return DNS_REC_LIST{}, fmt.Errorf("failed to unmarshal DNS records response: %w", err)
	}

	dnsRecords := DNS_REC_LIST{}
	for _, record := range data.Result {
		rec := DNS_REC{
			ID:      record["id"].(string),
			ZoneID:  record["zone_id"],
			Name:    record["name"],
			Typpe:   record["type"],
			Proxied: record["proxied"],
			Comment: record["comment"],
			Tags:    record["tags"],
			Content: record["content"].(string),
			Ttl:     record["ttl"],
		}
		dnsRecords.list = append(dnsRecords.list, rec)
	}
	return dnsRecords, nil
}

// filterARecords filters a list of DNS records to only include "A" type records.
func filterARecords(allRecords DNS_REC_LIST) DNS_REC_LIST {
	aRecords := DNS_REC_LIST{}
	for _, record := range allRecords.list {
		if record.Typpe == "A" {
			log.Println(record.String())
			aRecords.list = append(aRecords.list, record)
		}
	}
	return aRecords
}

// updateDNSRecord sends a PATCH request to update a specific DNS record with a new IP.
func updateDNSRecord(zoneID string, record DNS_REC, newIP string, header http.Header) error {
	if record.Comment == nil {
		record.Comment = ""
	}

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, record.ID)

	payload := strings.NewReader(fmt.Sprintf(`{
        "content": "%s",
        "name": "%v",
        "proxied": %v,
        "type": "%v",
        "comment": "%v",
        "tags": [],
        "ttl": %v
    }`, newIP, record.Name, record.Proxied, record.Typpe, record.Comment, record.Ttl))

	req, err := http.NewRequest("PATCH", url, payload)
	if err != nil {
		return fmt.Errorf("failed to create update request: %w", err)
	}
	req.Header = header

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute update request: %w", err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read update response body: %w", err)
	}

	log.Printf("Update response for %s: %s", record.Name, string(body))
	// log.Println("HTTP Response:", res)
	return nil
}
