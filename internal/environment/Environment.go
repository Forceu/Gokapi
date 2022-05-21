package environment

import (
	"fmt"
	envParser "github.com/caarlos0/env/v6"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/environment/flagparser"
	"github.com/forceu/gokapi/internal/helper"
	"os"
	"strconv"
)

// DefaultPort for the webserver
const DefaultPort = 53842

// Environment is a struct containing available env variables
type Environment struct {
	ConfigDir     string `env:"CONFIG_DIR" envDefault:"config"`
	ConfigFile    string `env:"CONFIG_FILE" envDefault:"config.json"`
	DataDir       string `env:"DATA_DIR" envDefault:"data"`
	WebserverPort int    `env:"PORT" envDefault:"53842"`
	LengthId      int    `env:"LENGTH_ID" envDefault:"15"`
	MaxMemory     int    `env:"MAX_MEMORY_UPLOAD" envDefault:"40"`
	MaxFileSize   int    `env:"MAX_FILESIZE" envDefault:"102400"` // 102400==100GB
	AwsBucket     string `env:"AWS_BUCKET"`
	AwsRegion     string `env:"AWS_REGION"`
	AwsKeyId      string `env:"AWS_KEY"`
	AwsKeySecret  string `env:"AWS_KEY_SECRET"`
	AwsEndpoint   string `env:"AWS_ENDPOINT"`
	ConfigPath    string
	FileDbPath    string
	FileDb        string `env:"FILE_DB" envDefault:"filestorage.db"`
}

// New parses the env variables
func New() Environment {
	result := Environment{WebserverPort: DefaultPort}
	err := envParser.Parse(&result, envParser.Options{
		Prefix: "GOKAPI_",
	})
	if err != nil {
		fmt.Println("Error parsing env variables:", err)
		osExit(1)
		return Environment{}
	}
	helper.Check(err)

	flags := flagparser.ParseFlags()
	if flags.IsPortSet {
		result.WebserverPort = flags.Port
	}
	if flags.IsConfigDirSet {
		result.ConfigDir = flags.ConfigDir
	}
	if flags.IsDataDirSet {
		result.DataDir = flags.DataDir
	}

	result.ConfigPath = result.ConfigDir + "/" + result.ConfigFile
	if flags.IsConfigPathSet {
		result.ConfigPath = flags.ConfigPath
	}
	result.FileDbPath = result.DataDir + "/" + result.FileDb
	if IsDockerInstance() && os.Getenv("TMPDIR") == "" {
		os.Setenv("TMPDIR", result.DataDir)
	}
	if result.LengthId < 5 {
		result.LengthId = 5
	}
	if result.LengthId > database.GetLengthAvailable() {
		result.LengthId = database.GetLengthAvailable()
		fmt.Println("Reduced ID length to " + strconv.Itoa(database.GetLengthAvailable()) + " due to database constraints")
	}
	if result.MaxMemory < 5 {
		result.MaxMemory = 5
	}
	if result.MaxFileSize < 1 {
		result.MaxFileSize = 5
	}

	return result
}

// IsAwsProvided returns true if all required env variables have been set for using AWS S3 / Backblaze
func (e *Environment) IsAwsProvided() bool {
	return e.AwsBucket != "" &&
		e.AwsRegion != "" &&
		e.AwsKeyId != "" &&
		e.AwsKeySecret != ""
}

// GetConfigPaths returns the config paths to config files and the directory containing the files. The following results are returned:
// Path to config file, Path to directory containing config file, Name of config file, Path to AWS config file
func GetConfigPaths() (string, string, string, string) {
	env := New()
	awsConfigPAth := env.ConfigDir + "/cloudconfig.yml"
	return env.ConfigPath, env.ConfigDir, env.ConfigFile, awsConfigPAth
}

var osExit = os.Exit
