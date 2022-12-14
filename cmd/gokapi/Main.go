package main

/**
Main routine
*/

import (
	"fmt"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/cloudconfig"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/configuration/setup"
	"github.com/forceu/gokapi/internal/encryption"
	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/environment/flagparser"
	"github.com/forceu/gokapi/internal/logging"
	"github.com/forceu/gokapi/internal/storage"
	"github.com/forceu/gokapi/internal/storage/cloudstorage/aws"
	"github.com/forceu/gokapi/internal/webserver"
	"github.com/forceu/gokapi/internal/webserver/authentication"
	"github.com/forceu/gokapi/internal/webserver/ssl"
	"math/rand"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"
)

// Version is the current version in readable form.
// The go generate call below needs to be modified as well
const Version = "1.6.2"

//go:generate sh "../../build/setVersionTemplate.sh" "1.6.1"
//go:generate sh -c "cp \"$(go env GOROOT)/misc/wasm/wasm_exec.js\" ../../internal/webserver/web/static/js/ && echo Copied wasm_exec.js"
//go:generate sh -c "GOOS=js GOARCH=wasm go build -o ../../internal/webserver/web/main.wasm github.com/forceu/gokapi/cmd/wasmdownloader && echo Compiled Downloader WASM module"
//go:generate sh -c "GOOS=js GOARCH=wasm go build -o ../../internal/webserver/web/e2e.wasm github.com/forceu/gokapi/cmd/wasme2e && echo Compiled E2E WASM module"

// Main routine that is called on startup
func main() {
	passedFlags := flagparser.ParseFlags()
	showVersion(passedFlags)
	rand.Seed(time.Now().UnixNano())
	fmt.Println(logo)
	fmt.Println("Gokapi v" + Version + " starting")
	setup.RunIfFirstStart()
	configuration.Load()
	reconfigureServer(passedFlags)
	encryption.Init(*configuration.Get())
	authentication.Init(configuration.Get().Authentication)
	createSsl(passedFlags)
	initCloudConfig(passedFlags)
	go storage.CleanUp(true)
	logging.AddString("Gokapi started")
	go webserver.Start()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	shutdown()
	os.Exit(0)
}

func shutdown() {
	fmt.Println("Shutting down...")
	webserver.Shutdown()
	database.Close()
}

// Checks for command line arguments that have to be parsed before loading the configuration
func showVersion(passedFlags flagparser.MainFlags) {
	if passedFlags.ShowVersion {
		fmt.Println("Gokapi v" + Version)
		fmt.Println()
		fmt.Println("Builder: " + environment.Builder)
		fmt.Println("Build Date: " + environment.BuildTime)
		fmt.Println("Docker Version: " + environment.IsDocker)
		info, ok := debug.ReadBuildInfo()
		if ok {
			fmt.Println("Go Version: " + info.GoVersion)
		} else {
			fmt.Println("Go Version: unknown")
		}
		parseBuildSettings(info.Settings)
		osExit(0)
	}
}

func parseBuildSettings(infos []debug.BuildSetting) {
	lookups := make(map[string]string)
	lookups["-tags"] = "Build Tags"
	lookups["vcs.revision"] = "Git Commit"
	lookups["vcs.time"] = "Git Commit Timestamp"
	lookups["GOARCH"] = "Architecture"
	lookups["GOOS"] = "Operating System"

	for key, value := range lookups {
		result := "Not found"
		for _, buildSetting := range infos {
			if buildSetting.Key == key {
				result = buildSetting.Value
				break
			}
		}
		fmt.Println(value + ": " + result)
	}

	for _, info := range infos {
		if info.Key == "vcs.modified" {
			if info.Value == "true" {
				fmt.Println("Code has been modified after last git commit")
			}
			break
		}
	}
}

func initCloudConfig(passedFlags flagparser.MainFlags) {
	cConfig, ok := cloudconfig.Load()
	if ok && aws.Init(cConfig.Aws) {
		fmt.Println("Saving new files to cloud storage")
		encLevel := configuration.Get().Encryption.Level
		if (encLevel == encryption.FullEncryptionStored || encLevel == encryption.FullEncryptionInput) && !passedFlags.DisableCorsCheck {
			ok, err := aws.IsCorsCorrectlySet(cConfig.Aws.Bucket, configuration.Get().ServerUrl)
			if err != nil {
				fmt.Println("Warning: Cannot check CORS settings. " + err.Error())
				fmt.Println("If your provider does not implement the CORS API call and you are certain that it is set correctly, you can disable this check with --disable-cors-check")
			} else {
				if !ok {
					fmt.Println("Warning: CORS settings for bucket " + cConfig.Aws.Bucket + " might not be set correctly. Download might not be possible with encryption.")
				}
			}
		}
	} else {
		fmt.Println("Saving new files to local storage")
	}
}

// Checks for command line arguments that have to be parsed after loading the configuration
func reconfigureServer(passedFlags flagparser.MainFlags) {
	if passedFlags.Reconfigure {
		setup.RunConfigModification()
	}
}

func createSsl(passedFlags flagparser.MainFlags) {
	if passedFlags.CreateSsl {
		ssl.GenerateIfInvalidCert(configuration.Get().ServerUrl, true)
	}
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

// Copy go mod file to docker image builder
//go:generate cp "../../go.mod" "../../build/go.mod"
//go:generate echo "Copied go.mod to Docker build directory"
