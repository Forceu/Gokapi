package helper

/**
Generates / annotates strings
*/

import (
	cryptorand "crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
)

// Returns securely generated random bytes.
func generateRandomBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = cryptorand.Read(b)
	return b
}

// GenerateRandomString returns a URL-safe, base64 encoded securely generated random string.
func GenerateRandomString(length int) string {
	b := generateRandomBytes(length + 10)
	result := cleanRandomString(base64.URLEncoding.EncodeToString(b))
	if len(result) < length {
		return GenerateRandomString(length)
	}
	return result[:length]
}

// ByteCountSI converts bytes to a human-readable format
func ByteCountSI(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

var regexRandomString = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// Removes special characters from string
func cleanRandomString(input string) string {
	return regexRandomString.ReplaceAllString(input, "")
}

var regexContent = regexp.MustCompile(`[^a-zA-Z0-9/ \-=\+\.]+`)

// SanitiseContentType removes invalid characters from the contentType string
// or returns default when too long or too short
func SanitiseContentType(contentType string) string {
	if len(contentType) > 100 || len(strings.TrimSpace(contentType)) < 2 {
		return "application/octet-stream"
	}
	return regexContent.ReplaceAllString(contentType, "")
}

// Remove characters that are dangerous in filenames on common OSes or in HTTP headers:
//
//	/ \ : * ? " < > |  — forbidden on Windows and/or meaningful on Unix
//	\r \n                — would break HTTP header injection
var regexFileName = regexp.MustCompile(`[/\\:*?"<>|\r\n]`)

// SanitiseFilename removes or replaces characters from a filename that could be
// used for path traversal, header injection, or shell injection attacks.
// It preserves the base name only (strips any directory components), then
// removes ASCII control characters and the following special characters:
// / \ : * ? " < > | null byte, and trims leading dots to prevent hidden files.
// String is limited to 400 characters
func SanitiseFilename(name string) string {
	// Remove null bytes and ASCII control characters (0x00–0x1F, 0x7F), limit string length
	var b strings.Builder
	if len(name) > 400 {
		name = name[0:400] + "..."
	}
	for _, r := range name {
		if r == 0x00 || (r >= 0x01 && r <= 0x1F) || r == 0x7F {
			continue
		}
		b.WriteRune(r)
	}
	name = b.String()

	name = regexFileName.ReplaceAllString(name, "_")

	// Trim leading dots to prevent hidden files (e.g. ".bashrc", "..foo")
	name = strings.TrimLeft(name, ".")

	return strings.TrimSpace(name)
}
