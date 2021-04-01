package helper

/**
Various functions, mostly for OS access
*/

import (
	"bufio"
	cryptorand "crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
)

// Returns true if a folder exists
func FolderExists(folder string) bool {
	_, err := os.Stat(folder)
	if err == nil {
		return true
	}
	return !os.IsNotExist(err)
}

// Returns true if a file exists
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// Converts bytes to a human readable format
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

// Returns a URL-safe, base64 encoded securely generated random string.
func GenerateRandomString(length int) string {
	b, err := generateRandomBytes(length)
	if err != nil {
		return generateUnsafeId(length)
	}
	return base64.URLEncoding.EncodeToString(b)
}

// Creates the data folder if it does not exist
func CreateDataDir(dataDir string) {
	if !FolderExists(dataDir) {
		err := os.Mkdir(dataDir, 0770)
		Check(err)
	}
}

// Creates the ServerSettings folder if it does not exist
func CreateConfigDir(configDir string) {
	if !FolderExists(configDir) {
		err := os.Mkdir(configDir, 0770)
		Check(err)
	}
}

// Reads a line from the terminal and returns it as a string
func ReadLine() string {
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	return strings.Replace(text, "\n", "", -1)
}

// Panics if error is not nil
func Check(e error) {
	if e != nil {
		panic(e)
	}
}
