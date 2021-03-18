package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"
)

//needs to be changed in ./templates/string_constants.tmpl as well
const VERSION = "1.1.0"

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

func checkArguments() {
	if len(os.Args) > 1 {
		if os.Args[1] == "--reset-pw" {
			fmt.Println("Password change requested")
			globalConfig.AdminPassword = hashPassword(askForPassword(), SALT_PW_ADMIN)
			saveConfig()
		}
	}
}
