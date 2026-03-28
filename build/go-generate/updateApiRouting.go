package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strconv"
	"strings"
)

const fileRouting = "../../internal/webserver/api/routing.go"
const fileOutput = "../../internal/webserver/api/routingParsing.go"

// Function to find all declared types referenced in the RequestParser field
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
		genDecl, ok := decl.(*ast.GenDecl)
		if ok && genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if ok {
					if strings.HasPrefix(typeSpec.Name.String(), "param") {
						declaredTypes = append(declaredTypes, typeSpec)
					}
				}
			}
		}
	}

	return declaredTypes, nil
}

// hasParsableTags returns true if any field has a "header:" or "json:" struct tag.
func hasParsableTags(fields []*ast.Field) bool {
	return hasHeaderTags(fields) || hasJsonTags(fields) || hasHttpRequestTags(fields) || hasPostFormTags(fields)
}

// hasHeaderTags returns true if any field has a "header:" struct tag.
func hasHeaderTags(fields []*ast.Field) bool {
	for _, field := range fields {
		if field.Tag != nil {
			tag := field.Tag.Value[1 : len(field.Tag.Value)-1]
			for _, part := range strings.Split(tag, " ") {
				if strings.HasPrefix(part, "header:") {
					return true
				}
			}
		}
	}
	return false
}

// hasJsonTags returns true if any field has a "json:" struct tag.
func hasJsonTags(fields []*ast.Field) bool {
	for _, field := range fields {
		if field.Tag != nil {
			tag := field.Tag.Value[1 : len(field.Tag.Value)-1]
			for _, part := range strings.Split(tag, " ") {
				if strings.HasPrefix(part, "json:") {
					return true
				}
			}
		}
	}
	return false
}

// hasHttpRequestTags returns true if any field has a "isHttpRequest:" struct tag.
func hasHttpRequestTags(fields []*ast.Field) bool {
	for _, field := range fields {
		if field.Tag != nil {
			tag := field.Tag.Value[1 : len(field.Tag.Value)-1]
			for _, part := range strings.Split(tag, " ") {
				if strings.HasPrefix(part, "isHttpRequest:") {
					return true
				}
			}
		}
	}
	return false
}

// hasPostFormTags returns true if any field has a "postForm:" struct tag.
func hasPostFormTags(fields []*ast.Field) bool {
	for _, field := range fields {
		if field.Tag != nil {
			tag := field.Tag.Value[1 : len(field.Tag.Value)-1]
			for _, part := range strings.Split(tag, " ") {
				if strings.HasPrefix(part, "postForm:") {
					return true
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

func hasBase64Tag(tags []string) bool {
	// Check if the tag contains "supportBase64:true"
	for _, tag := range tags {
		if strings.HasPrefix(tag, "supportBase64") {
			return true
		}
	}
	return false
}

func headerExists(headerName string, required, isString, base64Support bool) string {
	base64SupportEntry := ""
	if base64Support {
		base64SupportEntry = ", has base64support"
	}
	return fmt.Sprintf("\n"+`
								// RequestParser header value %s, required: %v%s
								exists, err = checkHeaderExists(r, %s, %v, %v)
								if err != nil {
									return err
								}
								p.foundHeaders[%s] = exists`, headerName, required, base64SupportEntry, headerName, required, isString, headerName)
}

func generateParseRequestMethod(typeName string, fields []*ast.Field) string {
	// Start generating the ParseRequest method
	if !hasParsableTags(fields) {
		return fmt.Sprintf(`
				// ParseRequest parses the header file. As %s has no fields with the tags header,
				// json or isHttpRequest, this method does nothing except calling ProcessParameter()
				func (p *%s) ParseRequest(r *http.Request) error {
					return p.ProcessParameter(r)
				}
				%s`, typeName, typeName, writeNewInstanceCode(typeName))
	}

	needsHeader := hasHeaderTags(fields)
	needsJson := hasJsonTags(fields)
	needsHttpRequest := hasHttpRequestTags(fields)
	needsPostForm := hasPostFormTags(fields)

	// Build preamble: header parsing needs foundHeaders + exists; JSON-only still
	// needs err for the Decode call.
	preamble := ""
	if needsHeader {
		preamble = `var err error
			var exists bool
			p.foundHeaders = make(map[string]bool)`
	} else {
		if needsJson || needsPostForm {
			preamble = "var err error"
		}
	}

	var readValues []string
	if needsHttpRequest {
		readValues = append(readValues, "HTTP request")
	}
	if needsHeader {
		readValues = append(readValues, "header")
	}
	if needsJson {
		readValues = append(readValues, "JSON")
	}
	if needsPostForm {
		readValues = append(readValues, "POST form")
	}

	method := fmt.Sprintf(`// ParseRequest reads r and saves the passed %s values in the %s struct
		// In the end, ProcessParameter() is called
		func (p *%s) ParseRequest(r *http.Request) error {
			%s`, strings.Join(readValues, " and "), typeName, typeName, preamble)

	// Emit the JSON decode block before individual field assignments.
	// A single anonymous struct is decoded once; fields are then assigned individually
	// so that required-field checks and the struct's own field names are preserved.
	if needsJson {
		type jsonField struct {
			fieldName string
			jsonKey   string
			fieldType string
			required  bool
		}
		var jsonFields []jsonField
		for _, field := range fields {
			if field.Tag == nil {
				continue
			}
			tag := field.Tag.Value[1 : len(field.Tag.Value)-1]
			for _, part := range strings.Split(tag, " ") {
				if strings.HasPrefix(part, "json:") {
					jsonKey := strings.TrimPrefix(part, "json:")
					jsonKey = strings.Trim(jsonKey, "\"")
					jsonKey = strings.Split(jsonKey, ",")[0] // strip omitempty etc.
					fieldType := field.Type.(*ast.Ident).Name
					required := hasRequiredTag(strings.Split(tag, " "))
					jsonFields = append(jsonFields, jsonField{
						fieldName: field.Names[0].Name,
						jsonKey:   jsonKey,
						fieldType: fieldType,
						required:  required,
					})
				}
			}
		}

		// Build an anonymous intermediate struct matching the JSON shape
		intermediateFields := ""
		for _, jf := range jsonFields {
			intermediateFields += fmt.Sprintf("\n\t\t\t%s %s `json:\"%s\"`", jf.fieldName, jf.fieldType, jf.jsonKey)
		}
		method += fmt.Sprintf(`
			var jsonBody struct {%s
			}
			err = json.NewDecoder(r.Body).Decode(&jsonBody)
			if err != nil {
				return err
			}`, intermediateFields)

		// Emit required checks followed by assignment for each json field
		for _, jf := range jsonFields {
			if jf.required {
				switch jf.fieldType {
				case "string":
					method += fmt.Sprintf(`
			if jsonBody.%s == "" {
				return fmt.Errorf("json field \"%s\" is required")
			}`, jf.fieldName, jf.jsonKey)
				case "int", "int64":
					method += fmt.Sprintf(`
			if jsonBody.%s == 0 {
				return fmt.Errorf("json field \"%s\" is required")
			}`, jf.fieldName, jf.jsonKey)
				}
			}
			method += fmt.Sprintf(`
			p.%s = jsonBody.%s`, jf.fieldName, jf.fieldName)
		}
	}

	// Emit the POST form parsing block.
	// Each field is limited to defaultPostFormFieldMaxBytes unless the field
	// carries a maxPostBytes tag specifying a higher limit.
	// The overall body is also capped at the largest per-field limit before
	// ParseMultipartForm is called, so oversized requests are rejected early.
	if needsPostForm {
		const defaultPostFormFieldMaxBytes = 1024 // 1 KB

		type postFormField struct {
			fieldName    string
			formKey      string
			fieldType    string
			required     bool
			maxPostBytes int64 // per-field size limit in bytes
		}
		var postFormFields []postFormField
		for _, field := range fields {
			if field.Tag == nil {
				continue
			}
			tag := field.Tag.Value[1 : len(field.Tag.Value)-1]
			tagParts := strings.Split(tag, " ")
			for _, part := range tagParts {
				if strings.HasPrefix(part, "postForm:") {
					formKey := strings.Trim(strings.TrimPrefix(part, "postForm:"), "\"")
					fieldType := field.Type.(*ast.Ident).Name
					required := hasRequiredTag(tagParts)
					fieldMax := int64(defaultPostFormFieldMaxBytes)
					for _, p2 := range tagParts {
						if strings.HasPrefix(p2, "maxPostMb:") {
							raw := strings.Trim(strings.TrimPrefix(p2, "maxPostMb:"), "\"")
							n, err := strconv.Atoi(raw)
							if err == nil && n > 0 {
								fieldMax = int64(n) * 1024 * 1024
							}
						}
					}
					postFormFields = append(postFormFields, postFormField{
						fieldName:    field.Names[0].Name,
						formKey:      formKey,
						fieldType:    fieldType,
						required:     required,
						maxPostBytes: fieldMax,
					})
				}
			}
		}

		// Derive the total body limit from the largest individual field limit.
		// This is a conservative upper bound — the per-field checks below are
		// the authoritative enforcement.
		var totalLimit int64
		for _, pf := range postFormFields {
			if pf.maxPostBytes > totalLimit {
				totalLimit = pf.maxPostBytes
			}
		}
		method += fmt.Sprintf(`
			r.Body = http.MaxBytesReader(nil, r.Body, %d)
			err = r.ParseMultipartForm(int64(configuration.Get().MaxMemory) * 1024 * 1024)
			if err != nil {
				return err
			}`, totalLimit)

		for _, pf := range postFormFields {
			switch pf.fieldType {
			case "string":
				if pf.required {
					method += fmt.Sprintf(`
			if r.FormValue(%q) == "" {
				return fmt.Errorf("post form field \"%s\" is required")
			}`, pf.formKey, pf.formKey)
				}
				method += fmt.Sprintf(`
			if len(r.FormValue(%q)) > %d {
				return fmt.Errorf("post form field \"%s\" exceeds maximum length of %d bytes")
			}
			p.%s = r.FormValue(%q)`, pf.formKey, pf.maxPostBytes, pf.formKey, pf.maxPostBytes, pf.fieldName, pf.formKey)
			case "int", "int64":
				parseExpr := fmt.Sprintf(`strconv.Atoi(r.FormValue(%q))`, pf.formKey)
				assignExpr := fmt.Sprintf("p.%s", pf.fieldName)
				if pf.fieldType == "int64" {
					parseExpr = fmt.Sprintf(`strconv.ParseInt(r.FormValue(%q), 10, 64)`, pf.formKey)
				}
				if pf.required {
					method += fmt.Sprintf(`
			if r.FormValue(%q) == "" {
				return fmt.Errorf("post form field \"%s\" is required")
			}`, pf.formKey, pf.formKey)
				}
				method += fmt.Sprintf(`
			if r.FormValue(%q) != "" {
				%s, err = %s
				if err != nil {
					return fmt.Errorf("invalid value in post form field \"%s\"")
				}
			}`, pf.formKey, assignExpr, parseExpr, pf.formKey)
			case "bool":
				method += fmt.Sprintf(`
			if r.FormValue(%q) != "" {
				p.%s, err = strconv.ParseBool(r.FormValue(%q))
				if err != nil {
					return fmt.Errorf("invalid value in post form field \"%s\"")
				}
			}`, pf.formKey, pf.fieldName, pf.formKey, pf.formKey)
			default:
				panic("unsupported postForm field type: " + pf.fieldType)
			}
		}
	}

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
				base64Support := hasBase64Tag(tagParts)
				for _, part := range tagParts {
					if strings.HasPrefix(part, "isHttpRequest:") {
						method += fmt.Sprintf("\np.%s = r", field.Names[0].Name)
					}
					if strings.HasPrefix(part, "header:") {
						// Extract the header name after 'header:'
						headerName := strings.TrimPrefix(part, "header:")

						fieldType := field.Type.(*ast.Ident).Name

						// Use the appropriate parsing function based on the field type
						switch fieldType {
						case "string":
							method += headerExists(headerName, required, true, base64Support)
							if !base64Support {
								method += fmt.Sprintf(`
									if (exists) {
										p.%s = r.Header.Get(%s)
									}
									`, field.Names[0].Name, headerName)
							} else {
								method += fmt.Sprintf(`
									if (exists) {
										p.%s = r.Header.Get(%s)
										if strings.HasPrefix(p.%s, "base64:") {
											decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(p.%s, "base64:"))
											if err != nil {
												return err
											}
											p.%s = string(decoded)
										}
									}
								`, field.Names[0].Name, headerName, field.Names[0].Name, field.Names[0].Name, field.Names[0].Name)
							}

						case "bool":
							method += headerExists(headerName, required, false, false)
							method += fmt.Sprintf(`
							if (exists) {
								p.%s, err = parseHeaderBool(r, %s)
								if err != nil {
									 return	fmt.Errorf("invalid value in header %s supplied")
								}
							}
							`, field.Names[0].Name, headerName, strings.Replace(headerName, "\"", "", -1))

						case "int":
							method += headerExists(headerName, required, false, false)
							method += fmt.Sprintf(`
							if (exists) {
								p.%s, err = parseHeaderInt(r, %s)
									if err != nil {
										return fmt.Errorf("invalid value in header %s supplied")
								}
							}
							`, field.Names[0].Name, headerName, strings.Replace(headerName, "\"", "", -1))

						case "int64":
							method += headerExists(headerName, required, false, false)
							method += fmt.Sprintf(`
							if (exists) {
								p.%s, err = parseHeaderInt64(r, %s)
								if err != nil {
								    return fmt.Errorf("invalid value in header %s supplied")
								}
							}
							`, field.Names[0].Name, headerName, strings.Replace(headerName, "\"", "", -1))

						default:
							panic("unsupported field type")
						}
					}
				}
			}
		}
	}
	method += "\nreturn p.ProcessParameter(r)\n}\n"
	method += writeNewInstanceCode(typeName)

	return method
}

func writeNewInstanceCode(name string) string {
	return fmt.Sprintf(`
				// New returns a new instance of %s struct
				func (p *%s) New() requestParser {
					return &%s{}
				}`, name, name, name)
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

	// Conditionally import "encoding/json" only when at least one struct uses
	// json tags, to avoid an unused-import compile error in the generated file.
	needsJsonImport := false
	needsStrconvImport := false
	needsConfigImport := false
	for _, typeSpec := range types {
		if structType, ok := typeSpec.Type.(*ast.StructType); ok {
			if hasJsonTags(structType.Fields.List) {
				needsJsonImport = true
			}
			if hasPostFormTags(structType.Fields.List) {
				needsStrconvImport = true
				needsConfigImport = true
			}
		}
	}

	jsonImport := ""
	if needsJsonImport {
		jsonImport = "\n\t\"encoding/json\""
	}
	strconvImport := ""
	if needsStrconvImport {
		strconvImport = "\n\t\"strconv\""
	}
	if needsConfigImport {
		strconvImport += "\n\t\"github.com/forceu/gokapi/internal/configuration\""
	}

	output.WriteString(fmt.Sprintf(`// Code generated by updateApiRouting.go - DO NOT EDIT.
			package api
			
			import (%s%s
				"encoding/base64"
				"fmt"
				"net/http"
				"strings"
			)
			
			// Do not modify: This is an automatically generated file created by updateApiRouting.go
			// It contains the code that is used to parse the headers submitted in an API request

			`, jsonImport, strconvImport))

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
