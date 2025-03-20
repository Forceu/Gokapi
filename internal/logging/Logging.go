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
const categoryAuth = "auth"
const categoryWarning = "warning"

var outputToStdout = false

// Init sets the path where to write the log file to
func Init(filePath string) {
	logPath = filePath + "/log.txt"
	env := environment.New()
	outputToStdout = env.LogToStdout
}

// GetAll returns all log entries as a single string and if the log file exists
func GetAll() (string, bool) {
	if helper.FileExists(logPath) {
		content, err := os.ReadFile(logPath)
		helper.Check(err)
		return string(content), true
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
	return createLogFormatCustomTimestamp(category, text, time.Now())
}
func createLogFormatCustomTimestamp(category, text string, timestamp time.Time) string {
	return fmt.Sprintf("%s   [%s] %s", getDate(timestamp), category, text)
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

// LogUserDeletion adds a log entry to indicate that a user was deleted. Non-blocking
func LogUserDeletion(modifiedUser, userEditor models.User) {
	createLogEntry(categoryAuth, fmt.Sprintf("%s (#%d) was deleted by %s (user #%d)",
		modifiedUser.Name, modifiedUser.Id, userEditor.Name, userEditor.Id), false)
}

// LogUserEdit adds a log entry to indicate that a user was modified. Non-blocking
func LogUserEdit(modifiedUser, userEditor models.User) {
	createLogEntry(categoryAuth, fmt.Sprintf("%s (#%d) was modified by %s (user #%d)",
		modifiedUser.Name, modifiedUser.Id, userEditor.Name, userEditor.Id), false)
}

// LogUserCreation adds a log entry to indicate that a user was created. Non-blocking
func LogUserCreation(modifiedUser, userEditor models.User) {
	createLogEntry(categoryAuth, fmt.Sprintf("%s (#%d) was created by %s (user #%d)",
		modifiedUser.Name, modifiedUser.Id, userEditor.Name, userEditor.Id), false)
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

// LogEdit adds a log entry when an upload was edited. Non-Blocking
func LogEdit(file models.File, user models.User) {
	createLogEntry(categoryEdit, fmt.Sprintf("%s, ID %s, edited by %s (user #%d)", file.Name, file.Id, user.Name, user.Id), false)
}

// LogReplace adds a log entry when an upload was replaced. Non-Blocking
func LogReplace(originalFile, newContent models.File, user models.User) {
	createLogEntry(categoryEdit, fmt.Sprintf("%s, ID %s had content replaced with %s (ID %s) by %s (user #%d)",
		originalFile.Name, originalFile.Id, newContent.Name, newContent.Id, user.Name, user.Id), false)
}

// LogDelete adds a log entry when an upload was deleted. Non-Blocking
func LogDelete(file models.File, user models.User) {
	createLogEntry(categoryEdit, fmt.Sprintf("%s, ID %s, deleted by %s (user #%d)", file.Name, file.Id, user.Name, user.Id), false)
}

// UpgradeToV2 adds tags to existing logs
// deprecated
func UpgradeToV2() {
	content, exists := GetAll()
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

func DeleteLogs(userName string, userId int, cutoff int64, r *http.Request) {
	if cutoff == 0 {
		deleteAllLogs(userName, userId, r)
		return
	}
	mutex.Lock()
	logFile, err := os.ReadFile(logPath)
	helper.Check(err)
	var newFile strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(string(logFile)))
	newFile.WriteString(getLogDeletionMessage(userName, userId, r, time.Unix(cutoff, 0)))
	for scanner.Scan() {
		line := scanner.Text()
		timeEntry, err := parseTimeLogEntry(line)
		if err != nil {
			fmt.Println(err)
			continue
		}
		if timeEntry.Unix() > cutoff {
			newFile.WriteString(line + "\n")
		}
	}
	err = os.WriteFile(logPath, []byte(newFile.String()), 0600)
	helper.Check(err)
	defer mutex.Unlock()
}

func parseTimeLogEntry(input string) (time.Time, error) {
	const layout = "Mon, 02 Jan 2006 15:04:05 MST"
	lineContent := strings.Split(input, "   [")
	return time.Parse(layout, lineContent[0])
}

func getLogDeletionMessage(userName string, userId int, r *http.Request, timestamp time.Time) string {
	return createLogFormatCustomTimestamp(categoryWarning, fmt.Sprintf("Previous logs deleted by %s (user #%d) on %s. IP: %s\n",
		userName, userId, getDate(time.Now()), getIpAddress(r)), timestamp)
}

func deleteAllLogs(userName string, userId int, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()
	message := getLogDeletionMessage(userName, userId, r, time.Now())
	err := os.WriteFile(logPath, []byte(message), 0600)
	helper.Check(err)
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

func getDate(timestamp time.Time) string {
	return timestamp.UTC().Format(time.RFC1123)
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
