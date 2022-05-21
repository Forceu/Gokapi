package logging

import (
	"fmt"
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

// Init sets the path where to write the log file to
func Init(filePath string) {
	logPath = filePath + "/log.txt"
}

// AddString adds a line to the logfile including the current date. Non-Blocking
func AddString(text string) {
	go writeToFile(text)
}

// AddDownload adds a line to the logfile when a download was requested. Non-Blocking
func AddDownload(file *models.File, r *http.Request) {
	AddString(fmt.Sprintf("Download: Filename %s, IP %s, ID %s, Useragent %s", file.Name, getIpAddress(r), file.Id, r.UserAgent()))
}

func writeToFile(text string) {
	mutex.Lock()
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	helper.Check(err)
	defer file.Close()
	defer mutex.Unlock()
	_, err = file.WriteString(time.Now().UTC().Format(time.RFC1123) + "   " + text + "\n")
	helper.Check(err)
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
