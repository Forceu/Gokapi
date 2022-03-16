//go:build test

package testconfiguration

import (
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/storage/cloudstorage/aws"
	"github.com/forceu/gokapi/internal/test"
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

func TestSetUpgradeConfigFileV0(t *testing.T) {
	os.Remove(configFile)
	WriteUpgradeConfigFileV0()
	test.FileExists(t, configFile)
	TestDelete(t)
}
func TestSetUpgradeConfigFileV8(t *testing.T) {
	os.Remove(configFile)
	WriteUpgradeConfigFileV0()
	test.FileExists(t, configFile)
	TestDelete(t)
}

func TestWriteEncryptedFile(t *testing.T) {
	database.Init("./test/filestorage.db")
	fileId := WriteEncryptedFile()
	file, ok := database.GetMetaDataById(fileId)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, file.Id, fileId)
	database.Close()
}

func TestEnableS3(t *testing.T) {
	EnableS3()
	if aws.IsMockApi {
		test.IsEqualString(t, os.Getenv("GOKAPI_AWS_REGION"), "mock-region-1")
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

func TestWriteCloudConfigFile(t *testing.T) {
	test.FileDoesNotExist(t, "test/cloudconfig.yml")
	WriteCloudConfigFile(true)
	test.FileExists(t, "test/cloudconfig.yml")
	os.Remove("test/cloudconfig.yml")
	test.FileDoesNotExist(t, "test/cloudconfig.yml")
	WriteCloudConfigFile(false)
	test.FileExists(t, "test/cloudconfig.yml")
	Delete()
}
