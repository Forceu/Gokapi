package main

/**
Main routine
*/

import (
	"Gokapi/src/configuration"
	"Gokapi/src/environment"
	"Gokapi/src/storage"
	"Gokapi/src/webserver"
	"embed"
	"fmt"
	"math/rand"
	"os"
	"time"
)

// needs to be changed in ./templates/string_constants.tmpl as well
const VERSION = "1.1.2"

// Main routine that is called on startup
func main() {
	checkPrimaryArguments()
	rand.Seed(time.Now().UnixNano())
	fmt.Println(logo)
	fmt.Println("Gokapi v" + VERSION + " starting")
	configuration.Load()
	checkArguments()
	go storage.CleanUp(true)
	webserver.Start(staticFolderEmbedded, templateFolderEmbedded)
}

// Checks for command line arguments that have to be parsed before loading the configuration
func checkPrimaryArguments() {
	if len(os.Args) > 1 {
		if os.Args[1] == "--version" || os.Args[1] == "-v" {
			fmt.Println("Gokapi v" + VERSION)
			fmt.Println("Builder: " + environment.Builder)
			fmt.Println("Build Date: " + environment.BuildTime)
			fmt.Println("Docker Version: " + environment.IsDocker)
			os.Exit(0)
		}
	}
}

// Checks for command line arguments that have to be parsed after loading the configuration
func checkArguments() {
	if len(os.Args) > 1 {
		if os.Args[1] == "--reset-pw" {
			fmt.Println("Password change requested")
			configuration.DisplayPasswordReset()
		}
	}
}

// ASCII art logo
const logo = ` ██████   ██████  ██   ██  █████  ██████  ██ 
██       ██    ██ ██  ██  ██   ██ ██   ██ ██ 
██   ███ ██    ██ █████   ███████ ██████  ██ 
██    ██ ██    ██ ██  ██  ██   ██ ██      ██ 
 ██████   ██████  ██   ██ ██   ██ ██      ██ 
                                             `

// Embedded version of the "static" folder
// This contains JS files, CSS, images etc
//go:embed static
var staticFolderEmbedded embed.FS

// Embedded version of the "templates" folder
// This contains templates that Gokapi uses for creating the HTML output
//go:embed templates
var templateFolderEmbedded embed.FS
