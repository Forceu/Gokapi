//go:build !integration && test

package main

import (
	"github.com/forceu/gokapi/internal/configuration/flagparser"
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
	flags := flagparser.ParseFlags()
	test.IsEqualBool(t, flags.ShowVersion, true)
	test.IsEqualBool(t, flags.Reconfigure, true)
	test.IsEqualBool(t, flags.CreateSsl, true)
	os.Args = []string{"gokapi", "--reconfigure", "-create-ssl"}
	flags = flagparser.ParseFlags()
	test.IsEqualBool(t, flags.ShowVersion, false)
	test.IsEqualBool(t, flags.Reconfigure, true)
	test.IsEqualBool(t, flags.CreateSsl, true)
}

func TestShowVersion(t *testing.T) {
	showVersion(flagparser.MainFlags{})
	osExit = test.ExitCode(t, 0)
	showVersion(flagparser.MainFlags{ShowVersion: true})
}

func TestNoResetPw(t *testing.T) {
	reconfigureServer(flagparser.MainFlags{})
}

func TestCreateSsl(t *testing.T) {
	test.FileDoesNotExist(t, "test/ssl.key")
	createSsl(flagparser.MainFlags{})
	test.FileDoesNotExist(t, "test/ssl.key")
	createSsl(flagparser.MainFlags{CreateSsl: true})
	test.FileExists(t, "test/ssl.key")
}
