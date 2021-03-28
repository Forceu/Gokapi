package main

/**
Main routine
*/

import (
	"fmt"
	"math/rand"
	"os"
	"time"
)


// needs to be changed in ./templates/string_constants.tmpl as well
const VERSION = "1.1.0"

// Salt for the admin password hash
const SALT_PW_ADMIN = "eefwkjqweduiotbrkl##$2342brerlk2321"

// Salt for the file password hashes
const SALT_PW_FILES = "P1UI5sRNDwuBgOvOYhNsmucZ2pqo4KEvOoqqbpdu"

// Main routine that is called on startup
func main() {
	checkPrimaryArguments()
	rand.Seed(time.Now().UnixNano())
	fmt.Println("Gokapi v" + VERSION + " starting")
	createDataDir()
	loadConfig()
	checkArguments()
	initTemplates()
	go cleanUpOldFiles(true)
	startWebserver()
}

// Checks for command line arguments that have to be parsed before loading the configuration
func checkPrimaryArguments() {
	if len(os.Args) > 1 {
		if os.Args[1] == "--version" || os.Args[1] == "-v" {
			fmt.Println("Gokapi v" + VERSION)
			fmt.Println("Builder: " + BUILDER)
			fmt.Println("Build Date: " + BUILD_TIME)
			fmt.Println("Docker Version: " + IS_DOCKER)
			os.Exit(0)
		}
	}
}

// Checks for command line arguments that have to be parsed after loading the configuration
func checkArguments() {
	if len(os.Args) > 1 {
		if os.Args[1] == "--reset-pw" {
			fmt.Println("Password change requested")
			globalConfig.AdminPassword = hashPassword(askForPassword(), SALT_PW_ADMIN)
			saveConfig()
		}
	}
}
