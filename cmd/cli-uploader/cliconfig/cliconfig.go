package cliconfig

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/forceu/gokapi/cmd/cli-uploader/cliapi"
	"github.com/forceu/gokapi/cmd/cli-uploader/cliconstants"
	"github.com/forceu/gokapi/cmd/cli-uploader/cliflags"
	"github.com/forceu/gokapi/internal/helper"
)

type configFile struct {
	Url    string `json:"Url"`
	Apikey string `json:"Apikey"`
	E2ekey []byte `json:"E2Ekey"`
}

var configPathLocation string

// CreateLogin creates a login for the CLI.
// It will ask the user for the URL and API key.
// It will then test the connection and download the configuration.
// If the configuration is valid, the login information will be saved.
func CreateLogin() {
	fmt.Print("Gokapi URL: ")
	url := helper.ReadLine()
	if (!strings.HasPrefix(url, "http://")) && (!strings.HasPrefix(url, "https://")) {
		fmt.Println("ERROR: URL must start with http:// or https://")
		os.Exit(1)
	}

	url = strings.TrimSuffix(url, "/admin")
	if strings.HasPrefix(url, "http://") {
		fmt.Println("WARNING: This URL uses an insecure connection. All data, including your API key, will be sent in plain text. This is not recommended for production use.")
	}
	fmt.Print("API key: ")
	apikey := helper.ReadLine()
	if len(apikey) < 3 {
		fmt.Println("ERROR: Invalid API key")
		os.Exit(1)
	}
	fmt.Println("")
	fmt.Print("Testing connection...")
	cliapi.Init(url, apikey, []byte{})
	vstr, vint, err := cliapi.GetVersion()
	if err != nil {
		fmt.Println()
		if errors.Is(cliapi.ErrUnauthorised, err) {
			fmt.Println("ERROR: Unauthorised API key")
			os.Exit(1)
		}
		if errors.Is(cliapi.ErrNotFound, err) {
			fmt.Println("ERROR: API not found")
			fmt.Println("The provided URL does not respond to API calls as expected. You most likely entered an incorrect URL.")
			os.Exit(1)
		}
		if errors.Is(cliapi.ErrInvalidRequest, err) {
			fmt.Println("ERROR: API does not support Gokapi CLI")
			fmt.Println("This is most likely caused by an old Gokapi version. Please make sure that your Gokapi instance is running v2.1.0 or newer.")
			os.Exit(1)
		}
		fmt.Println(err)
		os.Exit(1)
	}

	if vint < cliconstants.MinGokapiVersionInt {
		fmt.Println("\nERROR: Gokapi version must be at least " + cliconstants.MinGokapiVersionStr)
		fmt.Println("Your version is " + vstr)
		os.Exit(1)
	}
	fmt.Print("OK\nDownloading configuration...")

	_, _, isE2E, err := cliapi.GetConfig()
	if err != nil {
		fmt.Println("FAIL")
		if errors.Is(cliapi.ErrUnauthorised, err) {
			fmt.Println("ERROR: API key does not have the permission to upload new files.")
		} else {
			fmt.Println(err)
		}
		os.Exit(1)
	}
	fmt.Println("OK")
	var e2ekey []byte
	if isE2E {
		fmt.Print("End-to-end encryption key: ")
		e2ekeyString := helper.ReadLine()
		e2ekey, err = base64.StdEncoding.DecodeString(e2ekeyString)
		if err != nil {
			fmt.Println("ERROR: Invalid end-to-end encryption key")
			os.Exit(1)
		}
		cliapi.Init(url, apikey, e2ekey)
		_, err = cliapi.GetE2eInfo()
		if err != nil {
			if errors.Is(cliapi.ErrE2eKeyIncorrect, err) {
				fmt.Println("ERROR: Incorrect end-to-end encryption key")
			} else {
				fmt.Println(err)
			}
			os.Exit(1)
		}
		// TODO check if key has not been generated yet
		// TODO warn user not to upload e2e simultaneously
	}

	err = save(url, apikey, e2ekey)
	if err != nil {
		fmt.Println("ERROR: Could not save login information")
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Login successful")
	fmt.Print("Credentials saved to: ")
	fmt.Println(getConfigPath())
}

// getConfigPath returns the path to the configuration file.
// It first checks if the user has specified a custom location using the --config flag.
// If not, it checks the default locations.
// If no existing file is found, it returns the default location.
//
// Caches the result! For tests, make sure to clear configPathLocation first
func getConfigPath() string {
	if configPathLocation != "" {
		return configPathLocation
	}
	configPath, isDefault := cliflags.GetConfigLocation()
	if !isDefault {
		configPathLocation = configPath[0]
		return configPathLocation
	}
	for _, location := range configPath {
		exists, err := helper.FileExists(location)
		if err != nil {
			continue
		}
		if exists {
			configPathLocation = location
			break
		}
	}
	// If no existing file was found, use the first default location
	if configPathLocation == "" {
		configPathLocation = configPath[0]
	}
	return configPathLocation
}

func save(url, apikey string, e2ekey []byte) error {
	configData := configFile{
		Url:    url,
		Apikey: apikey,
		E2ekey: e2ekey,
	}

	jsonData, err := json.Marshal(configData)
	if err != nil {
		return err
	}

	return os.WriteFile(getConfigPath(), jsonData, 0600)
}

// Load initialises the configuration by reading login information from a file and setting up CLI API parameters.
// Verifies the existence of the configuration file and validates its integrity, terminating on errors.
func Load() {
	exists, err := helper.FileExists(getConfigPath())
	helper.Check(err)
	if !exists {
		fmt.Println("ERROR: No login information found")
		fmt.Println("Please run 'gokapi-cli login' to create a login")
		os.Exit(1)
	}
	data, err := os.ReadFile(getConfigPath())
	helper.Check(err)

	var config configFile
	err = json.Unmarshal(data, &config)
	if err != nil {
		fmt.Println("ERROR: Could not read login information")
		os.Exit(1)
	}

	cliapi.Init(config.Url, config.Apikey, config.E2ekey)
}

// Delete deletes the login information file.
// It will return an error if the file exists but could not be deleted.
func Delete() error {
	exists, err := helper.FileExists(getConfigPath())
	if err != nil {
		fmt.Println("ERROR: Could not check if login information exists")
		fmt.Println(err)
		os.Exit(1)
	}
	if !exists {
		return nil
	}
	return os.Remove(getConfigPath())
}
