package main

import (
	"fmt"
	"github.com/forceu/gokapi/cmd/cli-uploader/cliconfig"
	"os"
)

const (
	paramLogin  = "login"
	paramLogout = "logout"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("ERROR: No command given")
		os.Exit(1)
	}
	switch os.Args[1] {
	case paramLogin:
		cliconfig.CreateLogin()
	case paramLogout:
		doLogout()
	default:
	}

}

func doLogout() {
	cliconfig.Delete()
}
