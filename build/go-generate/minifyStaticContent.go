//go:build gogenerate

package main

import (
	"fmt"
	minify "github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/js"
	"os"
	"path/filepath"
	"strconv"
)

const pathPrefix = "../../internal/webserver/web/static/"

type converter struct {
	InputPath       string
	OutputPath      string
	PreviousVersion string
	Type            string
	Name            string
}

func main() {
	for _, f := range getPaths() {
		minifyContent(f)
	}
}

func getPaths() []converter {
	var result []converter
	result = append(result, converter{
		InputPath:       pathPrefix + "css/*.css",
		OutputPath:      pathPrefix + "css/min/gokapi.min." + strconv.Itoa(cssMainVersion) + ".css",
		PreviousVersion: pathPrefix + "css/min/gokapi.min." + strconv.Itoa(cssMainVersion-1) + ".css",
		Type:            "text/css",
		Name:            "Main CSS",
	})
	result = append(result, converter{
		InputPath:       pathPrefix + "js/admin_*.js",
		OutputPath:      pathPrefix + "js/min/admin.min." + strconv.Itoa(jsAdminVersion) + ".js",
		PreviousVersion: pathPrefix + "js/min/admin.min." + strconv.Itoa(jsAdminVersion-1) + ".js",
		Type:            "text/javascript",
		Name:            "Admin JS",
	})
	result = append(result, converter{
		InputPath:       pathPrefix + "js/end2end_admin.js",
		OutputPath:      pathPrefix + "js/min/end2end_admin.min." + strconv.Itoa(jsE2EVersion) + ".js",
		PreviousVersion: pathPrefix + "js/min/end2end_admin.min." + strconv.Itoa(jsE2EVersion-1) + ".js",
		Type:            "text/javascript",
		Name:            "Admin E2E JS",
	})
	result = append(result, converter{
		InputPath:       pathPrefix + "js/end2end_download.js",
		OutputPath:      pathPrefix + "js/min/end2end_download.min." + strconv.Itoa(jsE2EVersion) + ".js",
		PreviousVersion: pathPrefix + "js/min/end2end_download.min." + strconv.Itoa(jsE2EVersion-1) + ".js",
		Type:            "text/javascript",
		Name:            "Download E2E JS",
	})
	result = append(result, converter{
		InputPath:  pathPrefix + "js/streamsaver.js",
		OutputPath: pathPrefix + "js/min/streamsaver.min.js",
		Type:       "text/javascript",
		Name:       "Streamsaver JS",
	})
	result = append(result, converter{
		InputPath:  pathPrefix + "js/wasm_exec.js",
		OutputPath: pathPrefix + "js/min/wasm_exec.min.js",
		Type:       "text/javascript",
		Name:       "wasm_exec JS",
	})
	return result
}

func minifyContent(conv converter) {
	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("text/javascript", js.Minify)

	files, err := m.Bytes(conv.Type, getAllFiles(conv.InputPath))
	if err != nil {
		fmt.Println(err)
		os.Exit(4)
	}
	err = os.WriteFile(conv.OutputPath, files, 0664)
	if err != nil {
		fmt.Println("Could not write " + conv.Name + " files")
		fmt.Println(err)
		os.Exit(5)
	}
	fmt.Println("Minified " + conv.Name)
	if conv.PreviousVersion != "" && fileExists(conv.PreviousVersion) {
		fmt.Println("Removing old version of " + conv.Name)
		err = os.Remove(conv.PreviousVersion)
		if err != nil {
			fmt.Println("Could not remove old " + conv.Name + " file")
			fmt.Println(err)
			os.Exit(6)
		}
	}
}

func getAllFiles(pattern string) []byte {
	var result, content []byte
	matches, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Println("Bad pattern:")
		fmt.Println(err)
		os.Exit(1)
	}
	if len(matches) < 1 {
		fmt.Println("No files found for minifying. Pattern: " + pattern)
		os.Exit(2)
	}

	for _, fpath := range matches {
		content, err = os.ReadFile(fpath)
		if err != nil {
			fmt.Println("Could not read file")
			fmt.Println(err)
			os.Exit(3)
		}
		result = append(result, content...)
	}
	return result
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// Auto-generated content below, do not modify
// Version codes can be changed in updateVersionNumbers.go

const jsAdminVersion = 9
const jsE2EVersion = 5
const cssMainVersion = 4
