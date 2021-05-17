package main

/**
Main routine
*/

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/configuration/cloudconfig"
	"Gokapi/internal/environment"
	"Gokapi/internal/helper"
	"Gokapi/internal/storage"
	"Gokapi/internal/storage/cloudstorage/aws"
	"Gokapi/internal/webserver"
	"Gokapi/internal/webserver/ssl"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"
)

// Version is the current version in readable form.
// The go generate call below needs to be modified as well
const Version = "1.3.0"

//go:generate sh "../../build/setVersionTemplate.sh" "1.3.0"

// Main routine that is called on startup
func main() {
	passedFlags := parseFlags()
	showVersion(passedFlags)
	rand.Seed(time.Now().UnixNano())
	fmt.Println(logo)
	fmt.Println("Gokapi v" + Version + " starting")
	configuration.Load()
	resetPassword(passedFlags)
	createSsl(passedFlags)

	cConfig, ok := cloudconfig.Load()
	if ok && aws.Init(cConfig.Aws) {
		fmt.Println("Saving new files to cloud storage")
	} else {
		fmt.Println("Saving new files to local storage")
	}
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
		osExit(0)
	}
}

func parseFlags() flags {
	passedFlags := flag.FlagSet{}
	versionShortFlag := passedFlags.Bool("v", false, "Show version info")
	versionLongFlag := passedFlags.Bool("version", false, "Show version info")
	resetPwFlag := passedFlags.Bool("reset-pw", false, "Show prompt to reset admin password")
	createSslFlag := passedFlags.Bool("create-ssl", false, "Creates a new SSL certificate valid for 365 days")
	err := passedFlags.Parse(os.Args[1:])
	helper.Check(err)
	return flags{
		showVersion: *versionShortFlag || *versionLongFlag,
		resetPw:     *resetPwFlag,
		createSsl:   *createSslFlag,
	}
}

// Checks for command line arguments that have to be parsed after loading the configuration
func resetPassword(passedFlags flags) {
	if passedFlags.resetPw {
		fmt.Println("Password change requested")
		configuration.DisplayPasswordReset()
		fmt.Println("Password has been changed!")
		osExit(0)
	}
}

func createSsl(passedFlags flags) {
	if passedFlags.createSsl {
		settings := configuration.GetServerSettings()
		configuration.Release()
		ssl.GenerateIfInvalidCert(settings.ServerUrl, true)
	}
}

type flags struct {
	showVersion bool
	resetPw     bool
	createSsl   bool
}

var osExit = os.Exit

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
//go:generate echo "Copied go.mod to Docker build directory"
