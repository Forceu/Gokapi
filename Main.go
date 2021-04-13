package main

/**
Main routine
*/

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/environment"
	"Gokapi/internal/storage"
	"Gokapi/internal/webserver"
	"embed"
	"fmt"
	"math/rand"
	"os"
	"time"
)

// Version is the current version in readable form.
// The go generate call below needs to be modified as well
const Version = "1.1.4-dev"

//go:generate sh "./build/setVersionTemplate.sh" "1.1.4-dev"

// Main routine that is called on startup
func main() {
	checkPrimaryArguments()
	rand.Seed(time.Now().UnixNano())
	fmt.Println(logo)
	fmt.Println("Gokapi v" + Version + " starting")
	configuration.Load()
	checkArguments()
	go storage.CleanUp(true)
	webserver.Start(&StaticFolderEmbedded, &TemplateFolderEmbedded, true)
}

// Checks for command line arguments that have to be parsed before loading the configuration
func checkPrimaryArguments() {
	if len(os.Args) > 1 {
		if os.Args[1] == "--version" || os.Args[1] == "-v" {
			fmt.Println("Gokapi v" + Version)
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
			fmt.Println("Password has been changed!")
			os.Exit(0)
		}
	}
}

// StaticFolderEmbedded is the embedded version of the "static" folder
// This contains JS files, CSS, images etc
//go:embed web/static
var StaticFolderEmbedded embed.FS

// TemplateFolderEmbedded is the embedded version of the "templates" folder
// This contains templates that Gokapi uses for creating the HTML output
//go:embed web/templates
var TemplateFolderEmbedded embed.FS

// ASCII art logo
const logo = `
██████   ██████  ██   ██  █████  ██████  ██ 
██       ██    ██ ██  ██  ██   ██ ██   ██ ██ 
██   ███ ██    ██ █████   ███████ ██████  ██ 
██    ██ ██    ██ ██  ██  ██   ██ ██      ██ 
 ██████   ██████  ██   ██ ██   ██ ██      ██ 
                                             `
