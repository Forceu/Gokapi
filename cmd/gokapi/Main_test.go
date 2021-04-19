package main

import (
	"Gokapi/pkg/test"
	"os"
	"testing"
)

func TestParseFlags(t *testing.T) {
	os.Args = []string{"gokapi", "--version", "--reset-pw"}
	flags := parseFlags()
	test.IsEqualBool(t, flags.showVersion, true)
	test.IsEqualBool(t, flags.resetPw, true)
}

func TestNoShowVersion(t *testing.T) {
	showVersion(flags{})
}

func TestNoResetPw(t *testing.T) {
	resetPassword(flags{})
}
