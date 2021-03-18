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
	rand.Seed(time.Now().UnixNano())
	fmt.Println("Gokapi v" + VERSION + " starting")
	createDataDir()
	loadConfig()
	checkArguments()
	initTemplates()
	go cleanUpOldFiles(true)
	startWebserver()
}

func checkArguments() {
	if len(os.Args) > 1 {
		if os.Args[1] == "--reset-pw" {
			fmt.Println("Password change requested")
			globalConfig.AdminPassword = hashPassword(askForPassword(),SALT_PW_ADMIN)
			saveConfig()
		}
	}
}
