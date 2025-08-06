package cliflags

import (
	"fmt"
	"os"
	"strconv"
)

const (
	ModeLogin = iota
	ModeLogout
	ModeUpload
	ModeInvalid
)

type UploadConfig struct {
	File            string
	JsonOutput      bool
	DisableE2e      bool
	ExpiryDays      int
	ExpiryDownloads int
	Password        string
}

func Parse() int {
	if len(os.Args) < 2 {
		printUsage()
		return ModeInvalid
	}
	switch os.Args[1] {
	case "login":
		return ModeLogin
	case "logout":
		return ModeLogout
	case "upload":
		return ModeUpload
	default:
		printUsage()
		return ModeInvalid
	}
}

func GetUploadParameters() UploadConfig {
	result := UploadConfig{}
	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--json":
			result.JsonOutput = true
		case "--disable-e2e":
			result.DisableE2e = true
		case "--file":
			result.File = getParameter(&i)
		case "--expiry-days":
			result.ExpiryDays = requireInt(getParameter(&i))
		case "--expiry-downloads":
			result.ExpiryDownloads = requireInt(getParameter(&i))
		case "--password":
			result.Password = getParameter(&i)
		}
	}
	if result.File == "" {
		fmt.Println("ERROR: Missing parameter --file")
		os.Exit(2)
	}
	if result.ExpiryDownloads < 0 {
		result.ExpiryDownloads = 0
	}
	if result.ExpiryDays < 0 {
		result.ExpiryDays = 0
	}
	return result
}

func getParameter(position *int) string {
	newPosition := *position + 1
	position = &newPosition
	if newPosition >= len(os.Args) {
		printUsage()
		os.Exit(3)
	}
	return os.Args[newPosition]
}

func requireInt(input string) int {
	result, err := strconv.Atoi(input)
	if err != nil {
		fmt.Println("ERROR: " + input + " is not a valid integer")
		os.Exit(2)
	}
	return result
}

func printUsage() {
	fmt.Println("Gokapi CLI")
	fmt.Println()
	fmt.Println("Valid options are:")
	fmt.Println("   gokapi-cli login")
	fmt.Println("   gokapi-cli logout")
	fmt.Println("   gokapi-cli upload --file /file/to/upload [--json] [--disable-e2e]\n" +
		"                     [--expiry-days INT] [--expiry-downloads INT]\n" +
		"                     [--password STRING] ")
	fmt.Println()
	fmt.Println("gokapi-cli upload:")
	fmt.Println("--json              Outputs the result as JSON only")
	fmt.Println("--disable-e2e       Disables end-to-end encryption")
	fmt.Println("--expiry-days       Sets the expiry date of the file in days, otherwise unlimited")
	fmt.Println("--expiry-downloads  Sets the allowed downloads, otherwise unlimited")
	fmt.Println("--password          Sets a password")
	os.Exit(3)
}
