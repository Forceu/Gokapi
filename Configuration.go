package main

/**
Loading and saving of the persistent configuration
*/

import (
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"os"
	"strings"
)


// Name of the config dir that will be created
const configDir = "config"

// Name of the config file where the configuration is written to
const configFile = "config.json"

// Full path of configDir and configFile
const configPath = configDir + "/" + configFile

// Name of the data dir that will be created
const dataDir = "data"

// Global object containing the configuration
var globalConfig Configuration

// Struct that contains the global configuration
type Configuration struct {
	Port             string              `json:"Port"`
	AdminName        string              `json:"AdminName"`
	AdminPassword    string              `json:"AdminPassword"`
	ServerUrl        string              `json:"ServerUrl"`
	DefaultDownloads int                 `json:"DefaultDownloads"`
	DefaultExpiry    int                 `json:"DefaultExpiry"`
	DefaultPassword  string              `json:"DefaultPassword"`
	RedirectUrl      string              `json:"RedirectUrl"`
	Sessions         map[string]Session  `json:"Sessions"`
	Files            map[string]FileList `json:"Files"`
}

// Loads the configuration or creates the folder structure and a default configuration
func loadConfig() {
	createConfigDir()
	if !fileExists(configPath) {
		generateDefaultConfig()
	}
	file, err := os.Open(configPath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	globalConfig = Configuration{}
	err = decoder.Decode(&globalConfig)
	if err != nil {
		log.Fatal(err)
	}
}

// Creates a default configuration and asks for items like username/password etc.
func generateDefaultConfig() {
	fmt.Println("First start, creating new admin account")
	username := askForUsername()
	password := askForPassword()
	url := askForUrl()
	redirect := askForRedirect()
	localOnly := askForLocalOnly()
	port := "127.0.0.1:53842"
	if !localOnly {
		port = "0.0.0.0:53842"
	}

	globalConfig = Configuration{
		Port:             port,
		AdminName:        username,
		AdminPassword:    hashPassword(password, SALT_PW_ADMIN),
		ServerUrl:        url,
		DefaultDownloads: 1,
		DefaultExpiry:    14,
		RedirectUrl:      redirect,
		Files:            make(map[string]FileList),
		Sessions:         make(map[string]Session),
	}
	saveConfig()
}

// Saves the configuration as a json file
func saveConfig() {
	file, err := os.OpenFile(configPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Error reading configuration:", err)
		os.Exit(1)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	err = encoder.Encode(&globalConfig)
	if err != nil {
		fmt.Println("Error writing configuration:", err)
		os.Exit(1)
	}
}

// Asks for username and returns input as string if valid
func askForUsername() string {
	fmt.Print("Username: ")
	username := readLine()
	if len(username) >= 4 {
		return username
	}
	fmt.Println("Username needs to be at least 4 characters long")
	return askForUsername()
}

// Asks for password and returns input as string if valid
func askForPassword() string {
	fmt.Print("Password: ")
	password1, err := terminal.ReadPassword(0)
	check(err)
	if len(password1) < 6 {
		fmt.Println("\nPassword needs to be at least 6 characters long")
		return askForPassword()
	}
	fmt.Print("\nPassword (repeat): ")
	password2, err := terminal.ReadPassword(0)
	check(err)
	if string(password1) != string(password2) {
		fmt.Println("\nPasswords dont match")
		return askForPassword()
	}
	fmt.Println()
	return string(password1)
}

// Asks if the server shall be bound to 127.0.0.1 and returns result as bool
func askForLocalOnly() bool {
	if IS_DOCKER != "false" {
		return false
	}
	fmt.Print("Bind port to localhost only? [Y/n]: ")
	input := strings.ToLower(readLine())
	return input != "n"
}

// Asks for server URL and returns input as string if valid
func askForUrl() string {
	fmt.Print("Server URL [eg. https://gokapi.url/]: ")
	url := readLine()
	if !isValidUrl(url) {
		return askForUrl()
	}
	if !strings.HasSuffix(url, "/") {
		url = url + "/"
	}
	return url
}

// Asks for redirect URL and returns input as string if valid
func askForRedirect() string {
	fmt.Print("URL that the index gets redirected to [eg. https://yourcompany.com/]: ")
	url := readLine()
	if url == "" {
		return "https://github.com/Forceu/Gokapi/"
	}
	if !isValidUrl(url) {
		return askForUrl()
	}
	if !strings.HasSuffix(url, "/") {
		url = url + "/"
	}
	return url
}

// Returns true if URL starts with http:// or https://
func isValidUrl(url string) bool {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		fmt.Println("URL needs to start with http:// or https://")
		return false
	}
	return true
}
