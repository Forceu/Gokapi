package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
)

const fileRouting = "../../internal/webserver/api/routings.go"
const fileOutput = "../../internal/webserver/api/routingParsing.go"

// Function to find all declared types referenced in the Parsing field
func findDeclaredTypes(filePath string) ([]*ast.TypeSpec, error) {
	// Open the source file containing the struct
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Parse the Go source code
	fs := token.NewFileSet()
	node, err := parser.ParseFile(fs, filePath, file, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// Map to store the found types
	var declaredTypes []*ast.TypeSpec

	// Traverse the AST to find the struct definitions (type declarations)
	for _, decl := range node.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if strings.HasPrefix(typeSpec.Name.String(), "param") {
						declaredTypes = append(declaredTypes, typeSpec)
					}
				}
			}
		}
	}

	return declaredTypes, nil
}

func hasTags(fields []*ast.Field) bool {
	for _, field := range fields {
		if field.Tag != nil {
			// Extract the header tag by accessing the field.Tag.Value
			tag := field.Tag.Value
			if tag != "" {
				// Remove backticks
				tag = tag[1 : len(tag)-1]

				// Check if the tag has the "header" key and extract its value
				tagParts := strings.Split(tag, " ")
				for _, part := range tagParts {
					if strings.HasPrefix(part, "header:") {
						return true
					}
				}
			}
		}
	}
	return false
}

func hasRequiredTag(tags []string) bool {
	// Check if the tag contains "required:true"
	for _, tag := range tags {
		if strings.HasPrefix(tag, "required") {
			return true
		}
	}
	return false
}

func generateParseRequestMethod(typeName string, fields []*ast.Field) string {
	// Start generating the ParseRequest method
	if !hasTags(fields) {
		return fmt.Sprintf(`
				// ParseRequest parses the header file. As %s has no fields with the
				// tag header, this method does nothing, except calling ProcessParameter()
				func (p *%s) ParseRequest(r *http.Request) error {
					return p.ProcessParameter(r)
				}`, typeName, typeName)
	}

	method := fmt.Sprintf(`// ParseRequest reads r and saves the passed header values in the %s struct
// In the end, ProcessParameter() is called
func (p *%s) ParseRequest(r *http.Request) error {
var err error
p.foundHeaders = make(map[string]bool)`, typeName, typeName)

	// Iterate over the fields and generate parsing logic for those with a header tag
	for _, field := range fields {
		if field.Tag != nil {
			// Extract the header tag by accessing the field.Tag.Value
			tag := field.Tag.Value
			if tag != "" {
				// Remove backticks
				tag = tag[1 : len(tag)-1]

				// Check if the tag has the "header" key and extract its value
				tagParts := strings.Split(tag, " ")
				required := hasRequiredTag(tagParts)
				for _, part := range tagParts {
					if strings.HasPrefix(part, "header:") {
						// Extract header name after 'header:'
						headerName := strings.TrimPrefix(part, "header:")

						// Check for the "required" tag

						fieldType := field.Type.(*ast.Ident).Name

						// Use appropriate parsing function based on the field type
						switch fieldType {
						case "string":
							method += fmt.Sprintf(`
							p.%s, err = parseHeaderString(r, %s, %v, p.foundHeaders)
							if err != nil {
								return err
							}
					`, field.Names[0].Name, headerName, required)

						case "bool":
							method += fmt.Sprintf(`
							p.%s, err = parseHeaderBool(r, %s, %v, p.foundHeaders)
							if err != nil {
								return err
							}
							`, field.Names[0].Name, headerName, required)

						case "int":
							method += fmt.Sprintf(`
								p.%s, err = parseHeaderInt(r, %s, %v, p.foundHeaders)
								if err != nil {
									return err
								}
						`, field.Names[0].Name, headerName, required)

						case "int64":
							method += fmt.Sprintf(`
								p.%s, err = parseHeaderInt64(r, %s, %v, p.foundHeaders)
								if err != nil {
									return err
								}
							`, field.Names[0].Name, headerName, required)

						default:
							panic("unsupported field type")
						}
					}
				}
			}
		}
	}

	// Close the ParseRequest method
	method += "\nreturn p.ProcessParameter(r)\n}"

	return method
}

func writeAndFormatCode(generatedCode string, filePath string) error {
	// Write the generated code to the specified file
	err := os.WriteFile(filePath, []byte(generatedCode), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	// Read the file to format
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	// Format the content using go fmt
	formattedContent, err := format.Source(fileContent)
	if err != nil {
		return fmt.Errorf("failed to format file: %v", err)
	}

	// Write the formatted content back to the file
	err = os.WriteFile(filePath, formattedContent, 0644)
	if err != nil {
		return fmt.Errorf("failed to write formatted file: %v", err)
	}

	return nil
}

func main() {

	// Find declared types in the routings.go file
	types, err := findDeclaredTypes(fileRouting)
	if err != nil {
		log.Fatalf("Error finding types: %v", err)
	}

	var output strings.Builder

	output.WriteString(`// Code generated by updateApiRouting.go - DO NOT EDIT.
			package api
			
			import "net/http"
			
			// Do not modify: This is an automatically generated file created by updateApiRouting.go
			// It contains the code that is used to parse the headers submitted in an API request

			`)

	// Process each struct type
	for _, typeSpec := range types {
		// Find the struct definition and its fields
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}

		// Generate the ParseRequest method for the struct
		method := generateParseRequestMethod(typeSpec.Name.Name, structType.Fields.List)

		output.WriteString(method + "\n\n")
	}

	err = writeAndFormatCode(output.String(), fileOutput)
	if err != nil {
		log.Fatalf("Error writing file: %v", err)
	}

	fmt.Println("Updated API parsing")
}
