package helper

/**
Simplified OS functions
*/

import (
	"bufio"
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
