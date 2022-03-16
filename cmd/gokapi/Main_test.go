//go:build !integration && test

package main

import (
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

func TestParseFlags(t *testing.T) {
	os.Args = []string{"gokapi", "--version", "--reconfigure", "-create-ssl"}
	flags := parseFlags()
	test.IsEqualBool(t, flags.showVersion, true)
	test.IsEqualBool(t, flags.reconfigure, true)
	test.IsEqualBool(t, flags.createSsl, true)
	os.Args = []string{"gokapi", "--reconfigure", "-create-ssl"}
	flags = parseFlags()
	test.IsEqualBool(t, flags.showVersion, false)
	test.IsEqualBool(t, flags.reconfigure, true)
	test.IsEqualBool(t, flags.createSsl, true)
}

func TestShowVersion(t *testing.T) {
	showVersion(flags{})
	osExit = test.ExitCode(t, 0)
	showVersion(flags{showVersion: true})
}

func TestNoResetPw(t *testing.T) {
	reconfigureServer(flags{})
}

func TestCreateSsl(t *testing.T) {
	test.FileDoesNotExist(t, "test/ssl.key")
	createSsl(flags{})
	test.FileDoesNotExist(t, "test/ssl.key")
	createSsl(flags{createSsl: true})
	test.FileExists(t, "test/ssl.key")
}
