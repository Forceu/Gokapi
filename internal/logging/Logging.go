package logging

import (
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
const categoryAuth = "authentication"
const categoryWarning = "warning"

var outputToStdout = false

// Init sets the path where to write the log file to
func Init(filePath string) {
	logPath = filePath + "/log.txt"
	env := environment.New()
	outputToStdout = env.LogToStdout
}

// createLogEntry adds a line to the logfile including the current date. Also outputs to Stdout if set.
func createLogEntry(category, text string, blocking bool) {
	output := fmt.Sprintf("%s   [%s] %s", getDate(), category, text)
	if outputToStdout {
		fmt.Println(output)
	}
	if blocking {
		writeToFile(output)
	} else {
		go writeToFile(output)
	}
}

// GetLogPath returns the relative path to the log file
func GetLogPath() string {
	return logPath
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
func LogDownload(file *models.File, r *http.Request, saveIp bool) {
	if saveIp {
		createLogEntry(categoryDownload, fmt.Sprintf("Download: Filename %s, IP %s, ID %s, Useragent %s", file.Name, getIpAddress(r), file.Id, r.UserAgent()), false)
	} else {
		createLogEntry(categoryDownload, fmt.Sprintf("Download: Filename %s, ID %s, Useragent %s", file.Name, file.Id, r.UserAgent()), false)
	}
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
