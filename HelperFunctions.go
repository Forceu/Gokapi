package main

import (
	cryptorand "crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"os"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func hashPassword(password string) string {
	const salt = "eefwkjqweduiotbrkl##$2342brerlk2321"
	bytes := []byte(password + salt)
	hash := sha1.New()
	hash.Write(bytes)
	return hex.EncodeToString(hash.Sum(nil))
}

func folderExists(folder string) bool {
	_, err := os.Stat(folder)
	if err == nil {
		return true
	}
	return !os.IsNotExist(err)
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

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

//Used if unable to generate secure random string. A warning will be output
//to the CLI window
func unsafeId(n int) string {
	log.Println("Warning! Cannot generate securely random ID!")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// generateRandomBytes returns securely generated random bytes.
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

// generateRandomString returns a URL-safe, base64 encoded
// securely generated random string.
func generateRandomString(s int) (string, error) {
	b, err := generateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}
