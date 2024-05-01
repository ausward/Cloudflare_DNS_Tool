package get_config

import (
	"log"
	"os"

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
