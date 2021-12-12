//go:build test
// +build test

package cloudconfig

import (
	"Gokapi/internal/models"
	"Gokapi/internal/test"
	"Gokapi/internal/test/testconfiguration"
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
	testconfiguration.WriteCloudConfigFile(true)
	os.Setenv("GOKAPI_AWS_BUCKET", "test")
	os.Setenv("GOKAPI_AWS_REGION", "test")
	os.Setenv("GOKAPI_AWS_KEY", "test")
	os.Setenv("GOKAPI_AWS_KEY_SECRET", "test")
	config, ok := Load()
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
