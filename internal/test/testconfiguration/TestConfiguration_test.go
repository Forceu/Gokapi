package testconfiguration

import (
	"Gokapi/internal/helper"
	"Gokapi/internal/test"
	"os"
	"testing"
)

func TestCreate(t *testing.T) {
	Create(true)
	test.IsEqualBool(t, helper.FolderExists(dataDir), true)
	test.IsEqualBool(t, helper.FileExists(configFile), true)
	test.IsEqualBool(t, helper.FileExists("test/data/a8fdc205a9f19cc1c7507a60c4f01b13d11d7fd0"), true)
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
	test.IsEqualBool(t, helper.FileExists(configFile), true)
	TestDelete(t)
}