package api

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/test"
	"Gokapi/internal/test/testconfiguration"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	configuration.Load()
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestIsValidApiKey(t *testing.T) {
	test.IsEqualBool(t, isValidApiKey(""), false)
	test.IsEqualBool(t, isValidApiKey("invalid"), false)
	test.IsEqualBool(t, isValidApiKey("validkey"), true)
}
