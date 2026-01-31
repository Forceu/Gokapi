//go:build gogenerate

package main

import (
	"fmt"
	"os"
	"path/filepath"

	minify "github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/js"
)

const pathPrefix = "../../internal/webserver/web/static/"

type converter struct {
	InputPath         string
	OutputPath        string
	GlobPreviousFiles string
	Type              string
	Name              string
}

func main() {
	for _, f := range getPaths() {
		minifyContent(f)
	}
}

func getPaths() []converter {
	var result []converter
	result = append(result, converter{
		InputPath:         pathPrefix + "css/*.css",
		OutputPath:        pathPrefix + "css/min/gokapi.min." + cssMainVersion + ".css",
		GlobPreviousFiles: pathPrefix + "css/min/gokapi.min.*.css",
		Type:              "text/css",
		Name:              "Main CSS",
	})
	result = append(result, converter{
		InputPath:         pathPrefix + "js/admin_*.js",
		OutputPath:        pathPrefix + "js/min/admin.min." + jsAdminVersion + ".js",
		GlobPreviousFiles: pathPrefix + "js/min/admin.min.*.js",
		Type:              "text/javascript",
		Name:              "Admin JS",
	})
	result = append(result, converter{
		InputPath:         pathPrefix + "js/end2end_admin.js",
		OutputPath:        pathPrefix + "js/min/end2end_admin.min." + jsE2EVersion + ".js",
		GlobPreviousFiles: pathPrefix + "js/min/end2end_admin.min.*.js",
		Type:              "text/javascript",
		Name:              "Admin E2E JS",
	})
	result = append(result, converter{
		InputPath:         pathPrefix + "js/end2end_download.js",
		OutputPath:        pathPrefix + "js/min/end2end_download.min." + jsE2EVersion + ".js",
		GlobPreviousFiles: pathPrefix + "js/min/end2end_download.min.*.js",
		Type:              "text/javascript",
		Name:              "Download E2E JS",
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
	result = append(result, converter{
		InputPath:  pathPrefix + "js/all_public.js",
		OutputPath: pathPrefix + "js/min/all_public.min.js",
		Type:       "text/javascript",
		Name:       "Public functions JS",
	})
	result = append(result, converter{
		InputPath:  pathPrefix + "js/public_upload.js",
		OutputPath: pathPrefix + "js/min/public_upload.min.js",
		Type:       "text/javascript",
		Name:       "Public upload JS",
	})
	return result
}

func minifyContent(conv converter) {
	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("text/javascript", js.Minify)

	if conv.GlobPreviousFiles != "" {
		removeOldFiles(conv.GlobPreviousFiles)
	}

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
}

func removeOldFiles(pattern string) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		panic(err)
	}

	if len(matches) == 0 {
		return
	}

	if len(matches) > 1 {
		fmt.Println("Multiple matching files found for " + pattern + ", refusing to delete")
		os.Exit(6)
	}

	err = os.Remove(matches[0])
	if err != nil {
		panic(err)
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

const jsAdminVersion = "485eaf17ab"
const jsE2EVersion = "485eaf17ab"
const cssMainVersion = "485eaf17ab"
