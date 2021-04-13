package helper

/**
Simplified OS functions
*/

import (
	"bufio"
	"os"
	"strings"
)

// FolderExists returns true if a folder exists
func FolderExists(folder string) bool {
	_, err := os.Stat(folder)
	if err == nil {
		return true
	}
	return !os.IsNotExist(err)
}

// FileExists returns true if a file exists
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// CreateDataDir creates the data folder if it does not exist
func CreateDataDir(dataDir string) {
	if !FolderExists(dataDir) {
		err := os.Mkdir(dataDir, 0770)
		Check(err)
	}
}

// CreateConfigDir creates the ServerSettings folder if it does not exist
func CreateConfigDir(configDir string) {
	if !FolderExists(configDir) {
		err := os.Mkdir(configDir, 0770)
		Check(err)
	}
}

// ReadLine reads a line from the terminal and returns it as a string
func ReadLine() string {
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	return strings.Replace(text, "\n", "", -1)
}

// Check panics if error is not nil
func Check(e error) {
	if e != nil {
		panic(e)
	}
}

// IsInArray returns true if value is in array
func IsInArray(haystack []string, needle string) bool {
	for _, item := range haystack {
		if needle == item {
			return true
		}
	}
	return false
}
