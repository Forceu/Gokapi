// +build !integration

package main

import (
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

func TestParseFlags(t *testing.T) {
	os.Args = []string{"gokapi", "--version", "--reset-pw", "-create-ssl"}
	flags := parseFlags()
	test.IsEqualBool(t, flags.showVersion, true)
	test.IsEqualBool(t, flags.resetPw, true)
	test.IsEqualBool(t, flags.createSsl, true)
}

func TestNoShowVersion(t *testing.T) {
	showVersion(flags{})
}

func TestNoResetPw(t *testing.T) {
	resetPassword(flags{})
}

func TestCreateSsl(t *testing.T) {
	test.FileDoesNotExist(t, "test/ssl.key")
	createSsl(flags{})
	test.FileDoesNotExist(t, "test/ssl.key")
	createSsl(flags{createSsl: true})
	test.FileExists(t, "test/ssl.key")
}
