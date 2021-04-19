package testconfiguration

import (
	"Gokapi/internal/helper"
	"Gokapi/pkg/test"
	"testing"
)

func TestCreate(t *testing.T) {
	Create(true)
	test.IsEqualBool(t, helper.FolderExists("test"), true)
	test.IsEqualBool(t, helper.FileExists("test/config.json"), true)
	test.IsEqualBool(t, helper.FileExists("test/data/a8fdc205a9f19cc1c7507a60c4f01b13d11d7fd0"), true)
}

func TestDelete(t *testing.T) {
	Delete()
	test.IsEqualBool(t, helper.FolderExists("test"), false)
}

func TestMockInputStdin(t *testing.T) {
	original := StartMockInputStdin(t, "test")
	result := helper.ReadLine()
	StopMockInputStdin(original)
	test.IsEqualString(t, result, "test")
}
