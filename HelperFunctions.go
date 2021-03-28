package main

/**
Various functions, mostly for OS access
*/

import (
	"bufio"
	cryptorand "crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
)


// Hashes a password with SHA256 and a salt
func hashPassword(password, salt string) string {
	if password == "" {
		return ""
	}
	bytes := []byte(password + salt)
	hash := sha1.New()
	hash.Write(bytes)
	return hex.EncodeToString(hash.Sum(nil))
}

// Returns true if a folder exists
func folderExists(folder string) bool {
	_, err := os.Stat(folder)
	if err == nil {
		return true
	}
	return !os.IsNotExist(err)
}

// Returns true if a file exists
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// Converts bytes to a human readable format
func byteCountSI(b int64) string {
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

//Used if unable to generate secure random string. A warning will be output
//to the CLI window
func unsafeId(length int) string {
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
func generateRandomString(length int) (string, error) {
	b, err := generateRandomBytes(length)
	return base64.URLEncoding.EncodeToString(b), err
}

// Creates the data folder if it does not exist
func createDataDir() {
	if !folderExists(dataDir) {
		err := os.Mkdir(dataDir, 0770)
		check(err)
	}
}

// Creates the config folder if it does not exist
func createConfigDir() {
	if !folderExists(configDir) {
		err := os.Mkdir(configDir, 0770)
		check(err)
	}
}

// Reads a line from the terminal and returns it as a string
func readLine() string {
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	return strings.Replace(text, "\n", "", -1)
}

// Panics if error is not nil
func check(e error) {
	if e != nil {
		panic(e)
	}
}
