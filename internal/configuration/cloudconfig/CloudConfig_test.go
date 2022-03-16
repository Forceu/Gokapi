package cloudconfig

import (
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestLoad(t *testing.T) {
	os.Unsetenv("GOKAPI_AWS_REGION")
	os.Unsetenv("GOKAPI_AWS_KEY")
	os.Unsetenv("GOKAPI_AWS_KEY_SECRET")
	config, ok := Load()
	test.IsEqualBool(t, ok, false)
	testconfiguration.WriteCloudConfigFile(true)
	os.Setenv("GOKAPI_AWS_BUCKET", "test")
	os.Setenv("GOKAPI_AWS_REGION", "test")
	os.Setenv("GOKAPI_AWS_KEY", "test")
	os.Setenv("GOKAPI_AWS_KEY_SECRET", "test")
	config, ok = Load()
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, config.Aws == models.AwsConfig{
		Bucket:    "test",
		Region:    "test",
		Endpoint:  "",
		KeyId:     "test",
		KeySecret: "test",
	}, true)
	os.Unsetenv("GOKAPI_AWS_BUCKET")
	config, ok = Load()
	savedConfig := models.AwsConfig{
		Bucket:    "gokapi",
		Region:    "test-region",
		Endpoint:  "test-endpoint",
		KeyId:     "test-keyid",
		KeySecret: "test-secret",
	}
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, config.Aws == savedConfig, true)
	os.Unsetenv("GOKAPI_AWS_REGION")
	os.Unsetenv("GOKAPI_AWS_KEY")
	os.Unsetenv("GOKAPI_AWS_KEY_SECRET")
	config, ok = Load()
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, config.Aws == savedConfig, true)
	os.Remove("test/cloudconfig.yml")
	config, ok = Load()
	test.IsEqualBool(t, ok, false)
	test.IsEqualBool(t, config.Aws == models.AwsConfig{}, true)
	testconfiguration.WriteCloudConfigFile(false)
	config, ok = Load()
	test.IsEqualBool(t, ok, false)
	test.IsEqualBool(t, config.Aws == models.AwsConfig{}, true)
}

func TestWrite(t *testing.T) {
	os.Remove("test/cloudconfig.yml")
	test.FileDoesNotExist(t, "test/cloudconfig.yml")
	config := CloudConfig{Aws: models.AwsConfig{
		Bucket:    "test1",
		Region:    "test2",
		Endpoint:  "test3",
		KeyId:     "test4",
		KeySecret: "test5",
	}}
	Write(config)
	test.FileExists(t, "test/cloudconfig.yml")
	newConfig, ok := Load()
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, newConfig.Aws == config.Aws, true)
}

func TestDelete(t *testing.T) {
	test.FileExists(t, "test/cloudconfig.yml")
	err := Delete()
	test.IsNil(t, err)
	test.FileDoesNotExist(t, "test/cloudconfig.yml")
	_, result := loadFromFile("test/cloudconfig.yml")
	test.IsEqualBool(t, result, false)
}
