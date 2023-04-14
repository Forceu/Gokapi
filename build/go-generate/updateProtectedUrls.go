//go:build gogenerate

package main

import (
	"fmt"
	"golang.org/x/exp/slices"
	"os"
	"regexp"
	"strings"
)

const fileSetup = "../../internal/webserver/Webserver.go"
const fileSetupConstants = "../../internal/configuration/setup/ProtectedUrls.go"
const fileDocumentation = "../../docs/setup.rst"

func main() {
	checkFileExistsUrl(fileSetup)
	checkFileExistsUrl(fileSetupConstants)
	checkFileExistsUrl(fileDocumentation)
	urls := parseProtectedUrls()
	writeConstantFile(urls)
	writeDocumentationFile(urls)
}

func checkFileExistsUrl(filename string) {
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

func parseProtectedUrls() []string {
	source, err := os.ReadFile(fileSetup)
	if err != nil {
		fmt.Println("ERROR: Cannot read file: ")
		fmt.Println(err)
		os.Exit(4)
	}
	urls := make([]string, 0)
	regex := regexp.MustCompile(`mux\.HandleFunc\("([^"]+)",\s*requireLogin\(`)
	matches := regex.FindAllStringSubmatch(string(source), -1)
	for _, match := range matches {
		fn := strings.TrimSpace(match[1])
		urls = append(urls, fn)
	}
	if len(urls) < 4 {
		fmt.Println("ERROR: Could not find protected URLs")
		os.Exit(5)
	}
	return urls
}

func writeConstantFile(urls []string) {
	var output = `package setup

// Do not modify: This is an automatically generated File.
// It contains all URLs that need to be protected when using an external authentication.

// protectedUrls contains a list of URLs that need to be protected if authentication is disabled.
// This list will be displayed during the setup
var protectedUrls = []string{`

	slices.Sort(urls)
	for i, url := range urls {
		output = output + "\"" + url + "\""
		if i < len(urls)-1 {
			output = output + ", "
		} else {
			output = output + "}\n"
		}
	}
	err := os.WriteFile(fileSetupConstants, []byte(output), 0664)
	if err != nil {
		fmt.Println("ERROR: Cannot write file:")
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Updated protected URLs variable")
}

func writeDocumentationFile(urls []string) {
	documentationContent, err := os.ReadFile(fileDocumentation)
	if err != nil {
		fmt.Println("ERROR: Cannot read file:")
		fmt.Println(err)
		os.Exit(6)
	}
	output := "proxy:\n\n"
	for _, url := range urls {
		output = output + "- ``" + url + "``\n"
	}
	regex := regexp.MustCompile(`proxy:\n+((?:- ` + "``" + `\/\w+` + "``" + `\n)+)`)
	matches := regex.FindAllIndex(documentationContent, -1)
	if len(matches) != 1 {
		fmt.Println("ERROR: Not one match found exactly for documentation")
		os.Exit(7)
	}
	documentationContent = regex.ReplaceAll(documentationContent, []byte(output))
	err = os.WriteFile(fileDocumentation, documentationContent, 0664)
	if err != nil {
		fmt.Println("ERROR: Cannot write file:")
		fmt.Println(err)
		os.Exit(8)
	}
}
