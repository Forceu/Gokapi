package environment

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"strconv"

	envParser "github.com/caarlos0/env/v6"
	"github.com/forceu/gokapi/internal/environment/deprecation"
	"github.com/forceu/gokapi/internal/environment/flagparser"
	"github.com/forceu/gokapi/internal/helper"
)

// DefaultPort for the webserver
const DefaultPort = 53842

// Environment is a struct containing available env variables
type Environment struct {
	// Sets the directory for the config file
	ConfigDir string `env:"CONFIG_DIR" envDefault:"config"`
	// Sets the name of the config file
	ConfigFile string `env:"CONFIG_FILE" envDefault:"config.json"`
	// The full path to the config file
	ConfigPath string
	// Sets the directory for the data
	DataDir string `env:"DATA_DIR" envDefault:"data" persistent:"true"`
	// Sets the size of chunks that are uploaded in MB
	ChunkSizeMB int `env:"CHUNK_SIZE_MB" envDefault:"45" onlyPositive:"true" persistent:"true"`
	// Sets the length of the download IDs
	LengthId int `env:"LENGTH_ID" envDefault:"15" minValue:"5"`
	// Sets the length of the hotlink IDs
	LengthHotlinkId int `env:"LENGTH_HOTLINK_ID" envDefault:"40" minValue:"8"`
	// Sets the maximum allowed file size in MB
	// Default 102400 = 100GB
	MaxFileSize int `env:"MAX_FILESIZE" envDefault:"102400" onlyPositive:"true" persistent:"true"`
	// Sets the amount of RAM in MB that can be allocated for an upload chunk or file
	// Any chunk or file with a size greater than that will be written to a temporary file
	MaxMemory                   int  `env:"MAX_MEMORY_UPLOAD" envDefault:"50" onlyPositive:"true" persistent:"true"`
	MaxFilesGuestUpload         int  `env:"MAX_FILES_GUESTUPLOAD" envDefault:"100" onlyPositive:"true"`
	MaxSizeGuestUploadMb        int  `env:"MAX_SIZE_GUESTUPLOAD" envDefault:"10240" onlyPositive:"true"` // 10240 = 10GB
	PermRequestGrantedByDefault bool `env:"ALLOW_GUEST_UPLOADS_BY_DEFAULT" envDefault:"false"`
	// Set the number of chunks that are uploaded in parallel for a single file
	MaxParallelUploads int `env:"MAX_PARALLEL_UPLOADS" envDefault:"3" onlyPositive:"true" persistent:"true"`
	// Sets the minium free space on the disk in MB for accepting an upload
	MinFreeSpaceMB int `env:"MIN_FREE_SPACE" envDefault:"400" onlyPositive:"true"`
	// Sets the minium password length
	MinLengthPassword int `env:"MIN_LENGTH_PASSWORD" envDefault:"8" minValue:"6"`
	// Sets the webserver port
	WebserverPort int `env:"PORT" envDefault:"53842" onlyPositive:"true" persistent:"true"`
	// Disables the CORS check on startup and during setup, if set to true
	DisableCorsCheck bool `env:"DISABLE_CORS_CHECK" envDefault:"false"`
	// Also outputs all log file entries to the console output, if set to true
	LogToStdout bool `env:"LOG_STDOUT" envDefault:"false"`
	// Allow hotlinking of videos. Note: Due to buffering, playing a video might count as
	// multiple downloads. It is only recommended to use video hotlinking for uploads with
	// unlimited downloads enabled
	HotlinkVideos bool `env:"ENABLE_HOTLINK_VIDEOS" envDefault:"false"`
	// Sets the AWS bucket name
	AwsBucket string `env:"AWS_BUCKET"`
	// Sets the AWS region name
	AwsRegion string `env:"AWS_REGION"`
	// Sets the AWS API key
	AwsKeyId string `env:"AWS_KEY"`
	// Sets the AWS API secret
	AwsKeySecret string `env:"AWS_KEY_SECRET"`
	// Sets the AWS endpoint
	AwsEndpoint string `env:"AWS_ENDPOINT"`
	// List of active deprecations
	ActiveDeprecations []deprecation.Deprecation
	isSet              bool
}

func (e *Environment) IsParsed() bool {
	return e.isSet
}

// New parses the env variables
func New() Environment {
	result := Environment{
		WebserverPort: DefaultPort,
		isSet:         true,
	}

	result = parseEnvVars(result)
	err := enforceIntLimits(&result)
	if err != nil {
		fmt.Println("Error parsing env variables:", err)
		osExit(1)
	}
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
	return result
}

func enforceIntLimits(result *Environment) error {
	v := reflect.ValueOf(result)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("env must be a pointer to a struct")
	}

	v = v.Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		fieldVal := v.Field(i)
		fieldType := t.Field(i)

		checkForPositive := fieldType.Tag.Get("onlyPositive") != ""
		checkForMinValue := fieldType.Tag.Get("minValue") != ""

		if !checkForPositive && !checkForMinValue {
			continue
		}

		// Only handle signed integers
		switch fieldVal.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

			if checkForMinValue {
				valStr := fieldType.Tag.Get("minValue")
				minVal, err := strconv.ParseInt(valStr, 10, fieldVal.Type().Bits())
				if err != nil {
					return fmt.Errorf("invalid minValue for field %s: %w", fieldType.Name, err)
				}
				if fieldVal.Int() < minVal {
					if fieldVal.CanSet() {
						fieldVal.SetInt(minVal)
					} else {
						return fmt.Errorf("cannot set fieldval %s", fieldType.Name)
					}
					continue
				}
			}

			if !checkForPositive {
				continue
			}
			if fieldVal.Int() >= 0 {
				continue
			}

			defaultStr := fieldType.Tag.Get("envDefault")
			defaultVal, err := strconv.ParseInt(defaultStr, 10, fieldVal.Type().Bits())
			if err != nil {
				return fmt.Errorf("invalid envDefault for field %s: %w", fieldType.Name, err)
			}

			if fieldVal.CanSet() {
				fieldVal.SetInt(defaultVal)
			} else {
				return fmt.Errorf("cannot set fieldval %s", fieldType.Name)
			}
		default:
			continue
		}
	}
	return nil
}

func parseFlags(result Environment) Environment {
	flags := flagparser.ParseFlags()
	if flags.IsPortSet {
		if flags.Port < 1 {
			flags.Port = DefaultPort
		}
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
