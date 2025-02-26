package logging

import (
	"bufio"
	"fmt"
	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var logPath = "config/log.txt"
var mutex sync.Mutex

const categoryInfo = "info"
const categoryDownload = "download"
const categoryUpload = "upload"
const categoryEdit = "edit"
const categoryAuth = "authentication"
const categoryWarning = "warning"

var outputToStdout = false

// Init sets the path where to write the log file to
func Init(filePath string) {
	logPath = filePath + "/log.txt"
	env := environment.New()
	outputToStdout = env.LogToStdout
}

// GetAll returns all log entries as a single string and if the log file exists
func GetAll(reverse bool) (string, bool) {
	if helper.FileExists(logPath) {
		content, err := os.ReadFile(logPath)
		helper.Check(err)
		result := string(content)
		if reverse {
			result = reverseLogFile(result)
		}
		return result, true
	} else {
		return fmt.Sprintf("[%s] No log file found!", categoryWarning), false
	}
}

// createLogEntry adds a line to the logfile including the current date. Also outputs to Stdout if set.
func createLogEntry(category, text string, blocking bool) {
	output := createLogFormat(category, text)
	if outputToStdout {
		fmt.Println(output)
	}
	if blocking {
		writeToFile(output)
	} else {
		go writeToFile(output)
	}
}

func createLogFormat(category, text string) string {
	return fmt.Sprintf("%s   [%s] %s", getDate(), category, text)
}

// LogStartup adds a log entry to indicate that Gokapi has started. Non-blocking
func LogStartup() {
	createLogEntry(categoryInfo, "Gokapi started", false)
}

// LogShutdown adds a log entry to indicate that Gokapi is shutting down. Blocking call
func LogShutdown() {
	createLogEntry(categoryInfo, "Gokapi shutting down", true)
}

// LogSetup adds a log entry to indicate that the setup was run. Non-blocking
func LogSetup() {
	createLogEntry(categoryAuth, "Re-running Gokapi setup", false)
}

// LogDeploymentPassword adds a log entry to indicate that a deployment password was set. Non-blocking
func LogDeploymentPassword() {
	createLogEntry(categoryAuth, "Setting new admin password", false)
}

// LogDownload adds a log entry when a download was requested. Non-Blocking
func LogDownload(file models.File, r *http.Request, saveIp bool) {
	if saveIp {
		createLogEntry(categoryDownload, fmt.Sprintf("%s, IP %s, ID %s, Useragent %s", file.Name, getIpAddress(r), file.Id, r.UserAgent()), false)
	} else {
		createLogEntry(categoryDownload, fmt.Sprintf("%s, ID %s, Useragent %s", file.Name, file.Id, r.UserAgent()), false)
	}
}

// LogUpload adds a log entry when an upload was created. Non-Blocking
func LogUpload(file models.File, user models.User) {
	createLogEntry(categoryUpload, fmt.Sprintf("%s, ID %s, uploaded by %s (user #%d)", file.Name, file.Id, user.Name, user.Id), false)
}

type logEntry struct {
	Previous *logEntry
	Next     *logEntry
	Content  string
}

func reverseLogFile(input string) string {
	var reversedLogs strings.Builder
	current := &logEntry{}
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		newEntry := logEntry{
			Content:  line,
			Previous: current,
		}
		current.Next = &newEntry
		current = &newEntry
	}
	for current.Previous != nil {
		reversedLogs.WriteString(current.Content + "\n")
		current = current.Previous
	}
	return reversedLogs.String()
}

// UpgradeToV2 adds tags to existing logs
// deprecated
func UpgradeToV2() {
	content, exists := GetAll(false)
	mutex.Lock()
	if !exists {
		return
	}
	var newLogs strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Gokapi started") {
			line = strings.Replace(line, "Gokapi started", "["+categoryInfo+"] Gokapi started", 1)
		}
		if strings.Contains(line, "Download: Filename") {
			line = strings.Replace(line, "Download: Filename", "["+categoryDownload+"] Filename", 1)
		}
		newLogs.WriteString(line)
		newLogs.WriteString("\n")
	}
	helper.Check(scanner.Err())
	err := os.WriteFile(logPath, []byte(newLogs.String()), 0600)
	helper.Check(err)
	defer mutex.Unlock()
}

func DeleteLogs(userName string, userId int, r *http.Request) {
	mutex.Lock()
	message := createLogFormat(categoryWarning, fmt.Sprintf("Previous logs deleted by %s (user #%d). IP: %s\n",
		userName, userId, getIpAddress(r)))
	err := os.WriteFile(logPath, []byte(message), 0600)
	helper.Check(err)
	defer mutex.Unlock()
}

func writeToFile(text string) {
	mutex.Lock()
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	helper.Check(err)
	defer file.Close()
	defer mutex.Unlock()
	_, err = file.WriteString(text + "\n")
	helper.Check(err)
}

func getDate() string {
	return time.Now().UTC().Format(time.RFC1123)
}

func getIpAddress(r *http.Request) string {
	// Get IP from X-FORWARDED-FOR header
	ips := r.Header.Get("X-FORWARDED-FOR")
	splitIps := strings.Split(ips, ",")
	for _, ip := range splitIps {
		netIP := net.ParseIP(ip)
		if netIP != nil {
			return ip
		}
	}

	// Get IP from the X-REAL-IP header
	ip := r.Header.Get("X-REAL-IP")
	netIP := net.ParseIP(ip)
	if netIP != nil {
		return ip
	}

	// Get IP from RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "Unknown IP"
	}
	netIP = net.ParseIP(ip)
	if netIP != nil {
		return ip
	}
	return "Unknown IP"
}
