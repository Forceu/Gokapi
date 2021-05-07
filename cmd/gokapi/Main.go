package main

/**
Main routine
*/

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/environment"
	"Gokapi/internal/storage"
	"Gokapi/internal/webserver"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"
)

// Version is the current version in readable form.
// The go generate call below needs to be modified as well
const Version = "1.2.0"

//go:generate sh "../../build/setVersionTemplate.sh" "1.2.0"

// Main routine that is called on startup
func main() {
	passedFlags := parseFlags()
	showVersion(passedFlags)
	rand.Seed(time.Now().UnixNano())
	fmt.Println(logo)
	fmt.Println("Gokapi v" + Version + " starting")
	configuration.Load()
	resetPassword(passedFlags)
	go storage.CleanUp(true)
	webserver.Start()
}

// Checks for command line arguments that have to be parsed before loading the configuration
func showVersion(passedFlags flags) {
	if passedFlags.showVersion {
		fmt.Println("Gokapi v" + Version)
		fmt.Println("Builder: " + environment.Builder)
		fmt.Println("Build Date: " + environment.BuildTime)
		fmt.Println("Docker Version: " + environment.IsDocker)
		os.Exit(0)
	}
}

func parseFlags() flags {
	versionShortFlag := flag.Bool("v", false, "Show version info")
	versionLongFlag := flag.Bool("version", false, "Show version info")
	resetPwFlag := flag.Bool("reset-pw", false, "Show prompt to reset admin password")
	flag.Parse()
	return flags{
		showVersion: *versionShortFlag || *versionLongFlag,
		resetPw:     *resetPwFlag,
	}
}

// Checks for command line arguments that have to be parsed after loading the configuration
func resetPassword(passedFlags flags) {
	if passedFlags.resetPw {
		fmt.Println("Password change requested")
		configuration.DisplayPasswordReset()
		fmt.Println("Password has been changed!")
		os.Exit(0)
	}
}

type flags struct {
	showVersion bool
	resetPw     bool
}

// ASCII art logo
const logo = `
██████   ██████  ██   ██  █████  ██████  ██ 
██       ██    ██ ██  ██  ██   ██ ██   ██ ██ 
██   ███ ██    ██ █████   ███████ ██████  ██ 
██    ██ ██    ██ ██  ██  ██   ██ ██      ██ 
 ██████   ██████  ██   ██ ██   ██ ██      ██ 
                                             `

// Generates coverage badge
//go:generate sh "../../build/updateCoverage.sh"

// Copy go mod file to docker image builder
//go:generate cp "../../go.mod" "../../build/go.mod"
