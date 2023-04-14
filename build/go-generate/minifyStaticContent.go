//go:build gogenerate

package main

import (
	"fmt"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/js"
	"os"
	"path/filepath"
)

const pathPrefix = "../../internal/webserver/web/static/"

type converter struct {
	InputPath  string
	OutputPath string
	Type       string
	Name       string
}

func main() {
	for _, f := range getPaths() {
		minifyContent(f)
	}
}

func getPaths() []converter {
	var result []converter
	result = append(result, converter{
		InputPath:  pathPrefix + "css/*.css",
		OutputPath: pathPrefix + "css/min/gokapi.min.css",
		Type:       "text/css",
		Name:       "Main CSS",
	})
	result = append(result, converter{
		InputPath:  pathPrefix + "js/admin.js",
		OutputPath: pathPrefix + "js/min/admin.min.js",
		Type:       "text/javascript",
		Name:       "Admin JS",
	})
	result = append(result, converter{
		InputPath:  pathPrefix + "js/end2end_admin.js",
		OutputPath: pathPrefix + "js/min/end2end_admin.min.js",
		Type:       "text/javascript",
		Name:       "Admin E2E JS",
	})
	result = append(result, converter{
		InputPath:  pathPrefix + "js/end2end_download.js",
		OutputPath: pathPrefix + "js/min/end2end_download.min.js",
		Type:       "text/javascript",
		Name:       "Download E2E JS",
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
