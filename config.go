package main

import (
	"encoding/json"
	"io/ioutil"
)

// SMTPServer represents the SMTP configuration details
type SMTPServer struct {
	Host     string `json:"host"`
	Addr     string `json:"addr"`
	User     string `json:"user"`
	Password string `json:"pass"`
}

// Config represents the configuration information.
type Config struct {
	Url    string     `json:"url"`
	Addr   string     `json:"addr"`
	Email  string     `json:"email"`
	SMTP   SMTPServer `json:"smtp"`
	DBAddr string     `json:"dbAddr"`
}

// loadConfig reads the config file and returns the parsed Config
// If an error is found, the function will panic
func loadConfig() *Config {
	var conf Config
	// Get the config file
	config_file, err := ioutil.ReadFile("./config.json")
	if err != nil {
		Logger.Fatalf("Error loading config file: %v\n", err)
	}
	if err = json.Unmarshal(config_file, &conf); err != nil {
		Logger.Fatalf("Error loading config file: %v\n", err)
	}
	return &conf
}
