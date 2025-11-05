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

	configPath, _ := cliflags.GetConfigLocation()
	return os.WriteFile(configPath, jsonData, 0600)
}

// Load initialises the configuration by reading login information from a file and setting up CLI API parameters.
// Verifies the existence of the configuration file and validates its integrity, terminating on errors.
func Load() {
	configPath, _ := cliflags.GetConfigLocation()
	exists, err := helper.FileExists(configPath)
	helper.Check(err)
	if !exists {
		fmt.Println("ERROR: No login information found")
		fmt.Println("Please run 'gokapi-cli login' to create a login")
		os.Exit(1)
	}
	data, err := os.ReadFile(configPath)
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
	configPath, _ := cliflags.GetConfigLocation()
	exists, err := helper.FileExists(configPath)
	if err != nil {
		fmt.Println("ERROR: Could not check if login information exists")
		fmt.Println(err)
		os.Exit(1)
	}
	if exists {
		return nil
	}
	return os.Remove(configPath)
}
