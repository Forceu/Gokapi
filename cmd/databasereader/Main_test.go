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

func TestRun(t *testing.T) {
	originalArgs := os.Args
	hasExited := false
	osExit = func(code int) {
		hasExited = true
	}
	os.Args = []string{os.Args[0]}
	main()
	test.IsEqualBool(t, hasExited, true)
	hasExited = false

	os.Args = append(os.Args, "")
	main()
	test.IsEqualBool(t, hasExited, true)
	hasExited = false

	os.Args[1] = "invalidFolder"
	main()
	test.IsEqualBool(t, hasExited, true)
	hasExited = false

	os.Args[1] = "./test/filestorage.db"
	main()
	test.IsEqualBool(t, hasExited, false)

	os.Args = originalArgs
}
