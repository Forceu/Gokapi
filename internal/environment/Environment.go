package environment

import (
	"Gokapi/internal/helper"
	"fmt"
	"os"

	envParser "github.com/caarlos0/env/v6"
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
	result.ConfigPath = result.ConfigDir + "/" + result.ConfigFile
	result.FileDbPath = result.DataDir + "/" + result.FileDb
	if IsDocker == "true" && os.Getenv("TMPDIR") == "" {
		os.Setenv("TMPDIR", result.DataDir)
	}
	if result.LengthId < 5 {
		result.LengthId = 5
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

func GetConfigPaths() (string, string, string, string) {
	env := New()
	awsConfigPAth := env.ConfigDir + "/cloudconfig.yml"
	return env.ConfigPath, env.ConfigDir, env.ConfigFile, awsConfigPAth
}

var osExit = os.Exit
