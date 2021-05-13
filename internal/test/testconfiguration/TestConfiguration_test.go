package testconfiguration

import (
	"Gokapi/internal/helper"
	"Gokapi/internal/storage/aws"
	"Gokapi/internal/test"
	"os"
	"testing"
)

func TestCreate(t *testing.T) {
	Create(true)
	test.IsEqualBool(t, helper.FolderExists(dataDir), true)
	test.FileExists(t, configFile)
	test.FileExists(t, "test/data/a8fdc205a9f19cc1c7507a60c4f01b13d11d7fd0")
}

func TestDelete(t *testing.T) {
	Delete()
	test.IsEqualBool(t, helper.FolderExists(dataDir), false)
}

func TestMockInputStdin(t *testing.T) {
	original := StartMockInputStdin(dataDir)
	result := helper.ReadLine()
	StopMockInputStdin(original)
	test.IsEqualString(t, result, dataDir)
}

func TestSetUpgradeConfigFile(t *testing.T) {
	os.Remove(configFile)
	WriteUpgradeConfigFile()
	test.FileExists(t, configFile)
	TestDelete(t)
}

func TestEnableS3(t *testing.T) {
	EnableS3()
	if aws.IsMockApi {
		test.IsEqualString(t, os.Getenv("AWS_REGION"), "mock-region-1")
	}
}
func TestDisableS3S3(t *testing.T) {
	DisableS3()
	if aws.IsMockApi {
		test.IsEqualString(t, os.Getenv("AWS_REGION"), "")
	}
}

func TestWriteSslCertificates(t *testing.T) {
	test.FileDoesNotExist(t, "test/ssl.key")
	WriteSslCertificates(true)
	test.FileExists(t, "test/ssl.key")
	os.Remove("test/ssl.key")
	test.FileDoesNotExist(t, "test/ssl.key")
	WriteSslCertificates(false)
	test.FileExists(t, "test/ssl.key")
	Delete()
}
