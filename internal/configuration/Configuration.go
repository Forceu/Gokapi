package configuration

/**
Loading and saving of the persistent configuration
*/

import (
	"Gokapi/internal/environment"
	"Gokapi/internal/helper"
	"Gokapi/internal/models"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/term"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

// Default port that the program runs on
const defaultPort = "53842"

// Min length of admin password in characters
const minLengthPassword = 6

// Environment is an object containing the environment variables
var Environment environment.Environment

// ServerSettings is an object containing the server configuration
var serverSettings Configuration

// Version of the configuration structure. Used for upgrading
const currentConfigVersion = 6

// For locking this object to prevent race conditions
var mutex sync.Mutex

// Configuration is a struct that contains the global configuration
type Configuration struct {
	Port             string                           `json:"Port"`
	AdminName        string                           `json:"AdminName"`
	AdminPassword    string                           `json:"AdminPassword"`
	ServerUrl        string                           `json:"ServerUrl"`
	DefaultDownloads int                              `json:"DefaultDownloads"`
	DefaultExpiry    int                              `json:"DefaultExpiry"`
	DefaultPassword  string                           `json:"DefaultPassword"`
	RedirectUrl      string                           `json:"RedirectUrl"`
	Sessions         map[string]models.Session        `json:"Sessions"`
	Files            map[string]models.File           `json:"Files"`
	Hotlinks         map[string]models.Hotlink        `json:"Hotlinks"`
	DownloadStatus   map[string]models.DownloadStatus `json:"DownloadStatus"`
	ApiKeys          map[string]models.ApiKey         `json:"ApiKeys"`
	ConfigVersion    int                              `json:"ConfigVersion"`
	SaltAdmin        string                           `json:"SaltAdmin"`
	SaltFiles        string                           `json:"SaltFiles"`
	LengthId         int                              `json:"LengthId"`
	DataDir          string                           `json:"DataDir"`
	AwsBucket        string                           `json:"AwsBucket"`
	MaxMemory        int                              `json:"MaxMemory"`
}

// Load loads the configuration or creates the folder structure and a default configuration
func Load() {
	Environment = environment.New()
	helper.CreateDir(Environment.ConfigDir)
	if !helper.FileExists(Environment.ConfigPath) {
		generateDefaultConfig()
	}
	file, err := os.Open(Environment.ConfigPath)
	helper.Check(err)
	defer file.Close()
	decoder := json.NewDecoder(file)
	serverSettings = Configuration{}
	err = decoder.Decode(&serverSettings)
	helper.Check(err)
	updateConfig()
	serverSettings.AwsBucket = Environment.AwsBucketName
	serverSettings.MaxMemory = Environment.MaxMemory
	helper.CreateDir(serverSettings.DataDir)
}

// Lock locks configuration to prevent race conditions (blocking)
func Lock() {
	mutex.Lock()
}

// ReleaseAndSave unlocks and saves the configuration
func ReleaseAndSave() {
	save()
	mutex.Unlock()
}

// Release unlocks the configuration
func Release() {
	mutex.Unlock()
}

// GetServerSettings locks the settings returns a pointer to the configuration
// Release needs to be called when finished with the operation!
func GetServerSettings() *Configuration {
	mutex.Lock()
	return &serverSettings
}

// Upgrades the ServerSettings if saved with a previous version
func updateConfig() {
	// < v1.1.2
	if serverSettings.ConfigVersion < 3 {
		serverSettings.SaltAdmin = "eefwkjqweduiotbrkl##$2342brerlk2321"
		serverSettings.SaltFiles = "P1UI5sRNDwuBgOvOYhNsmucZ2pqo4KEvOoqqbpdu"
		serverSettings.LengthId = 15
		serverSettings.DataDir = Environment.DataDir
	}
	// < v1.1.3
	if serverSettings.ConfigVersion < 4 {
		serverSettings.Hotlinks = make(map[string]models.Hotlink)
	}
	// < v1.1.4
	if serverSettings.ConfigVersion < 5 {
		serverSettings.LengthId = 15
		serverSettings.DownloadStatus = make(map[string]models.DownloadStatus)
		for _, file := range serverSettings.Files {
			file.ContentType = "application/octet-stream"
			serverSettings.Files[file.Id] = file
		}
	}
	// < v1.2.0
	if serverSettings.ConfigVersion < 6 {
		serverSettings.ApiKeys = make(map[string]models.ApiKey)
	}
	if serverSettings.ConfigVersion < currentConfigVersion {
		fmt.Println("Successfully upgraded database")
		serverSettings.ConfigVersion = currentConfigVersion
		save()
	}
}

// Creates a default configuration and asks for items like username/password etc.
func generateDefaultConfig() {
	fmt.Println("First start, creating new admin account")
	saltAdmin := Environment.SaltAdmin
	if saltAdmin == "" {
		saltAdmin = helper.GenerateRandomString(30)
	}
	serverSettings = Configuration{
		SaltAdmin: saltAdmin,
	}
	username := askForUsername(1)
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

	serverSettings = Configuration{
		Port:             bindAddress,
		AdminName:        username,
		AdminPassword:    HashPassword(password, false),
		ServerUrl:        url,
		DefaultDownloads: 1,
		DefaultExpiry:    14,
		RedirectUrl:      redirect,
		Files:            make(map[string]models.File),
		Sessions:         make(map[string]models.Session),
		Hotlinks:         make(map[string]models.Hotlink),
		ApiKeys:          make(map[string]models.ApiKey),
		DownloadStatus:   make(map[string]models.DownloadStatus),
		ConfigVersion:    currentConfigVersion,
		SaltAdmin:        saltAdmin,
		SaltFiles:        saltFiles,
		DataDir:          Environment.DataDir,
		LengthId:         Environment.LengthId,
	}
	save()
}

// Save the configuration as a json file
func save() {
	file, err := os.OpenFile(Environment.ConfigPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Error reading configuration:", err)
		os.Exit(1)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	err = encoder.Encode(&serverSettings)
	if err != nil {
		fmt.Println("Error writing configuration:", err)
		os.Exit(1)
	}
}

// Asks for username or loads it from env and returns input as string if valid
func askForUsername(try int) string {
	if try > 5 {
		fmt.Println("Too many invalid entries! If you are running the setup with Docker, make sure to start the container with the -it flag.")
		os.Exit(1)
	}
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
	return askForUsername(try + 1)
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
	password1 := readPassword()
	if utf8.RuneCountInString(password1) < minLengthPassword {
		fmt.Println("\nPassword needs to be at least " + strconv.Itoa(minLengthPassword) + " characters long")
		return askForPassword()
	}
	fmt.Print("\nPassword (repeat): ")
	password2 := readPassword()
	if password1 != password2 {
		fmt.Println("\nPasswords dont match")
		return askForPassword()
	}
	fmt.Println()
	return password1
}

func readPassword() string {
	if runtime.GOOS != "windows" {
		pw, err := term.ReadPassword(0)
		if err == nil {
			return string(pw)
		}
	}
	return helper.ReadLine()
}

// Asks if the server shall be bound to 127.0.0.1 or loads it from env and returns result as bool
// Always returns environment.IsFalse for Docker environment
func askForLocalOnly() string {
	if environment.IsDocker != "false" {
		return environment.IsFalse
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
	serverSettings.AdminPassword = HashPassword(askForPassword(), false)
	// Log out all sessions
	serverSettings.Sessions = make(map[string]models.Session)
	save()
}

// HashPassword hashes a string with SHA256 and a salt
func HashPassword(password string, useFileSalt bool) string {
	if password == "" {
		return ""
	}
	salt := serverSettings.SaltAdmin
	if useFileSalt {
		salt = serverSettings.SaltFiles
	}
	bytes := []byte(password + salt)
	hash := sha1.New()
	hash.Write(bytes)
	return hex.EncodeToString(hash.Sum(nil))
}

// GetLengthId returns the length of the file IDs to be generated
func GetLengthId() int {
	return serverSettings.LengthId
}
