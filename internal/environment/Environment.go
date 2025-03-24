package environment

import (
	"fmt"
	envParser "github.com/caarlos0/env/v6"
	"github.com/forceu/gokapi/internal/environment/flagparser"
	"github.com/forceu/gokapi/internal/helper"
	"os"
	"path"
	"strings"
)

// DefaultPort for the webserver
const DefaultPort = 53842

// Environment is a struct containing available env variables
type Environment struct {
	ChunkSizeMB        int    `env:"CHUNK_SIZE_MB" envDefault:"45"`
	ConfigDir          string `env:"CONFIG_DIR" envDefault:"config"`
	ConfigFile         string `env:"CONFIG_FILE" envDefault:"config.json"`
	ConfigPath         string
	DataDir            string `env:"DATA_DIR" envDefault:"data"`
	DatabaseUrl        string `env:"DATABASE_URL" envDefault:"sqlite://[data]/gokapi.sqlite"`
	LengthId           int    `env:"LENGTH_ID" envDefault:"15"`
	MaxFileSize        int    `env:"MAX_FILESIZE" envDefault:"102400"` // 102400==100GB
	MaxMemory          int    `env:"MAX_MEMORY_UPLOAD" envDefault:"50"`
	MaxParallelUploads int    `env:"MAX_PARALLEL_UPLOADS" envDefault:"4"`
	WebserverPort      int    `env:"PORT" envDefault:"53842"`
	DisableCorsCheck   bool   `env:"DISABLE_CORS_CHECK" envDefault:"false"`
	LogToStdout        bool   `env:"LOG_STDOUT" envDefault:"false"`
	HotlinkVideos      bool   `env:"ENABLE_HOTLINK_VIDEOS" envDefault:"false"`
	AwsBucket          string `env:"AWS_BUCKET"`
	AwsRegion          string `env:"AWS_REGION"`
	AwsKeyId           string `env:"AWS_KEY"`
	AwsKeySecret       string `env:"AWS_KEY_SECRET"`
	AwsEndpoint        string `env:"AWS_ENDPOINT"`
	AwsProxyDownload   bool   `env:"AWS_PROXY_DOWNLOAD" envDefault:"false"`
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

	result.ConfigDir = path.Clean(result.ConfigDir)
	result.DataDir = path.Clean(result.DataDir)
	result.ConfigPath = result.ConfigDir + "/" + result.ConfigFile
	if flags.IsConfigPathSet {
		result.ConfigPath = flags.ConfigPath
	}
	if IsDockerInstance() && os.Getenv("TMPDIR") == "" {
		err = os.Setenv("TMPDIR", result.DataDir)
		helper.Check(err)
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

	if flags.IsDatabaseUrlSet {
		result.DatabaseUrl = flags.DatabaseUrl
	}
	result.DatabaseUrl = strings.Replace(result.DatabaseUrl, "[data]", result.DataDir, 1)

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
func GetConfigPaths() (pathConfigFile, pathConfigDir, nameConfigFile, pathAwsConfig string) {
	env := New()
	pathAwsConfig = env.ConfigDir + "/cloudconfig.yml"
	return env.ConfigPath, env.ConfigDir, env.ConfigFile, pathAwsConfig
}

var osExit = os.Exit
