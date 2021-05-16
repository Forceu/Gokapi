package models

// AwsConfig contains all configuration values / credentials for AWS cloud storage
type AwsConfig struct {
	Bucket    string `yaml:"Bucket"`
	Region    string `yaml:"Region"`
	Endpoint  string `yaml:"Endpoint"`
	KeyId     string `yaml:"KeyId"`
	KeySecret string `yaml:"KeySecret"`
}

// IsAllProvided returns true if all required variables have been set for using AWS S3 / Backblaze
func (c *AwsConfig) IsAllProvided() bool {
	return c.Bucket != "" &&
		c.Region != "" &&
		c.KeyId != "" &&
		c.KeySecret != ""
}
