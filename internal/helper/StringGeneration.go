package helper

/**
Generates / annotates strings
*/

import (
	cryptorand "crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"regexp"
)

// A rune array to be used for pseudo-random string generation
var characters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// Used if unable to generate secure random string. A warning will be output
// to the CLI window
func generateUnsafeId(length int) string {
	log.Println("Warning! Cannot generate securely random ID!")
	b := make([]rune, length)
	for i := range b {
		b[i] = characters[rand.Intn(len(characters))]
	}
	return string(b)
}

// Returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly
func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := cryptorand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// GenerateRandomString returns a URL-safe, base64 encoded securely generated random string.
func GenerateRandomString(length int) string {
	b, err := generateRandomBytes(length + 10)
	if err != nil {
		return generateUnsafeId(length)
	}
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
