//go:build gogenerate

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type githubRelease struct {
	URL         string    `json:"url"`
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at"`
	Body        string    `json:"body"`
}

func (r githubRelease) ToString() string {
	var sb strings.Builder
	r.Body = strings.Replace(r.Body, "\r\n", "\n", -1)
	r.Body = strings.Replace(r.Body, "`", "_backtick_", -1)
	r.Body = strings.Replace(r.Body, "_backtick_", "``", -1)
	r.Body = strings.Replace(r.Body, ":warning:", ".. warning::\n    ", -1)
	r.Body = convertMarkdownLinksToReST(r.Body)
	r.Body = insertMissingNewlineForSublist(r.Body)
	r.Body = insertIssueUrl(r.Body)

	sb.WriteString(r.TagName)
	sb.WriteString(" (")
	if r.Draft {
		sb.WriteString(time.Now().Format("2006-01-02"))
	} else {
		sb.WriteString(r.PublishedAt.Format("2006-01-02"))
	}
	sb.WriteString(")\n")
	sb.WriteString(strings.Repeat("^", sb.Len()-1))
	sb.WriteString("\n")
	sb.WriteString("\n")
	for _, line := range strings.Split(r.Body, "\n") {
		if strings.HasPrefix(line, "### ") {
			line = strings.TrimPrefix(line, "### ")
			sb.WriteString(line)
			sb.WriteString("\n")
			sb.WriteString(strings.Repeat("\"", len(line)))
			sb.WriteString("\n")
			continue
		}
		if strings.HasPrefix(line, "#### ") {
			line = strings.TrimPrefix(line, "#### ")
			sb.WriteString(line)
			sb.WriteString("\n")
			sb.WriteString(strings.Repeat("'", len(line)))
			sb.WriteString("\n")
			continue
		}
		if strings.HasPrefix(line, "- ") {
			line = strings.Replace(line, "- ", "* ", 1)
		}
		if strings.HasPrefix(line, "  - ") {
			line = strings.Replace(line, "  - ", "  * ", 1)
		}
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString("\n")
	return sb.String()
}

func convertMarkdownLinksToReST(text string) string {
	// Regex to match Markdown links: [link text](url)
	re := regexp.MustCompile(`\[(.*?)\]\((https?://[^\s\)]+)\)`)
	return re.ReplaceAllString(text, "`$1 <$2>`__")
}

func insertIssueUrl(text string) string {
	// Regex to match Markdown links: [link text](url)
	re := regexp.MustCompile(`#(\d+)`)
	return re.ReplaceAllString(text, "`#$1 <https://github.com/Forceu/Gokapi/issues/$1>`__")
}

func insertMissingNewlineForSublist(text string) string {
	re := regexp.MustCompile(`(?m)^(- [^\n]+?)(\n)(  - )`)
	return re.ReplaceAllString(text, "$1\n\n$3")
}

const fileChangelog = "./docs/changelog.rst"
const fileGithubSecret = "./build/go-generate/github.secret"

func main() {
	checkFileExistChangelog(fileChangelog)
	checkFileExistChangelog(fileGithubSecret)

	updateChangelog()

}

func checkFileExistChangelog(filename string) {
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

func updateChangelog() {
	secret := getSecret()
	releases := loadGithubReleases(secret)

	var sb strings.Builder
	sb.WriteString(headerChangelog)
	for _, release := range releases {
		if !release.Prerelease {
			sb.WriteString(release.ToString())
		}
	}
	err := os.WriteFile(fileChangelog, []byte(sb.String()), 0664)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Updated changelog")
}

func getSecret() string {
	content, err := os.ReadFile(fileGithubSecret)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return strings.TrimSpace(string(content))
}

func loadGithubReleases(secret string) []githubRelease {
	const url = "https://api.github.com/repos/Forceu/Gokapi/releases"

	spaceClient := http.Client{
		Timeout: time.Second * 20,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	req.Header.Set("User-Agent", "gokapi-changelog-updater")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Authorization", "Bearer "+secret)

	res, err := spaceClient.Do(req)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	if res.StatusCode != 200 {
		fmt.Println("ERROR: HTTP status code: " + res.Status)
		fmt.Println("Maybe token is incorrect or has expired?")
		os.Exit(3)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	var result []githubRelease
	err = json.Unmarshal(body, &result)
	if err != nil {
		fmt.Println(err)
		os.Exit(3)
	}
	return result
}

const headerChangelog = `.. _changelog:


Changelog
=========

Overview of all changes
-----------------------


`
