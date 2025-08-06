package main

import (
	"encoding/json"
	"fmt"
	"github.com/forceu/gokapi/cmd/cli-uploader/cliapi"
	"github.com/forceu/gokapi/cmd/cli-uploader/cliconfig"
	"github.com/forceu/gokapi/cmd/cli-uploader/cliflags"
	"os"
)

func main() {
	mode := cliflags.Parse()
	switch mode {
	case cliflags.ModeLogin:
		cliconfig.CreateLogin()
	case cliflags.ModeLogout:
		doLogout()
	case cliflags.ModeUpload:
		processUpload()
	case cliflags.ModeInvalid:
		os.Exit(3)
	}
}

func processUpload() {
	cliconfig.Load()
	uploadParam := cliflags.GetUploadParameters()

	result, err := cliapi.UploadFile(uploadParam)
	if err != nil {
		fmt.Println("ERROR: Could not upload file")
		fmt.Println(err)
		os.Exit(1)
	}
	if uploadParam.JsonOutput {
		jsonStr, _ := json.Marshal(result)
		fmt.Println(string(jsonStr))
	} else {
		fmt.Println("File uploaded successfully")
		fmt.Println("File ID: " + result.Id)
		fmt.Println("File Download URL: " + result.UrlDownload)
	}
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
