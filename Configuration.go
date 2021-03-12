package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"os"
	"strings"
)

const configDir = "config"
const configFile = "config.json"
const configPath = configDir + "/" + configFile

var globalConfig Configuration

type Configuration struct {
	Port             string              `json:"Port"`
	AdminName        string              `json:"AdminName"`
	AdminPassword    string              `json:"AdminPassword"`
	ServerUrl        string              `json:"ServerUrl"`
	DefaultDownloads int                 `json:"DefaultDownloads"`
	DefaultExpiry    int                 `json:"DefaultExpiry"`
	RedirectUrl      string              `json:"RedirectUrl"`
	Sessions         map[string]Session  `json:"Sessions"`
	Files            map[string]FileList `json:"Files"`
}

func (f *FileList) toJsonResult() string {
	result := Result{
		Result:   "OK",
		Url:      globalConfig.ServerUrl + "d?id=",
		FileInfo: f,
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		fmt.Println(err)
		return "{\"Result\":\"error\",\"ErrorMessage\":\"" + err.Error() + "\"}"
	}
	return string(bytes)
}

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
		AdminPassword:    hashPassword(password),
		ServerUrl:        url,
		DefaultDownloads: 1,
		DefaultExpiry:    14,
		RedirectUrl:      redirect,
		Files:            make(map[string]FileList),
		Sessions:         make(map[string]Session),
	}
	saveConfig()
}

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

func askForUsername() string {
	fmt.Print("Username: ")
	username := readLine()
	if len(username) >= 4 {
		return username
	}
	fmt.Println("Username needs to be at least 4 characters long")
	return askForUsername()
}

func askForLocalOnly() bool {
	if isDocker() {
		return false
	}
	fmt.Print("Bind port to localhost only? [Y/n]: ")
	input := strings.ToLower(readLine())
	return input != "n"
}

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

func askForRedirect() string {
	fmt.Print("URL that the index gets redirected to [eg. https://yourcompany.com/]: ")
	url := readLine()
	if !isValidUrl(url) {
		return askForUrl()
	}
	if !strings.HasSuffix(url, "/") {
		url = url + "/"
	}
	return url
}

func isValidUrl(url string) bool {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		fmt.Println("URL needs to start with http:// or https://")
		return false
	}
	return true
}
func readLine() string {
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	return strings.Replace(text, "\n", "", -1)
}

type Result struct {
	Result   string    `json:"Result"`
	FileInfo *FileList `json:"FileInfo"`
	Url      string    `json:"Url"`
}

func createConfigDir() {
	if !folderExists(configDir) {
		err := os.Mkdir(configDir, 0770)
		check(err)
	}
}

func isDocker() bool {
	return fileExists(".isdocker")
}
