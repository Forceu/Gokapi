//go:build gogenerate

package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const versionJsAdmin = "1"
const versionJsDropzone = "3"
const versionJsE2EAdmin = "2"
const versionCssMain = "1"

const fileMain = "../../cmd/gokapi/Main.go"
const fileVersionConstants = "../../internal/webserver/web/templates/string_constants.tmpl"

func main() {
	checkFileExists(fileMain)
	checkFileExists(fileVersionConstants)
	template := getTemplate()
	err := os.WriteFile(fileVersionConstants, []byte(template), 0664)
	if err != nil {
		fmt.Println("FAIL: Updating version numbers")
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Updated version numbers")
}

func checkFileExists(filename string) {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		fmt.Println("ERROR: File does not exist: " + filename)
		os.Exit(2)
	}
	if info.IsDir() {
		fmt.Println("ERROR: File is actually directory: " + filename)
		os.Exit(3)
	}
}

func getTemplate() string {
	versionGokapi := parseGokapiVersion()
	result := strings.ReplaceAll(templateVersions, "%gokapiversion%", versionGokapi)
	result = strings.ReplaceAll(result, "%jsadmin%", versionJsAdmin)
	result = strings.ReplaceAll(result, "%jsdropzone%", versionJsDropzone)
	result = strings.ReplaceAll(result, "%jse2e%", versionJsE2EAdmin)
	result = strings.ReplaceAll(result, "%css_main%", versionCssMain)
	return result
}

func parseGokapiVersion() string {
	file, err := os.Open(fileMain)
	if err != nil {
		fmt.Println("ERROR: Cannot open file: ")
		fmt.Println(err)
		os.Exit(4)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	constRegex := regexp.MustCompile(`const\s+versionGokapi\s+=\s+"(\S+)"`)
	for scanner.Scan() {
		line := scanner.Text()
		matches := constRegex.FindStringSubmatch(line)
		if len(matches) == 2 {
			return matches[1]

		}
	}
	fmt.Println("ERROR: Gokapi version not found")
	os.Exit(5)
	return ""
}

const templateVersions = `// Change these for rebranding
{{define "app_name"}}Gokapi{{end}}
{{define "version"}}%gokapiversion%{{end}}

// Specifies the version of JS files, so that the browser doesn't
// use a cached version, if the file has been updated
{{define "js_admin_version"}}%jsadmin%{{end}}
{{define "js_dropzone_version"}}%jsdropzone%{{end}}
{{define "js_e2eversion"}}%jse2e%{{end}}
{{define "css_main"}}%css_main%{{end}}`
