package main

import (
	"encoding/json"
	"fmt"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/helper"
	"log"
	"os"
)

func main() {
	if !correctArgs() {
		showUsageAndExit()
		return
	}
	path := os.Args[1]
	database.Init(path)
	metadata := database.GetAllMetadata()
	for _, file := range metadata {
		result, err := json.MarshalIndent(file, "", "  ")
		if err != nil {
			log.Fatal("Error encoding file: ", err)
		}
		fmt.Println(string(result))
		fmt.Println()
	}
}

func correctArgs() bool {
	if len(os.Args) < 2 {
		return false
	}
	path := os.Args[1]
	if path == "" {
		return false
	}
	if !helper.FolderExists(path) {
		fmt.Println("Error: Folder does not exist: " + path)
		return false
	}
	return true
}

func showUsageAndExit() {
	fmt.Println("Usage: ./databasereader /path/to/database")
	osExit(1)
	return
}

var osExit = os.Exit
