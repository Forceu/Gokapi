package environment

import (
	"os"
	"reflect"
	"strconv"
	"strings"
)

const IsTrue = "yes"
const IsFalse = "no"

type Environment struct {
	ConfigDir          string
	ConfigFile         string
	ConfigPath         string
	DataDir            string
	AdminName          string
	AdminPassword      string
	WebserverPort      string
	WebserverLocalhost string
	ExternalUrl        string
	RedirectUrl        string
	SaltAdmin          string
	SaltFiles          string
	LengthId           int
}

var defaultValues = defaultsEnvironment{
	CONFIG_DIR:  "config",
	CONFIG_FILE: "config.json",
	DATA_DIR:    "data",
	SALT_ADMIN:  "eefwkjqweduiotbrkl##$2342brerlk2321",
	SALT_FILES:  "P1UI5sRNDwuBgOvOYhNsmucZ2pqo4KEvOoqqbpdu",
	LENGTH_ID:   15,
}

func New() Environment {
	configDir := envString("CONFIG_DIR")
	configFile := envString("CONFIG_FILE")
	configPath := configDir + "/" + configFile

	return Environment{
		ConfigDir:          configDir,
		ConfigFile:         configFile,
		ConfigPath:         configPath,
		DataDir:            envString("DATA_DIR"),
		AdminName:          envString("USERNAME"),
		AdminPassword:      envString("PASSWORD"),
		WebserverPort:      envString("PORT"),
		ExternalUrl:        envString("EXTERNAL_URL"),
		RedirectUrl:        envString("REDIRECT_URL"),
		SaltAdmin:          envString("SALT_ADMIN"),
		SaltFiles:          envString("SALT_FILES"),
		WebserverLocalhost: envBool("LOCALHOST"),
		LengthId:           envInt("LENGTH_ID"),
	}
}

// Looks up an environment variable or returns an empty string
func envString(key string) string {
	val, ok := os.LookupEnv("GOKAPI_" + key)
	if !ok {
		return defaultValues.GetString(key)
	}
	return val
}

// Looks up a boolean environment variable, returns either IsFalse, IsTrue or IsUnset
func envBool(key string) string {
	val, ok := os.LookupEnv("GOKAPI_" + key)
	if !ok {
		return ""
	}
	valLower := strings.ToLower(val)
	if valLower == "true" || valLower == "yes" {
		return IsTrue
	}
	if valLower == "false" || valLower == "no" {
		return IsFalse
	}
	return ""
}

// Looks up an environment variable or returns an empty string
func envInt(key string) int {
	val, ok := os.LookupEnv("GOKAPI_" + key)
	if !ok {
		return defaultValues.GetInt(key)
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return -1
	}
	return intVal

}

func (structPointer *defaultsEnvironment) GetString(name string) string {
	field := reflect.ValueOf(structPointer).Elem().FieldByName(name)
	if field.IsValid() {
		return field.String()
	}
	return ""
}

func (structPointer *defaultsEnvironment) GetInt(name string) int {
	field := reflect.ValueOf(structPointer).Elem().FieldByName(name)
	if field.IsValid() {
		return int(field.Int())
	}
	return -1
}

type defaultsEnvironment struct {
	CONFIG_DIR  string
	CONFIG_FILE string
	DATA_DIR    string
	SALT_ADMIN  string
	SALT_FILES  string
	LENGTH_ID   int
}
