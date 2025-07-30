package cliconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/forceu/gokapi/cmd/cli-uploader/cliapi"
	"github.com/forceu/gokapi/internal/helper"
	"os"
	"strings"
)

const minGokapiVersionInt = 20100
const minGokapiVersionStr = "2.1.0"

const filename = "gokapi-cli.json"

type configFile struct {
	Url    string `json:"Url"`
	Apikey string `json:"Apikey"`
	E2ekey string `json:"E2Ekey"`
}

func CreateLogin() {
	fmt.Print("Gokapi URL: ")
	url := helper.ReadLine()
	if (!strings.HasPrefix(url, "http://")) && (!strings.HasPrefix(url, "https://")) {
		fmt.Println("ERROR: URL must start with http:// or https://")
		os.Exit(1)
	}
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
	cliapi.Init(url, apikey, "")
	vstr, vint, err := cliapi.GetVersion()
	if err != nil {
		fmt.Println()
		if errors.Is(cliapi.EUnauthorised, err) {
			fmt.Println("ERROR: Unauthorised API key")
		} else {
			fmt.Println(err)
		}
		os.Exit(1)
	}

	if vint < minGokapiVersionInt {
		fmt.Println("\nERROR: Gokapi version must be at least " + minGokapiVersionStr)
		fmt.Println("Your version is " + vstr)
		os.Exit(1)
	}
	fmt.Print("OK\nDownloading configuration...")

	_, _, isE2E, err := cliapi.GetConfig()
	if err != nil {
		fmt.Println("FAIL")
		if errors.Is(cliapi.EUnauthorised, err) {
			fmt.Println("ERROR: API key does not have the permission to upload new files.")
		} else {
			fmt.Println(err)
		}
		os.Exit(1)
	}
	fmt.Println("OK")
	var e2ekey = ""
	if isE2E {
		fmt.Print("End-to-end encryption key: ")
		e2ekey = helper.ReadLine()
		// TODO check if key is invalid or not generated yet
	}

	err = save(url, apikey, e2ekey)
	if err != nil {
		fmt.Println("ERROR: Could not save login information")
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Login successful")
}

func save(url, apikey, e2ekey string) error {
	configData := configFile{
		Url:    url,
		Apikey: apikey,
		E2ekey: e2ekey,
	}

	jsonData, err := json.Marshal(configData)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, jsonData, 0600)
}

func Load() {
	if !helper.FileExists(filename) {
		fmt.Println("ERROR: No login information found")
		fmt.Println("Please run 'gokapi-cli login' to create a login")
		os.Exit(1)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("ERROR: Could not read login information")
		os.Exit(1)
	}

	var config configFile
	err = json.Unmarshal(data, &config)
	if err != nil {
		fmt.Println("ERROR: Could not read login information")
		os.Exit(1)
	}

	cliapi.Init(config.Url, config.Apikey, config.E2ekey)
}

func Delete() error {
	if !helper.FileExists(filename) {
		return nil
	}
	return os.Remove(filename)
}
