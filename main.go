package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"

	"example.com/v1/CF/get_config" // Assuming this is your local module for config
)

// CloudflareError represents an error response from Cloudflare API
type CloudflareError struct {
	Errors []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	Success bool `json:"success"`
}

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
		println("Error getting Zone IDS: " + err.Error())

		log.Fatalf("Error getting zone IDs: %v", err)
	}

	// Get the current public IP address of the machine.
	localIP, err := getCurrentPublicIP()
	if err != nil {
		log.Fatalf("Error getting local IP: %v", err)
	}
	fmt.Println("local IP:", localIP)
	log.Println("local IP:", localIP)

	// get ignore data

	ignore, err := get_config.Read_ignore()
	if err != nil {
		println("Could not get ignore " + err.Error())
		ignore = nil
	}

	// Process each zone: fetch DNS records, check for updates, and update if necessary.
	for _, zoneID := range zoneIDs {
		processZone(zoneID, header, localIP, ignore)
	}

}

// checkCloudflareResponse checks HTTP response status and parses Cloudflare API errors
func checkCloudflareResponse(resp *http.Response, body []byte, operation string) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil // Success status code
	}

	// Handle authentication errors specifically
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return fmt.Errorf("authentication failed for %s: invalid API key or email. Please check your Cloudflare credentials in the config file (HTTP %d)", operation, resp.StatusCode)
	}

	// Try to parse Cloudflare error response
	var cfError CloudflareError
	if err := json.Unmarshal(body, &cfError); err == nil && len(cfError.Errors) > 0 {
		return fmt.Errorf("%s failed: %s (HTTP %d)", operation, cfError.Errors[0].Message, resp.StatusCode)
	}

	// Fallback to generic error
	return fmt.Errorf("%s failed with HTTP status %d: %s", operation, resp.StatusCode, string(body))
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

	// Check for HTTP errors including authentication failures
	if err := checkCloudflareResponse(response, body, "getting zones"); err != nil {
		return nil, err
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
func processZone(zoneID string, header http.Header, localIP string, ignore *get_config.Ignore) {
	dnsRecords, err := getDNSRecords(zoneID, header)
	if err != nil {
		println("Error getting DNS Records for Zone " + zoneID + ": " + err.Error()) // Added space for readability
		log.Printf("Error getting DNS records for zone %s: %v", zoneID, err)
		return
	}

	aRecords := filterARecords(dnsRecords)

	// Check for new records from create.yaml and add them.
	newRecordConfig, err := get_config.Read_yaml()
	if err == nil && newRecordConfig != nil { // Only proceed if the file exists and is readable, and content is not nil
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
	} else if err != nil && !os.IsNotExist(err) { // Log error if it's not just "file not found"
		log.Printf("Warning: Could not read create.yaml: %v (Proceeding without adding new records)", err)
	}

	// --- NEW LOGIC: Filter out ignored records before IP change check ---
	var managedARecords DNS_REC_LIST // This will hold A records that are NOT ignored

	println("Applying ignore rules and filtering records...")
filterLoop: // Label for the filtering loop
	for _, record := range aRecords.list {
		// Only consider A records for management
		if record.Typpe != "A" {
			continue // Skip non-A records from being added to managedARecords
		}

		if ignore != nil { // Ensure ignore object exists
			for _, ignoreRecord := range ignore.Ignore {
				if ignoreRecord.Domain != "" {
					matched, err := get_config.Match_string(ignoreRecord.Domain, record.Name.(string))
					if err != nil {
						log.Printf("Error matching regex pattern '%s' against record name '%s': %v. This ignore rule will be skipped for this record.", ignoreRecord.Domain, record.Name, err)
						continue // Continue to the next ignoreRecord in the list if the current pattern is invalid.
					}
					if matched {
						// Print ignored domain in red text
						fmt.Printf("\033[0;31mSkipping DNS record '%s' (ID: %s) in zone '%s' as it matches ignore pattern '%s'.\033[0m\n", record.Name, record.ID, zoneID, ignoreRecord.Domain)
						log.Printf("Skipping DNS record '%s' (ID: %s) in zone '%s' as it matches ignore pattern '%s'.", record.Name, record.ID, zoneID, ignoreRecord.Domain)

						// Check for desired IP conflict (if desired_ip is specified)
						if ignoreRecord.DesiredIP != "" && ignoreRecord.DesiredIP != record.Content {

							fmt.Printf("\033[41mIP Mismatch for '%s': Current IP is '%s', Desired IPs are '%s'. (NO ACTION TAKEN)\033[0m\n",
								record.Name, record.Content, ignoreRecord.DesiredIP)
							log.Printf("IP Mismatch for '%s': Current IP is '%s', Desired IPs are '%s'. (NO ACTION TAKEN)",
								record.Name, record.Content, ignoreRecord.DesiredIP)
						}

						continue filterLoop // Skip to the next 'record' in aRecords.list (don't add to managedARecords)
					}
				}
			}
		}
		// If the record is an A record and was not ignored, add it to the managed list
		managedARecords.list = append(managedARecords.list, record)
	}
	// --- END NEW LOGIC ---

	// Check if the IP address has changed. This check now only applies to *managed* A records.
	if len(managedARecords.list) > 0 && localIP == managedARecords.list[0].Content {
		log.Printf("IP address for managed records in zone %s has not changed for first A record. Skipping updates.", zoneID)
		fmt.Println("\033[0;31m IP address has not changed for managed records \033[0m")
		return
	}

	log.Printf("Updating A records for zone %s with IP: %s", zoneID, localIP)
	// Now iterate only over the managed records for actual updates
	for _, record := range managedARecords.list {
		// No need for ignore checks here, as they've already been filtered out
		if err := updateDNSRecord(zoneID, record, localIP, header); err != nil {
			log.Printf("Error updating DNS record %s for zone %s: %v", record.Name, zoneID, err)
		} else {
			log.Printf("Successfully updated DNS record '%s' (ID: %s) to IP: %s in zone '%s'.", record.Name, record.ID, localIP, zoneID)
			fmt.Printf("Successfully updated DNS record '%s' to IP: %s\n", record.Name, localIP)
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

	// Check for HTTP errors including authentication failures
	if err := checkCloudflareResponse(response, body, "getting DNS records"); err != nil {
		return DNS_REC_LIST{}, err
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
			// log.Println(record.String())
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

	// Check for HTTP errors including authentication failures
	if err := checkCloudflareResponse(res, body, "updating DNS record"); err != nil {
		return err
	}

	log.Printf("Update response for %s: %s", record.Name, string(body))
	// log.Println("HTTP Response:", res)
	return nil
}
