//go:build gogenerate

package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

const fileEnvVariable = "../../internal/environment/Environment.go"
const fileDocumentationAdvanced = "../../docs/advanced.rst"

func main() {
	checkFileExistsEnv(fileEnvVariable)
	checkFileExistsEnv(fileDocumentationAdvanced)
	vars, err := extractEnvVars()
	if err != nil {
		fmt.Println("ERROR: Cannot extract env vars: ")
		fmt.Println(err)
		os.Exit(5)
	}
	writeEnvDocumentationFile(vars)
}

func checkFileExistsEnv(filename string) {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		fmt.Println("ERROR: File does not exist: " + filename)
		os.Exit(2)
	}
	if info.IsDir() {
		fmt.Println("ERROR: File is actually directory: " + filename)
		os.Exit(3)
	}
	if err != nil {
		fmt.Println("ERROR: Cannot read file: ")
		fmt.Println(err)
		os.Exit(4)
	}
}

type envVar struct {
	Name       string
	Action     string
	Persistent bool
	Default    string
}

func extractEnvVars() ([]envVar, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, fileEnvVariable, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var result []envVar

	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}

		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != "Environment" {
				continue
			}

			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return nil, errors.New("environment is not a struct")
			}

			for _, field := range st.Fields.List {
				if field.Tag == nil || len(field.Names) == 0 {
					continue
				}

				tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
				envName := tag.Get("env")
				if envName == "" {
					continue
				}
				// Exclude AWS env vars, as they are in a separate table
				if strings.HasPrefix(envName, "AWS_") {
					continue
				}

				comment := ""
				if field.Doc != nil {
					comment = strings.TrimSpace(field.Doc.Text())
					comment = strings.TrimRight(field.Doc.Text(), "\n")
				}
				minValue := tag.Get("minValue")
				if minValue != "" {
					comment += ". Value must be " + minValue + " or greater"
				}

				result = append(result, envVar{
					Name:       "GOKAPI_" + envName,
					Action:     comment,
					Persistent: tag.Get("persistent") == "true",
					Default:    tag.Get("envDefault"),
				})
			}
		}
	}

	if len(result) == 0 {
		return nil, errors.New("no environment variables found")
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	result = append(result,
		envVar{
			Name:    "DOCKER_NONROOT",
			Action:  "DEPRECATED.\n\nDocker only: Runs the binary in the container as a non-root user, if set to \"true\"",
			Default: "false",
		},
		envVar{
			Name:    "TMPDIR",
			Action:  "Sets the path which contains temporary files",
			Default: "Non-Docker: Default OS path\n\nDocker: [DATA_DIR]",
		},
	)

	return result, nil
}

func writeEnvDocumentationFile(vars []envVar) {
	table := renderRSTTable(vars)

	data, err := os.ReadFile(fileDocumentationAdvanced)
	if err != nil {
		fmt.Println("ERROR: Cannot read file:")
		fmt.Println(err)
		os.Exit(6)
	}

	content := string(data)

	re := regexp.MustCompile(
		`(?s)(Available environment variables\s*=+\s*\n)(.*?)(\n\.\. \[\*] Variables that are persistent must be submitted during the first start when Gokapi creates a new config file\. They can be omitted afterwards\. Non-persistent variables need to be set on every start\.)`,
	)

	matches := re.FindStringSubmatchIndex(content)
	if matches == nil {
		fmt.Println("ERROR: environment variable table not found")
		os.Exit(6)
	}

	newContent := content[:matches[3]] + table + content[matches[5]:]
	err = os.WriteFile(fileDocumentationAdvanced, []byte(newContent), 0644)
	if err != nil {
		fmt.Println("ERROR: Cannot write file:")
		fmt.Println(err)
		os.Exit(6)
	}
	fmt.Println("Updated environment variables documentation")
}

func renderRSTTable(vars []envVar) string {
	headers := []string{"Name", "Action", "Persistent [*]_", "Default"}

	// Split all cells into lines
	type row [][]string
	var rows []row

	for _, v := range vars {
		persistent := "No"
		if v.Persistent {
			persistent = "Yes"
		}

		rows = append(rows, row{
			rstParagraphize(v.Name),
			rstParagraphize(v.Action),
			rstParagraphize(persistent),
			rstParagraphize(v.Default),
		})
	}

	// Determine column widths
	colWidths := make([]int, len(headers))
	for i, h := range headers {
		colWidths[i] = len(h)
	}

	for _, r := range rows {
		for c, cellLines := range r {
			for _, line := range cellLines {
				if len(line) > colWidths[c] {
					colWidths[c] = len(line)
				}
			}
		}
	}

	var b strings.Builder

	sep := func(ch string) {
		b.WriteString("+")
		for _, w := range colWidths {
			b.WriteString(strings.Repeat(ch, w+2))
			b.WriteString("+")
		}
		b.WriteString("\n")
	}

	writeRow := func(lines [][]string) {
		maxLines := 0
		for _, l := range lines {
			if len(l) > maxLines {
				maxLines = len(l)
			}
		}

		for i := 0; i < maxLines; i++ {
			b.WriteString("|")
			for c, cell := range lines {
				line := ""
				if i < len(cell) {
					line = cell[i]
				}
				b.WriteString(" ")
				b.WriteString(line)
				b.WriteString(strings.Repeat(" ", colWidths[c]-len(line)+1))
				b.WriteString("|")
			}
			b.WriteString("\n")
		}
	}

	// Header
	sep("-")
	writeRow([][]string{
		{headers[0]},
		{headers[1]},
		{headers[2]},
		{headers[3]},
	})
	sep("=")

	// Rows
	for _, r := range rows {
		writeRow(r)
		sep("-")
	}

	return b.String()
}
func rstParagraphize(s string) []string {
	// Normalize Windows line endings
	s = strings.ReplaceAll(s, "\r\n", "\n")

	// Split original lines
	lines := strings.Split(s, "\n")

	var out []string
	for i, line := range lines {
		out = append(out, line)
		// Insert blank line between lines (but not after last)
		if i < len(lines)-1 {
			out = append(out, "")
		}
	}
	return out
}
