package api

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

// OpenAPI specification structures
type OpenAPISpec struct {
	Paths map[string]PathItem `json:"paths"`
}

type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
}

type Operation struct {
	Parameters []Parameter `json:"parameters,omitempty"`
}

type Parameter struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Required bool   `json:"required"`
	Schema   Schema `json:"schema,omitempty"`
}

type Schema struct {
	Type string `json:"type"`
}

// Test function to validate OpenAPI spec against routes
func TestOpenAPISpecification(t *testing.T) {
	// Load OpenAPI specification
	openAPIPath := "../../../openapi.json"
	spec, err := loadOpenAPISpec(openAPIPath)
	if err != nil {
		t.Fatalf("Failed to load OpenAPI specification from %s: %v", openAPIPath, err)
	}

	// Track validation results
	var failures []string

	// 1. Check that all routes are defined in OpenAPI
	failures = append(failures, validateAllRoutesExist(spec)...)

	// 2. Check that all required headers are defined in OpenAPI
	failures = append(failures, validateRequiredHeaders(spec)...)

	// 3. Check for extra paths in OpenAPI that don't exist in routes
	failures = append(failures, validateNoExtraPaths(spec)...)

	// Report results
	if len(failures) > 0 {
		t.Errorf("OpenAPI validation failed with %d error(s):\n%s",
			len(failures), strings.Join(failures, "\n"))
	}
}

// loadOpenAPISpec loads and parses the OpenAPI JSON file
func loadOpenAPISpec(path string) (*OpenAPISpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var spec OpenAPISpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	return &spec, nil
}

// validateAllRoutesExist checks that all Go routes are defined in OpenAPI
func validateAllRoutesExist(spec *OpenAPISpec) []string {
	var failures []string

	for _, route := range routes {
		// Skip internal/undocumented routes (e2e endpoints)
		if strings.HasPrefix(route.Url, "/e2e/") {
			continue
		}

		// For wildcard routes, check base path
		checkPath := route.Url
		if route.HasWildcard {
			// OpenAPI uses {parameter} notation, so we need to check variations
			checkPath = strings.TrimSuffix(route.Url, "/")
			found := false

			// Check for exact match first
			if _, exists := spec.Paths[route.Url]; exists {
				found = true
			}

			// Check with OpenAPI parameter notation
			if !found {
				// Common patterns like /files/list/{id}
				paramPath := checkPath + "/{id}"
				if _, exists := spec.Paths[paramPath]; exists {
					found = true
				}
			}

			// Check without trailing slash
			if !found {
				if _, exists := spec.Paths[checkPath]; exists {
					found = true
				}
			}

			if !found {
				failures = append(failures, fmt.Sprintf(
					"Route %s (wildcard) not found in OpenAPI spec (expected variations: %s, %s/{id})",
					route.Url, checkPath, checkPath))
			}
		} else {
			// Non-wildcard routes should match exactly
			if _, exists := spec.Paths[route.Url]; !exists {
				failures = append(failures, fmt.Sprintf(
					"Route %s not found in OpenAPI spec", route.Url))
			}
		}
	}

	return failures
}

// validateRequiredHeaders checks that all required headers from RequestParser are in OpenAPI
// and also validates that there are no extra headers in OpenAPI
func validateRequiredHeaders(spec *OpenAPISpec) []string {
	var failures []string

	for _, route := range routes {
		// Skip routes without request parsers
		if route.RequestParser == nil {
			continue
		}

		// Skip internal/undocumented routes
		if strings.HasPrefix(route.Url, "/e2e/") {
			continue
		}

		// Get expected headers from the RequestParser
		expectedHeaders := extractHeadersFromParser(route.RequestParser)

		// Find the corresponding path in OpenAPI
		openAPIPath := findOpenAPIPath(spec, route)
		if openAPIPath == "" {
			// Already reported in validateAllRoutesExist
			continue
		}

		// Get parameters from OpenAPI
		openAPIHeaders := extractOpenAPIHeaders(spec.Paths[openAPIPath])

		// Validate each expected header from Go code
		for headerName, headerInfo := range expectedHeaders {
			// Skip unpublished headers - they should NOT be in OpenAPI
			if headerInfo.Unpublished {
				// But check if they accidentally ARE in OpenAPI
				if _, exists := openAPIHeaders[strings.ToLower(headerName)]; exists {
					failures = append(failures, fmt.Sprintf(
						"Route %s: header '%s' is marked as unpublished in Go code but exists in OpenAPI spec",
						route.Url, headerName))
				}
				continue
			}

			openAPIHeader, exists := openAPIHeaders[strings.ToLower(headerName)]
			if !exists {
				failures = append(failures, fmt.Sprintf(
					"Route %s: missing header '%s' in OpenAPI spec",
					route.Url, headerName))
				continue
			}

			// Check if required status matches EXACTLY (both directions)
			if headerInfo.Required != openAPIHeader.Required {
				if headerInfo.Required {
					failures = append(failures, fmt.Sprintf(
						"Route %s: header '%s' should be required in OpenAPI spec (currently optional)",
						route.Url, headerName))
				} else {
					failures = append(failures, fmt.Sprintf(
						"Route %s: header '%s' should be optional in OpenAPI spec (currently required)",
						route.Url, headerName))
				}
			}
		}

		// Check for extra headers in OpenAPI that don't exist in Go code
		for openAPIHeaderName, openAPIHeader := range openAPIHeaders {
			found := false
			for goHeaderName := range expectedHeaders {
				if strings.EqualFold(openAPIHeaderName, goHeaderName) {
					found = true
					break
				}
			}
			if !found {
				failures = append(failures, fmt.Sprintf(
					"Route %s: OpenAPI spec contains extra header '%s' that doesn't exist in Go code",
					route.Url, openAPIHeader.Name))
			}
		}
	}

	return failures
}

// validateNoExtraPaths checks for paths in OpenAPI that don't exist in routes
func validateNoExtraPaths(spec *OpenAPISpec) []string {
	var failures []string

	// Build a map of valid route paths
	validPaths := make(map[string]bool)
	for _, route := range routes {
		// Skip e2e routes as they're not documented
		if strings.HasPrefix(route.Url, "/e2e/") {
			continue
		}

		if route.HasWildcard {
			// For wildcard routes, we need to check multiple variations
			base := strings.TrimSuffix(route.Url, "/")
			validPaths[route.Url] = true
			validPaths[base] = true
			validPaths[base+"/{id}"] = true
			validPaths[base+"/{uuid}"] = true
		} else {
			validPaths[route.Url] = true
		}
	}

	// Check each OpenAPI path
	for path := range spec.Paths {
		// Skip if it's a valid path or a parameter variation
		if validPaths[path] {
			continue
		}

		// Check if it's a parameterized version of a wildcard route
		isValid := false
		for _, route := range routes {
			// Skip e2e routes
			if strings.HasPrefix(route.Url, "/e2e/") {
				continue
			}

			if route.HasWildcard {
				base := strings.TrimSuffix(route.Url, "/")
				if strings.HasPrefix(path, base+"/") &&
					(strings.Contains(path, "{") || path == base) {
					isValid = true
					break
				}
			}
		}

		if !isValid {
			failures = append(failures, fmt.Sprintf(
				"OpenAPI spec contains extra path '%s' that doesn't exist in Go routes", path))
		}
	}

	return failures
}

// HeaderInfo contains information about a header field
type HeaderInfo struct {
	Name          string
	Required      bool
	Unpublished   bool
	SupportBase64 bool
}

// extractHeadersFromParser uses reflection to extract header information
func extractHeadersFromParser(parser requestParser) map[string]HeaderInfo {
	headers := make(map[string]HeaderInfo)

	// Create a new instance to examine
	parserValue := reflect.ValueOf(parser)
	if parserValue.Kind() == reflect.Ptr {
		parserValue = parserValue.Elem()
	}
	parserType := parserValue.Type()

	// Iterate through fields
	for i := 0; i < parserType.NumField(); i++ {
		field := parserType.Field(i)

		// Look for 'header' tag
		headerTag := field.Tag.Get("header")
		if headerTag == "" {
			continue
		}

		// Check if required
		required := field.Tag.Get("required") == "true"

		// Check if not published
		unpublished := field.Tag.Get("unpublished") == "true"

		// Check if supports base64
		supportBase64 := field.Tag.Get("supportBase64") == "true"

		headers[headerTag] = HeaderInfo{
			Name:          headerTag,
			Required:      required,
			SupportBase64: supportBase64,
			Unpublished:   unpublished,
		}
	}

	return headers
}

// findOpenAPIPath finds the corresponding OpenAPI path for a route
func findOpenAPIPath(spec *OpenAPISpec, route apiRoute) string {
	// Try exact match first
	if _, exists := spec.Paths[route.Url]; exists {
		return route.Url
	}

	// For wildcard routes, try variations
	if route.HasWildcard {
		base := strings.TrimSuffix(route.Url, "/")

		// Try with {id}
		if _, exists := spec.Paths[base+"/{id}"]; exists {
			return base + "/{id}"
		}

		// Try without trailing slash
		if _, exists := spec.Paths[base]; exists {
			return base
		}

		// Try with {uuid}
		if _, exists := spec.Paths[base+"/{uuid}"]; exists {
			return base + "/{uuid}"
		}
	}

	return ""
}

// extractOpenAPIHeaders extracts all header parameters from an OpenAPI PathItem
func extractOpenAPIHeaders(pathItem PathItem) map[string]Parameter {
	headers := make(map[string]Parameter)

	// Check all HTTP methods
	operations := []*Operation{
		pathItem.Get,
		pathItem.Post,
		pathItem.Put,
		pathItem.Delete,
		pathItem.Patch,
	}

	for _, op := range operations {
		if op == nil {
			continue
		}

		for _, param := range op.Parameters {
			if param.In == "header" {
				headers[strings.ToLower(param.Name)] = param
			}
		}
	}

	return headers
}

// Helper test to print all routes and their headers (for debugging)
func TestPrintRoutesAndHeaders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping debug output in short mode")
	}

	t.Log("=== All Routes and Their Expected Headers ===")
	for _, route := range routes {
		t.Logf("\nRoute: %s (Wildcard: %v)", route.Url, route.HasWildcard)

		if route.RequestParser != nil {
			headers := extractHeadersFromParser(route.RequestParser)
			if len(headers) > 0 {
				t.Log("  Expected headers:")
				for name, info := range headers {
					requiredStr := ""
					if info.Required {
						requiredStr = " (required)"
					}
					unpublishedStr := ""
					if info.Unpublished {
						unpublishedStr = " (unpublished - should NOT be in OpenAPI)"
					}
					base64Str := ""
					if info.SupportBase64 {
						base64Str = " (supports base64)"
					}
					t.Logf("    - %s%s%s%s", name, requiredStr, unpublishedStr, base64Str)
				}
			}
		} else {
			t.Log("  No request parser")
		}
	}
}
