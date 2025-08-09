package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/forceu/gokapi/cmd/cli-uploader/cliapi"
	"github.com/forceu/gokapi/cmd/cli-uploader/cliconfig"
	"github.com/forceu/gokapi/cmd/cli-uploader/cliflags"
	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/helper"
	"os"
)

func main() {
	cliflags.Init(cliconfig.DockerFolderConfigFile, cliconfig.DockerFolderUpload)
	mode := cliflags.Parse()
	switch mode {
	case cliflags.ModeLogin:
		doLogin()
	case cliflags.ModeLogout:
		doLogout()
	case cliflags.ModeUpload:
		processUpload()
	case cliflags.ModeInvalid:
		os.Exit(3)
	}
}

func doLogin() {
	checkDockerFolders()
	cliconfig.CreateLogin()
}

func processUpload() {
	cliconfig.Load()
	uploadParam := cliflags.GetUploadParameters()

	result, err := cliapi.UploadFile(uploadParam)
	if err != nil {
		fmt.Println()
		if errors.Is(cliapi.EUnauthorised, err) {
			fmt.Println("ERROR: Unauthorised API key. Please re-run login.")
		} else {
			fmt.Println("ERROR: Could not upload file")
			fmt.Println(err)
		}
		os.Exit(1)
	}
	if uploadParam.JsonOutput {
		jsonStr, _ := json.Marshal(result)
		fmt.Println(string(jsonStr))
		return
	}
	fmt.Println("Upload successful")
	fmt.Println("File Name: " + result.Name)
	fmt.Println("File ID: " + result.Id)
	fmt.Println("File Download URL: " + result.UrlDownload)
}

func doLogout() {
	err := cliconfig.Delete()
	if err != nil {
		fmt.Println("ERROR: Could not delete configuration file")
		fmt.Println(err)
		os.Exit(2)
	}
	fmt.Println("Logged out. To login again, run: gokapi-cli login")
}

func checkDockerFolders() {
	if !environment.IsDockerInstance() {
		return
	}
	if !helper.FolderExists(cliconfig.DockerFolderConfig) {
		fmt.Println("Warning: Docker folder does not exist, configuration will be lost when creating a new container")
		helper.CreateDir(cliconfig.DockerFolderConfig)
	}
	if !helper.FolderExists(cliconfig.DockerFolderUpload) {
		helper.CreateDir(cliconfig.DockerFolderUpload)
	}
}
