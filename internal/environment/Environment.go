package environment

import (
	"fmt"
	"os"
	"path"

	envParser "github.com/caarlos0/env/v6"
	"github.com/forceu/gokapi/internal/environment/deprecation"
	"github.com/forceu/gokapi/internal/environment/flagparser"
	"github.com/forceu/gokapi/internal/helper"
)

// DefaultPort for the webserver
const DefaultPort = 53842

// Environment is a struct containing available env variables
type Environment struct {
	ConfigDir          string `env:"CONFIG_DIR" envDefault:"config"`
	ConfigFile         string `env:"CONFIG_FILE" envDefault:"config.json"`
	ConfigPath         string
	DataDir            string `env:"DATA_DIR" envDefault:"data"`
	ChunkSizeMB        int    `env:"CHUNK_SIZE_MB" envDefault:"45"`
	LengthId           int    `env:"LENGTH_ID" envDefault:"15"`
	LengthHotlinkId    int    `env:"LENGTH_HOTLINK_ID" envDefault:"40"`
	MaxFileSize        int    `env:"MAX_FILESIZE" envDefault:"102400"` // 102400==100GB
	MaxMemory          int    `env:"MAX_MEMORY_UPLOAD" envDefault:"50"`
	MaxParallelUploads int    `env:"MAX_PARALLEL_UPLOADS" envDefault:"3"`
	MinFreeSpaceMB     int    `env:"MIN_FREE_SPACE" envDefault:"400"`
	WebserverPort      int    `env:"PORT" envDefault:"53842"`
	MinLengthPassword  int    `env:"MIN_LENGTH_PASSWORD" envDefault:"8"`
	DisableCorsCheck   bool   `env:"DISABLE_CORS_CHECK" envDefault:"false"`
	LogToStdout        bool   `env:"LOG_STDOUT" envDefault:"false"`
	HotlinkVideos      bool   `env:"ENABLE_HOTLINK_VIDEOS" envDefault:"false"`
	AwsBucket          string `env:"AWS_BUCKET"`
	AwsRegion          string `env:"AWS_REGION"`
	AwsKeyId           string `env:"AWS_KEY"`
	AwsKeySecret       string `env:"AWS_KEY_SECRET"`
	AwsEndpoint        string `env:"AWS_ENDPOINT"`
	ActiveDeprecations []deprecation.Deprecation
}

// New parses the env variables
func New() Environment {
	result := Environment{WebserverPort: DefaultPort}

	result = parseEnvVars(result)
	result = parseFlags(result)
	result.ActiveDeprecations = deprecation.GetActive()

	return result
}

func parseEnvVars(result Environment) Environment {
	err := envParser.Parse(&result, envParser.Options{
		Prefix: "GOKAPI_",
	})
	if err != nil {
		fmt.Println("Error parsing env variables:", err)
		osExit(1)
		return Environment{}
	}
	helper.Check(err)

	if result.LengthId < 5 {
		result.LengthId = 5
	}
	if result.LengthHotlinkId < 8 {
		result.LengthHotlinkId = 8
	}
	if result.MaxMemory < 5 {
		result.MaxMemory = 5
	}
	if result.MaxFileSize < 1 {
		result.MaxFileSize = 5
	}

	return result
}

func parseFlags(result Environment) Environment {
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
		err := os.Setenv("TMPDIR", result.DataDir)
		helper.Check(err)
	}
	if result.LengthId < 5 {
		result.LengthId = 5
	}
	if result.LengthHotlinkId < 8 {
		result.LengthHotlinkId = 8
	}
	if result.MaxMemory < 5 {
		result.MaxMemory = 5
	}
	if result.MaxFileSize < 1 {
		result.MaxFileSize = 5
	}
	if result.MinLengthPassword < 6 {
		result.MinLengthPassword = 6
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
func GetConfigPaths() (pathConfigFile, pathConfigDir, nameConfigFile, pathAwsConfig string) {
	env := New()
	pathAwsConfig = env.ConfigDir + "/cloudconfig.yml"
	return env.ConfigPath, env.ConfigDir, env.ConfigFile, pathAwsConfig
}

var osExit = os.Exit
