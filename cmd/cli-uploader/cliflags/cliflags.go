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
		printUsage(3)
		return ModeInvalid
	}
	switch os.Args[1] {
	case "login":
		return ModeLogin
	case "logout":
		return ModeLogout
	case "upload":
		return ModeUpload
	case "help":
		printUsage(0)
	default:
		printUsage(3)
	}
	return ModeInvalid
}

func GetUploadParameters() UploadConfig {
	result := UploadConfig{}
	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-j":
			fallthrough
		case "--json":
			result.JsonOutput = true
		case "-n":
			fallthrough
		case "--disable-e2e":
			result.DisableE2e = true
		case "-f":
			fallthrough
		case "--file":
			result.File = getParameter(&i)
		case "-e":
			fallthrough
		case "--expiry-days":
			result.ExpiryDays = requireInt(getParameter(&i))
		case "-d":
			fallthrough
		case "--expiry-downloads":
			result.ExpiryDownloads = requireInt(getParameter(&i))
		case "-p":
			fallthrough
		case "--password":
			result.Password = getParameter(&i)
		case "-h":
			fallthrough
		case "--help":
			printUsage(0)
		}
	}
	if result.File == "" {
		if environment.IsDockerInstance() {
			ok, dockerFile := getDockerUpload()
			if !ok {
				fmt.Println("ERROR: Missing parameter --file and no file or more than one file found in " + dockerUploadFolder)
				os.Exit(2)
			}
			result.File = dockerFile
		} else {
			fmt.Println("ERROR: Missing parameter --file")
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
			fallthrough
		case "--configuration":
			return getParameter(&i)
		}
	}
	return "gokapi-cli.json"
}

func getParameter(position *int) string {
	newPosition := *position + 1
	position = &newPosition
	if newPosition >= len(os.Args) {
		printUsage(3)
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

func printUsage(exitCode int) {
	fmt.Println("Gokapi CLI v1.0")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gokapi-cli [command] [options]")
	fmt.Println()

	fmt.Println("Commands:")
	fmt.Println("  login       Save login credentials")
	fmt.Println("  upload      Upload a file to Gokapi instance")
	fmt.Println("  logout      Delete login credentials")
	fmt.Println()

	fmt.Println("Options:")
	fmt.Println("  -f, --file <path>               File to upload")
	if !environment.IsDockerInstance() {
		fmt.Println("  -c, --configuration <path>      Path to configuration file (default: gokapi-cli.json)")
	}
	fmt.Println("  -j, --json                      Output the result in JSON only")
	fmt.Println("  -n, --disable-e2e               Disable end-to-end encryption")
	fmt.Println("  -e, --expiry-days <int>         Set file expiry in days (default: unlimited)")
	fmt.Println("  -d, --expiry-downloads <int>    Set max allowed downloads (default: unlimited)")
	fmt.Println("  -p, --password <string>         Set a password for the file")
	fmt.Println("  -h, --help                      Show this help message")
	fmt.Println()

	fmt.Println("Examples:")
	fmt.Println("  gokapi-cli login")
	fmt.Println("  gokapi-cli logout -c /path/to/config")
	fmt.Println("  gokapi-cli upload -f /file/to/upload --expiry-days 7 --json")
	fmt.Println()
	os.Exit(exitCode)
}
