//go:build gogenerate

package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const versionJsAdmin = 9
const versionJsDropzone = 5
const versionJsE2EAdmin = 5
const versionCssMain = 4

const fileMain = "../../cmd/gokapi/Main.go"
const fileMinify = "../../build/go-generate/minifyStaticContent.go"
const fileVersionConstants = "../../internal/webserver/web/templates/string_constants.tmpl"

func main() {
	checkFileExists(fileMain)
	checkFileExists(fileMinify)
	checkFileExists(fileVersionConstants)
	writeVersionTemplates()
	writeMinify()
}

func writeVersionTemplates() {
	template := insertVersionNumbers(templateVersions)
	err := os.WriteFile(fileVersionConstants, []byte(template), 0664)
	if err != nil {
		fmt.Println("FAIL: Updating version template")
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Updated version template")
}

func writeMinify() {
	file, err := os.Open(fileMinify)
	if err != nil {
		fmt.Println("FAIL: Opening minify go file")
		fmt.Println(err)
		os.Exit(1)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	foundAutoGencomment := false
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		if strings.Contains(line, autoGenComment) {
			foundAutoGencomment = true
			break
		}
	}
	err = scanner.Err()
	if err != nil {
		fmt.Println("FAIL: Reading minify go file")
		fmt.Println(err)
		os.Exit(1)
	}

	if !foundAutoGencomment {
		fmt.Println("FAIL: Minify go file did not contain auto-gen comment")
		fmt.Println(err)
		os.Exit(2)
	}
	lines = append(lines, insertVersionNumbers(templateMinify))

	err = os.WriteFile(fileMinify, []byte(strings.Join(lines, "\n")), 0664)
	if err != nil {
		fmt.Println("FAIL: Wrining minify go file")
		fmt.Println(err)
		os.Exit(3)
	}
	fmt.Println("Updated minify go file")
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

func insertVersionNumbers(input string) string {
	versionGokapi := parseGokapiVersion()
	result := strings.ReplaceAll(input, "%gokapiversion%", versionGokapi)
	result = strings.ReplaceAll(result, "%jsadmin%", strconv.Itoa(versionJsAdmin))
	result = strings.ReplaceAll(result, "%jsdropzone%", strconv.Itoa(versionJsDropzone))
	result = strings.ReplaceAll(result, "%jse2e%", strconv.Itoa(versionJsE2EAdmin))
	result = strings.ReplaceAll(result, "%css_main%", strconv.Itoa(versionCssMain))
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

const templateVersions = `// File contains auto-generated values. Do not change manually
{{define "version"}}%gokapiversion%{{end}}

// Specifies the version of JS files, so that the browser doesn't
// use a cached version, if the file has been updated
{{define "js_admin_version"}}%jsadmin%{{end}}
{{define "js_dropzone_version"}}%jsdropzone%{{end}}
{{define "js_e2eversion"}}%jse2e%{{end}}
{{define "css_main"}}%css_main%{{end}}`

const autoGenComment = "// Auto-generated content below, do not modify"
const templateMinify = `// Version codes can be changed in updateVersionNumbers.go

const jsAdminVersion = %jsadmin%
const jsE2EVersion = %jse2e%
const cssMainVersion = %css_main%
`
