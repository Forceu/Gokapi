package helper

/**
Generates / annotates strings
*/

import (
	cryptorand "crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
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

// Removes special characters from string
func cleanRandomString(input string) string {
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	Check(err)
	return reg.ReplaceAllString(input, "")
}
