package main

import (
	"fmt"
	"github.com/forceu/gokapi/cmd/cli-uploader/cliapi"
	"github.com/forceu/gokapi/cmd/cli-uploader/cliconfig"
	"os"
)

const (
	paramLogin  = "login"
	paramLogout = "logout"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Valid options are:")
		fmt.Println("   gokapi-cli login")
		fmt.Println("   gokapi-cli logout")
		fmt.Println("   gokapi-cli upload [file to upload]")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "login":
		cliconfig.CreateLogin()
	case "logout":
		doLogout()
	case "upload":
		processUpload()
	default:
		printUsage()
	}
}

func processUpload() {
	cliconfig.Load()
	if len(os.Args) < 3 {
		fmt.Println("ERROR: Missing parameter file to upload")
		printUsage()
		os.Exit(1)
	}
	file, err := os.OpenFile(os.Args[2], os.O_RDONLY, 0664)
	if err != nil {
		fmt.Println("ERROR: Could not open file to upload")
		fmt.Println(err)
		os.Exit(2)
	}
	result, err := cliapi.UploadFile(file)
	if err != nil {
		fmt.Println("ERROR: Could not upload file")
		fmt.Println(err)
		os.Exit(3)
	}
	fmt.Println(result)
}

func printUsage() {
	fmt.Println("Valid options are:")
	fmt.Println("   gokapi-cli login")
	fmt.Println("   gokapi-cli logout")
	fmt.Println("   gokapi-cli upload")
	os.Exit(1)
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
