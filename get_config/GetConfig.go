package get_config

import (
	"errors"
	"log"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Email string `yaml:"X-Auth-Email"`
	Token string `yaml:"X-Auth-Key"`
}

func Get_account_info() (string, string) {
	_, err := os.Stat("/.dockerenv")
	var data []byte
	if err != nil {
		println("Not in Docker")
		data, err = os.ReadFile("./CONFIG/config.yaml")	
	
	} else {
		println("In Docker")
		data, err = os.ReadFile("/config/config.yaml")
	}



	// Unmarshal the YAML data into a struct
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	// Access the values from the struct
	return config.Email, config.Token
}


// create new A record

type Create struct {
	Content string   `yaml:"content"`
	Name    string   `yaml:"name"`
	Typpe   string   `yaml:"type"`
	Proxied bool     `yaml:"proxied"`
	Comment string   `yaml:"comment"`
	Tags    []string `yaml:"tags"`
	Ttl     int      `yaml:"ttl"`
}

func (c Create) String() string {
	return "Name: " + c.Name + "\nType: " + c.Typpe + "\nProxied: " + strconv.FormatBool(c.Proxied) + "\nComment: " + c.Comment + "\nTags: " + strings.Join(c.Tags, ", ") + "\nTTL: " + strconv.Itoa(c.Ttl) + "\nContent: " + c.Content
}

func Read_yaml() (*Create, error) {
	_, err := os.Stat("/.dockerenv")
	var data []byte
	if err != nil {
		println("Not in Docker")
		data, err = os.ReadFile("./CONFIG/create.yaml")	
		if err != nil {
			log.Fatalf("Failed to read file: %v", err)
		}
	
	} else {
		println("In Docker")
		data, err = os.ReadFile("/config/create.yaml")
		if err != nil {
			log.Fatalf("Failed to read file: %v", err)
		}
	}
	// Unmarshal the YAML data into a struct
	var create Create
	err = yaml.Unmarshal(data, &create)
	if err != nil {
		log.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	if create.Content == ""  {
		return nil, errors.New("fileds must not be blank")
	}
	// Access the values from the struct
	return &create, nil
}