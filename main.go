package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"example.com/v1/CF/get_config"
)
func (d DNS_REC) String() string{
	return fmt.Sprintf("Name: %v\nType: %v\nProxied: %v\nComment: %v\nTags: %v\nTTL: %v\nContent: %v\n",  d.name, d.typpe, d.proxied, d.comment, d.tags, d.ttl, d.content)
}

type DNS_REC struct {
	zone_id interface{}
	id      string
	name    interface{}
	typpe   interface{}
	proxied interface{}
	comment interface{}
	tags    interface{}
	ttl     interface{}
	content string
}

type DNS_REC_LIST struct {
	list []DNS_REC
}

func main() {
	email, key := get_config.Get_account_info()

	var account_id string = ""


	logPath := os.Getenv("LOGPATH")
	print(logPath)
	if logPath == "" {
		logPath = "./log.txt"
	}
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)



	var header = http.Header{}
	header.Add("X-Auth-Email", email)
	header.Add("Content-Type", "application/json")
	header.Add("X-Auth-Key", key)

	print(header)
	log.Println(header)

	// get Zone IDs

	var zones = http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "https",
			Host:   "api.cloudflare.com",
			Path:   "/client/v4/zones",
		},
		Header: header,
	}

	netClient := &http.Client{}
	response, err := netClient.Do(&zones)
	if err != nil {
		log.Fatal(err)
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	var data struct {
		Result []struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	
	log.Print(data.Result)

	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Fatal("No results found")
	}

	if len(data.Result) > 0 {
		firstID := data.Result[0].ID
		// fmt.Println("First ID:", firstID)
		account_id = firstID
	} else {
		log.Fatal("No results found")
	}

// Get DNS records

	var list = http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "https",
			Host:   "api.cloudflare.com",
			Path:   "/client/v4/zones/" + account_id + "/dns_records",
		},
		Header: header,
	}

	// netClient := &http.Client{}
	response, err = netClient.Do(&list)
	if err != nil {
		log.Fatal(err)
	}
	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	log.Print("\n\n\n")
	fmt.Println(string(body))
	println("\n\n\n")
	// Convert body to JSON
	var data2 interface{}
	err = json.Unmarshal(body, &data2)
	if err != nil {
		log.Print(err)
	}

	log.Println(data.Result)

	// Print the JSON data
	// fmt.Println(data)

	dns_records := DNS_REC_LIST{}

	a_records := data2.(map[string]interface{})["result"].([]interface{})

	for _, record := range a_records {
		record := record.(map[string]interface{})

		rec := DNS_REC{
			id:      record["id"].(string),
			zone_id: record["zone_id"],
			name:    record["name"],
			typpe:   record["type"],
			proxied: record["proxied"],
			comment: record["comment"],

			tags: record["tags"],

			content: record["content"].(string),
			ttl:     record["ttl"],
		}
		dns_records.list = append(dns_records.list, rec)
	}
	for _, rec := range dns_records.list {
		fmt.Println(rec.String())
	}

	A_rec := DNS_REC_LIST{}
	for _, record := range dns_records.list {
		if record.typpe == "A" {
			log.Println(record.content)
			A_rec.list = append(A_rec.list, record)

		}
	}
	// Make API call to get IP address
	test, err := http.Get("https://api.ipify.org")
	if err != nil {
		log.Fatal(err)
	}
	defer test.Body.Close()

	// Read the response body
	body, err = ioutil.ReadAll(test.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Print the response body
	var local_ip string = string(body)
	log.Println("local IP: ", local_ip)

// Check if the IP address has changed if not exit

	if local_ip == A_rec.list[0].content {
		log.Println("\033[0;31m IP address has not changed")
		logFile.Close()
		os.Exit(0)
	}


	// Update the DNS  A records

	for _, record := range A_rec.list {
		if record.comment == nil {
			record.comment = ""
		}
		url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%v/dns_records/%v", record.zone_id, record.id)

		payload := strings.NewReader(fmt.Sprintf(`{
			"content": "%v",
			"name": "%v",
			"proxied": %v,
			"type": "%v",
			"comment": "%v",
			"tags": [],
			"ttl": %v
		}`, local_ip, record.name, record.proxied, record.typpe, record.comment, record.ttl))

		req, err := http.NewRequest("PATCH", url, payload)
		if err != nil {
			log.Fatal(err)
		}

		req.Header = header

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Fatal(err)
		}

		log.Println(string(body))
		log.Println(res)
	result := res
		// Print the result to the log file
		log.Println("payload:", payload, "\nresults:", result)

	}
	logFile.Close()

}
