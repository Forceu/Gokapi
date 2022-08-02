package main

import (
	"encoding/json"
	"fmt"
	"github.com/forceu/gokapi/internal/configuration/database"
	"log"
	"os"
)

func main() {
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
