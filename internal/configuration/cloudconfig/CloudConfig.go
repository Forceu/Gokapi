package cloudconfig

import (
	"Gokapi/internal/environment"
	"Gokapi/internal/helper"
	"Gokapi/internal/models"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

// CloudConfig contains all configuration values / credentials for cloud storage
type CloudConfig struct {
	Aws models.AwsConfig `yaml:"aws"`
}

// Load loads cloud storage configuration / credentials from env variables or data/cloudconfig.yml
func Load() (CloudConfig, bool) {
	env := environment.New()
	if env.IsAwsProvided() {
		return loadFromEnv(&env), true
	}
	path := env.ConfigDir + "/cloudconfig.yml"
	if helper.FileExists(path) {
		return loadFromFile(path)
	}
	return CloudConfig{}, false
}

func Write(config CloudConfig) error {
	_, configDir, _, awsConfigPath := environment.GetConfigPaths()
	helper.CreateDir(configDir)
	file, err := os.OpenFile(awsConfigPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	err = encoder.Encode(config)
	if err != nil {
		return err
	}
	return nil
}

func Delete() error {
	_, _, _, awsConfigPath := environment.GetConfigPaths()
	if helper.FileExists(awsConfigPath) {
		err := os.Remove(awsConfigPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func loadFromEnv(env *environment.Environment) CloudConfig {
	return CloudConfig{Aws: models.AwsConfig{
		Bucket:    env.AwsBucket,
		Region:    env.AwsRegion,
		Endpoint:  env.AwsEndpoint,
		KeyId:     env.AwsKeyId,
		KeySecret: env.AwsKeySecret,
	}}
}

func loadFromFile(path string) (CloudConfig, bool) {
	var result CloudConfig
	file, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("Warning: Unable to read cloudconfig.yml!")
		return CloudConfig{}, false
	}
	err = yaml.Unmarshal(file, &result)
	if err != nil {
		fmt.Println("Warning: cloudconfig.yml contains invalid yaml!")
		return CloudConfig{}, false
	}
	return result, true
}
