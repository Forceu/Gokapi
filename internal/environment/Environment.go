package environment

import (
	"os"
	"reflect"
	"strconv"
)

// DefaultPort for the webserver
const DefaultPort = 53842

// Environment is a struct containing available env variables
type Environment struct {
	ConfigDir     string
	ConfigFile    string
	ConfigPath    string
	DataDir       string
	WebserverPort string
	LengthId      int
	MaxMemory     int
	MaxFileSize   int
	AwsBucket     string
	AwsRegion     string
	AwsKeyId      string
	AwsKeySecret  string
	AwsEndpoint   string
}

var defaultValues = defaultsEnvironment{
	CONFIG_DIR:           "config",
	CONFIG_FILE:          "config.json",
	DATA_DIR:             "data",
	PORT:                 strconv.Itoa(DefaultPort),
	LENGTH_ID:            15,
	MAX_MEMORY_UPLOAD_MB: 20,
	MAX_FILESIZE:         102400, // 100GB
}

// New parses the env variables
func New() Environment {
	configPath, configDir, configFile, _ := GetConfigPaths()
	return Environment{
		ConfigDir:     configDir,
		ConfigFile:    configFile,
		ConfigPath:    configPath,
		WebserverPort: GetPort(),
		DataDir:       envString("DATA_DIR"),
		LengthId:      envInt("LENGTH_ID", 5),
		MaxMemory:     envInt("MAX_MEMORY_UPLOAD_MB", 5),
		MaxFileSize:   envInt("MAX_FILESIZE", 1),
		AwsBucket:     envString("AWS_BUCKET"),
		AwsRegion:     envString("AWS_REGION"),
		AwsKeyId:      envString("AWS_KEY"),
		AwsKeySecret:  envString("AWS_KEY_SECRET"),
		AwsEndpoint:   envString("AWS_ENDPOINT"),
	}
}

// IsAwsProvided returns true if all required env variables have been set for using AWS S3 / Backblaze
func (e *Environment) IsAwsProvided() bool {
	return e.AwsBucket != "" &&
		e.AwsRegion != "" &&
		e.AwsKeyId != "" &&
		e.AwsKeySecret != ""
}

// Looks up an environment variable or returns an empty string
func envString(key string) string {
	val, ok := os.LookupEnv("GOKAPI_" + key)
	if !ok {
		return defaultValues.getString(key)
	}
	return val
}

// Looks up an environment variable or returns an empty string
func envInt(key string, minValue int) int {
	val, ok := os.LookupEnv("GOKAPI_" + key)
	if !ok {
		return defaultValues.getInt(key)
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return -1
	}
	if intVal < minValue {
		return minValue
	}
	return intVal

}

func GetConfigPaths() (string, string, string, string) {
	configDir := envString("CONFIG_DIR")
	configFile := envString("CONFIG_FILE")
	configPath := configDir + "/" + configFile
	awsConfigPAth := configDir + "/cloudconfig.yml"
	return configPath, configDir, configFile, awsConfigPAth
}

func GetPort() string {
	return envString("PORT")
}

// Gets the env variable or default value as string
func (structPointer *defaultsEnvironment) getString(name string) string {
	field := reflect.ValueOf(structPointer).Elem().FieldByName(name)
	if field.IsValid() {
		return field.String()
	}
	return ""
}

// Gets the env variable or default value as int
func (structPointer *defaultsEnvironment) getInt(name string) int {
	field := reflect.ValueOf(structPointer).Elem().FieldByName(name)
	if field.IsValid() {
		return int(field.Int())
	}
	return -1
}

type defaultsEnvironment struct {
	CONFIG_DIR           string
	CONFIG_FILE          string
	DATA_DIR             string
	PORT                 string
	LENGTH_ID            int
	MAX_MEMORY_UPLOAD_MB int
	MAX_FILESIZE         int
}
