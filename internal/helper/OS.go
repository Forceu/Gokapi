package helper

/**
Simplified OS functions
*/

import (
	"bufio"
	"errors"
	"golang.org/x/term"
	"os"
	"syscall"
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

// CreateDir creates the data folder if it does not exist
func CreateDir(name string) {
	if !FolderExists(name) {
		err := os.Mkdir(name, 0770)
		Check(err)
	}
}

// ReadLine reads a line from the terminal and returns it as a string
func ReadLine() string {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	text := scanner.Text()
	return text
}

// ReadPassword reads a line without displaying input from the terminal and returns it as a string
func ReadPassword() string {
	// int conversion is required for Windows systems
	pw, err := term.ReadPassword(int(syscall.Stdin))
	if err == nil {
		return string(pw)
	}
	return ReadLine()
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

// GetFileSize returns the file size in bytes
func GetFileSize(file *os.File) (int64, error) {
	fileInfo, err := file.Stat()
	if err != nil {
		return 0, err
	}
	return fileInfo.Size(), nil
}

// ErrPathDoesNotExist is raised if the requested path does not exist
var ErrPathDoesNotExist = errors.New("path does not exist")

// ErrPathIsNotDir is raised if the requested path is not a directory
var ErrPathIsNotDir = errors.New("path is not a directoryt")
