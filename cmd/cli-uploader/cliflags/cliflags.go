package cliflags

import (
	"fmt"
	"github.com/forceu/gokapi/internal/environment"
	"os"
	"path/filepath"
	"strconv"
)

const (
	ModeLogin = iota
	ModeLogout
	ModeUpload
	ModeInvalid
)

var dockerConfigFile string
var dockerUploadFolder string

type UploadConfig struct {
	File            string
	JsonOutput      bool
	DisableE2e      bool
	ExpiryDays      int
	ExpiryDownloads int
	Password        string
}

func Init(dockerConfigPath, dockerUploadPath string) {
	dockerConfigFile = dockerConfigPath
	dockerUploadFolder = dockerUploadPath
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
		case "-f":
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
		if environment.IsDockerInstance() {
			ok, dockerFile := getDockerUpload()
			if !ok {
				fmt.Println("ERROR: Missing parameter -f and no file or more than one file found in " + dockerUploadFolder)
				os.Exit(2)
			}
			result.File = dockerFile
		} else {
			fmt.Println("ERROR: Missing parameter -f")
			os.Exit(2)
		}
	}
	if result.ExpiryDownloads < 0 {
		result.ExpiryDownloads = 0
	}
	if result.ExpiryDays < 0 {
		result.ExpiryDays = 0
	}
	return result
}

func getDockerUpload() (bool, string) {
	if !environment.IsDockerInstance() {
		return false, ""
	}
	entries, err := os.ReadDir(dockerUploadFolder)
	if err != nil {
		return false, ""
	}

	var fileName string
	var fileFound bool
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			if fileFound {
				// More than one file exist
				return false, ""
			}
			fileName = entry.Name()
			fileFound = true
		}
	}
	if !fileFound {
		return false, ""
	}
	return true, filepath.Join(dockerUploadFolder, fileName)
}

func GetConfigLocation() string {
	if environment.IsDockerInstance() {
		return dockerConfigFile
	}
	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-c":
			return getParameter(&i)
		}
	}
	return "gokapi-cli.json"
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
	fmt.Println("Gokapi CLI v1.0")
	fmt.Println()
	fmt.Println("Valid options are:")
	fmt.Println("   gokapi-cli login [-c /path/to/config]")
	fmt.Println("   gokapi-cli logout [-c /path/to/config]")
	fmt.Println("   gokapi-cli upload --f /file/to/upload [--json] [--disable-e2e]\n" +
		"                     [--expiry-days INT] [--expiry-downloads INT]\n" +
		"                     [--password STRING] [-c /path/to/config]")
	fmt.Println()
	fmt.Println("gokapi-cli upload:")
	fmt.Println("--json              Outputs the result as JSON only")
	fmt.Println("--disable-e2e       Disables end-to-end encryption")
	fmt.Println("--expiry-days       Sets the expiry date of the file in days, otherwise unlimited")
	fmt.Println("--expiry-downloads  Sets the allowed downloads, otherwise unlimited")
	fmt.Println("--password          Sets a password")
	os.Exit(3)
}
