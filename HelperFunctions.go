package main

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

func check(e error) {
	if e != nil {
		panic(e)
	}
}

const SALT_PW_ADMIN = "eefwkjqweduiotbrkl##$2342brerlk2321"
const SALT_PW_FILES = "P1UI5sRNDwuBgOvOYhNsmucZ2pqo4KEvOoqqbpdu"

func hashPassword(password, salt string) string {
	if password == "" {
		return ""
	}
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
func generateRandomString(length int) (string, error) {
	b, err := generateRandomBytes(length)
	return base64.URLEncoding.EncodeToString(b), err
}

func createDataDir() {
	if !folderExists("data") {
		err := os.Mkdir("data", 0770)
		check(err)
	}
}

func createConfigDir() {
	if !folderExists(configDir) {
		err := os.Mkdir(configDir, 0770)
		check(err)
	}
}
func readLine() string {
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	return strings.Replace(text, "\n", "", -1)
}

func isDocker() bool {
	return fileExists(".isdocker")
}
