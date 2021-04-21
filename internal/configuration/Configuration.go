package configuration

/**
Loading and saving of the persistent configuration
*/

import (
	"Gokapi/internal/environment"
	"Gokapi/internal/helper"
	"Gokapi/internal/storage/filestructure"
	"Gokapi/internal/webserver/sessionstructure"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Default port that the program runs on
const defaultPort = "53842"

// Min length of admin password in characters
const minLengthPassword = 6

// Environment is an object containing the environment variables
var Environment environment.Environment

// ServerSettings is an object containing the server configuration
var ServerSettings Configuration

// Version of the configuration structure. Used for upgrading
const currentConfigVersion = 5

// Configuration is a struct that contains the global configuration
type Configuration struct {
	Port             string                                  `json:"Port"`
	AdminName        string                                  `json:"AdminName"`
	AdminPassword    string                                  `json:"AdminPassword"`
	ServerUrl        string                                  `json:"ServerUrl"`
	DefaultDownloads int                                     `json:"DefaultDownloads"`
	DefaultExpiry    int                                     `json:"DefaultExpiry"`
	DefaultPassword  string                                  `json:"DefaultPassword"`
	RedirectUrl      string                                  `json:"RedirectUrl"`
	Sessions         map[string]sessionstructure.Session     `json:"Sessions"`
	Files            map[string]filestructure.File           `json:"Files"`
	Hotlinks         map[string]filestructure.Hotlink        `json:"Hotlinks"`
	DownloadStatus   map[string]filestructure.DownloadStatus `json:"DownloadStatus"`
	ConfigVersion    int                                     `json:"ConfigVersion"`
	SaltAdmin        string                                  `json:"SaltAdmin"`
	SaltFiles        string                                  `json:"SaltFiles"`
	LengthId         int                                     `json:"LengthId"`
	DataDir          string                                  `json:"DataDir"`
}

// Load loads the configuration or creates the folder structure and a default configuration
func Load() {
	Environment = environment.New()
	helper.CreateDir(Environment.ConfigDir)
	if !helper.FileExists(Environment.ConfigPath) {
		generateDefaultConfig()
	}
	file, err := os.Open(Environment.ConfigPath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	ServerSettings = Configuration{}
	err = decoder.Decode(&ServerSettings)
	if err != nil {
		log.Fatal(err)
	}
	updateConfig()
	helper.CreateDir(ServerSettings.DataDir)
}

// Upgrades the ServerSettings if saved with a previous version
func updateConfig() {
	// < v1.1.2
	if ServerSettings.ConfigVersion < 3 {
		ServerSettings.SaltAdmin = "eefwkjqweduiotbrkl##$2342brerlk2321"
		ServerSettings.SaltFiles = "P1UI5sRNDwuBgOvOYhNsmucZ2pqo4KEvOoqqbpdu"
		ServerSettings.LengthId = 15
		ServerSettings.DataDir = Environment.DataDir
	}
	// < v1.1.3
	if ServerSettings.ConfigVersion < 4 {
		ServerSettings.Hotlinks = make(map[string]filestructure.Hotlink)
	}

	// < v1.1.4
	if ServerSettings.ConfigVersion < 5 {
		ServerSettings.LengthId = 15
		ServerSettings.DownloadStatus = make(map[string]filestructure.DownloadStatus)
		for _, file := range ServerSettings.Files {
			file.ContentType = "application/octet-stream"
			ServerSettings.Files[file.Id] = file
		}
	}

	if ServerSettings.ConfigVersion < currentConfigVersion {
		fmt.Println("Successfully upgraded database")
		ServerSettings.ConfigVersion = currentConfigVersion
		Save()
	}
}

// Creates a default configuration and asks for items like username/password etc.
func generateDefaultConfig() {
	fmt.Println("First start, creating new admin account")
	saltAdmin := Environment.SaltAdmin
	if saltAdmin == "" {
		saltAdmin = helper.GenerateRandomString(30)
	}
	ServerSettings = Configuration{
		SaltAdmin: saltAdmin,
	}
	username := askForUsername()
	password := askForPassword()
	port := askForPort()
	url := askForUrl(port)
	redirect := askForRedirect()
	localOnly := askForLocalOnly()
	bindAddress := "127.0.0.1:" + port
	if localOnly == environment.IsFalse {
		bindAddress = ":" + port
	}
	saltFiles := Environment.SaltFiles
	if saltFiles == "" {
		saltFiles = helper.GenerateRandomString(30)
	}

	ServerSettings = Configuration{
		Port:             bindAddress,
		AdminName:        username,
		AdminPassword:    HashPassword(password, false),
		ServerUrl:        url,
		DefaultDownloads: 1,
		DefaultExpiry:    14,
		RedirectUrl:      redirect,
		Files:            make(map[string]filestructure.File),
		Sessions:         make(map[string]sessionstructure.Session),
		Hotlinks:         make(map[string]filestructure.Hotlink),
		ConfigVersion:    currentConfigVersion,
		SaltAdmin:        saltAdmin,
		SaltFiles:        saltFiles,
		DataDir:          Environment.DataDir,
		LengthId:         Environment.LengthId,
	}
	Save()
}

// Save the configuration as a json file
func Save() {
	file, err := os.OpenFile(Environment.ConfigPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Error reading configuration:", err)
		os.Exit(1)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	err = encoder.Encode(&ServerSettings)
	if err != nil {
		fmt.Println("Error writing configuration:", err)
		os.Exit(1)
	}
}

// Asks for username or loads it from env and returns input as string if valid
func askForUsername() string {
	fmt.Print("Username: ")
	envUsername := Environment.AdminName
	if envUsername != "" {
		fmt.Println(envUsername)
		return envUsername
	}
	username := helper.ReadLine()
	if len(username) >= 4 {
		return username
	}
	fmt.Println("Username needs to be at least 4 characters long")
	return askForUsername()
}

// Asks for password or loads it from env and returns input as string if valid
func askForPassword() string {
	fmt.Print("Password: ")
	envPassword := Environment.AdminPassword
	if envPassword != "" {
		fmt.Println("*******************")
		if utf8.RuneCountInString(envPassword) < minLengthPassword {
			fmt.Println("\nPassword needs to be at least " + strconv.Itoa(minLengthPassword) + " characters long")
			os.Exit(1)
		}
		return envPassword
	}
	password1, err := terminal.ReadPassword(0)
	helper.Check(err)
	if utf8.RuneCountInString(string(password1)) < minLengthPassword {
		fmt.Println("\nPassword needs to be at least " + strconv.Itoa(minLengthPassword) + " characters long")
		return askForPassword()
	}
	fmt.Print("\nPassword (repeat): ")
	password2, err := terminal.ReadPassword(0)
	helper.Check(err)
	if string(password1) != string(password2) {
		fmt.Println("\nPasswords dont match")
		return askForPassword()
	}
	fmt.Println()
	return string(password1)
}

// Asks if the server shall be bound to 127.0.0.1 or loads it from env and returns result as bool
func askForLocalOnly() string {
	if environment.IsDocker != "false" {
		return environment.IsTrue
	}
	fmt.Print("Bind port to localhost only? [Y/n]: ")
	envLocalhost := Environment.WebserverLocalhost
	if envLocalhost != "" {
		fmt.Println(envLocalhost)
		return envLocalhost
	}
	input := strings.ToLower(helper.ReadLine())
	if input == "n" || input == "no" {
		return environment.IsFalse
	}
	return environment.IsTrue
}

// Asks for server port or loads it from env and returns input as string if valid
func askForPort() string {
	fmt.Print("Server Port [" + defaultPort + "]: ")
	envPort := Environment.WebserverPort
	if envPort != "" {
		fmt.Println(envPort)
		return envPort
	}
	port := helper.ReadLine()
	if port == "" {
		return defaultPort
	}
	if !isValidPortNumber(port) {
		return askForPort()
	}
	return port
}

// Asks for server URL or loads it from env and returns input as string if valid
func askForUrl(port string) string {
	fmt.Print("External Server URL [http://127.0.0.1:" + port + "/]: ")
	envUrl := Environment.ExternalUrl
	if envUrl != "" {
		fmt.Println(envUrl)
		if !isValidUrl(envUrl) {
			os.Exit(1)
		}
		return addTrailingSlash(envUrl)
	}
	url := helper.ReadLine()
	if url == "" {
		return "http://127.0.0.1:" + port + "/"
	}
	if !isValidUrl(url) {
		return askForUrl(port)
	}
	return addTrailingSlash(url)
}

// Asks for redirect URL or loads it from env and returns input as string if valid
func askForRedirect() string {
	fmt.Print("URL that the index gets redirected to [https://github.com/Forceu/Gokapi/]: ")
	envRedirect := Environment.RedirectUrl
	if envRedirect != "" {
		fmt.Println(envRedirect)
		if !isValidUrl(envRedirect) {
			os.Exit(1)
		}
		return envRedirect
	}
	url := helper.ReadLine()
	if url == "" {
		return "https://github.com/Forceu/Gokapi/"
	}
	if !isValidUrl(url) {
		return askForRedirect()
	}
	return url
}

// Returns true if URL starts with http:// or https://
func isValidUrl(url string) bool {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		fmt.Println("URL needs to start with http:// or https://")
		return false
	}
	postfix := strings.Replace(url, "http://", "", -1)
	postfix = strings.Replace(postfix, "https://", "", -1)
	return len(postfix) > 0
}

// Checks if the string is a valid port number
func isValidPortNumber(input string) bool {
	port, err := strconv.Atoi(input)
	if err != nil {
		fmt.Println("Input needs to be a number")
		return false
	}
	if port < 0 || port > 65353 {
		fmt.Println("Port needs to be between 0-65353")
		return false
	}
	return true
}

// Adds a / character to the end of an URL if it does not exist
func addTrailingSlash(url string) string {
	if !strings.HasSuffix(url, "/") {
		return url + "/"
	}
	return url
}

// DisplayPasswordReset shows a password prompt in the CLI and saves the new password
func DisplayPasswordReset() {
	ServerSettings.AdminPassword = HashPassword(askForPassword(), false)
	Save()
}

// HashPassword hashes a string with SHA256 and a salt
func HashPassword(password string, useFileSalt bool) string {
	if password == "" {
		return ""
	}
	salt := ServerSettings.SaltAdmin
	if useFileSalt {
		salt = ServerSettings.SaltFiles
	}
	bytes := []byte(password + salt)
	hash := sha1.New()
	hash.Write(bytes)
	return hex.EncodeToString(hash.Sum(nil))
}
