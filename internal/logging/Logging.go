package logging

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/environment/deprecation"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
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
var useCloudflare = false

var parsedTrustedIPs []net.IP
var parsedTrustedCIDRs []*net.IPNet

// Init sets the path where to write the log file to
func Init(filePath string) {
	logPath = filePath + "/log.txt"
	env := environment.New()
	outputToStdout = env.LogToStdout
	useCloudflare = env.UseCloudFlare
	parseTrustedProxies(env.TrustedProxies, !env.DisableDockerTrustedProxy)
}

// parseTrustedProxies processes the raw strings into net.IP and net.IPNet objects
func parseTrustedProxies(proxies []string, useDockerSubnet bool) {
	parsedTrustedIPs = nil
	parsedTrustedCIDRs = nil

	if environment.IsDockerInstance() && useDockerSubnet {
		subnet, err := getDockerSubnet()
		if err == nil {
			parsedTrustedCIDRs = append(parsedTrustedCIDRs, subnet)
		}
	}

	for _, proxy := range proxies {
		proxy = strings.TrimSpace(proxy)
		if strings.Contains(proxy, "/") {
			// Handle CIDR (e.g., "10.0.0.0/24")
			_, ipNet, err := net.ParseCIDR(proxy)
			if err == nil {
				parsedTrustedCIDRs = append(parsedTrustedCIDRs, ipNet)
			}
		} else {
			// Handle Fixed IP (e.g., "127.0.0.1")
			ip := net.ParseIP(proxy)
			if ip != nil {
				parsedTrustedIPs = append(parsedTrustedIPs, ip)
			}
		}
	}
}

func getDockerSubnet() (*net.IPNet, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			// Docker typically uses these private ranges
			// Common: 172.16.0.0/12, 192.168.0.0/16, 10.0.0.0/8
			// Docker bridge default: 172.17.0.0/16
			if ipnet.IP.IsPrivate() && !ipnet.IP.IsLoopback() {
				// Skip if it's just the host IP (not a subnet)
				if ipnet.IP.To4() != nil {
					return ipnet, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("no Docker subnet found")
}

// GetAll returns all log entries as a single string and if the log file exists
func GetAll() (string, bool) {
	exists, err := helper.FileExists(logPath)
	helper.Check(err)
	if exists {
		content, err := os.ReadFile(logPath)
		helper.Check(err)
		return string(content), true
	}
	return fmt.Sprintf("[%s] No log file found!", categoryWarning), false
}

// GetSince returns all log entries since a given timestamp
func GetSince(timestamp int64) string {
	exists, err := helper.FileExists(logPath)
	helper.Check(err)
	if !exists {
		return fmt.Sprintf("[%s] No log file found!", categoryWarning)
	}
	var (
		lines  []string
		output strings.Builder
	)

	err = readLinesReverse(logPath, timestamp, func(line string) (error, bool) {
		ts, err := parseTimeLogEntry(line)
		if err != nil {
			return nil, false // skip malformed lines
		}

		if ts.Unix() < timestamp {
			return nil, true
		}

		lines = append(lines, line)
		return nil, false
	})

	helper.Check(err)

	for i := len(lines) - 1; i >= 0; i-- {
		output.WriteString(lines[i])
		output.WriteByte('\n')
	}

	return output.String()
}

func readLinesReverse(path string, maxTime int64, handleLine func(string) (error, bool)) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	const chunkSize = 4096
	var buffer []byte

	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.ModTime().Unix() < maxTime {
		return nil
	}

	offset := info.Size()

	for offset > 0 {
		readSize := int64(chunkSize)
		if offset < readSize {
			readSize = offset
		}
		offset -= readSize

		_, err = file.Seek(offset, 0)
		if err != nil {
			return err
		}

		chunk := make([]byte, readSize)
		_, err = file.Read(chunk)
		if err != nil {
			return err
		}

		buffer = append(chunk, buffer...)
		for {
			idx := len(buffer) - 1
			for idx >= 0 && buffer[idx] != '\n' {
				idx--
			}
			if idx < 0 {
				break
			}
			line := string(buffer[idx+1:])
			buffer = buffer[:idx]
			err, endOfLine := handleLine(line)
			if err != nil || endOfLine {
				return err
			}
		}
	}

	// Handle the first line (start of file)
	if len(buffer) > 0 {
		err, endOfLine := handleLine(string(buffer))
		if err != nil || endOfLine {
			return err
		}
	}
	return nil
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

// LogInvalidLogin adds a log entry to indicate that an invalid login was attempted. Non-blocking
func LogInvalidLogin(username, ip string) {
	createLogEntry(categoryAuth, fmt.Sprintf("Invalid login for user %s by IP %s", username, ip), false)
}

// LogDownload adds a log entry when a download was requested. Non-Blocking
func LogDownload(file models.File, r *http.Request, saveIp bool) {
	if saveIp {
		createLogEntry(categoryDownload, fmt.Sprintf("%s, IP %s, ID %s, Useragent %s", file.Name, GetIpAddress(r), file.Id, r.UserAgent()), false)
	} else {
		createLogEntry(categoryDownload, fmt.Sprintf("%s, ID %s, Useragent %s", file.Name, file.Id, r.UserAgent()), false)
	}
}

// LogUpload adds a log entry when an upload was created. Non-Blocking
func LogUpload(file models.File, user models.User, fr models.FileRequest) {
	if fr.Id != "" {
		createLogEntry(categoryUpload, fmt.Sprintf("%s, ID %s, uploaded to file request %s (%s), owned by %s (user #%d) ", file.Name, file.Id, fr.Id, fr.Name, user.Name, user.Id), false)
	} else {
		createLogEntry(categoryUpload, fmt.Sprintf("%s, ID %s, uploaded by %s (user #%d)", file.Name, file.Id, user.Name, user.Id), false)
	}
}

// LogEdit adds a log entry when an upload was edited. Non-Blocking
func LogEdit(file models.File, user models.User) {
	createLogEntry(categoryEdit, fmt.Sprintf("%s, ID %s, edited by %s (user #%d)", file.Name, file.Id, user.Name, user.Id), false)
}

// LogCreateFileRequest adds a log entry when a file request was added. Non-Blocking
func LogCreateFileRequest(fr models.FileRequest, user models.User) {
	createLogEntry(categoryEdit, fmt.Sprintf("File request %s (%s) created by %s (user #%d)", fr.Id, fr.Name, user.Name, user.Id), false)
}

// LogEditFileRequest adds a log entry when a file request was edited. Non-Blocking
func LogEditFileRequest(fr models.FileRequest, user models.User) {
	createLogEntry(categoryEdit, fmt.Sprintf("File request %s (%s) created by %s (user #%d)", fr.Id, fr.Name, user.Name, user.Id), false)
}

// LogDeleteFileRequest adds a log entry when a file request was deleted. Non-Blocking
func LogDeleteFileRequest(fr models.FileRequest, user models.User) {
	createLogEntry(categoryEdit, fmt.Sprintf("File request %s (%s) and associated files deleted by %s (user #%d)", fr.Id, fr.Name, user.Name, user.Id), false)
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

// LogRestore adds a log entry when the pending deletion of a file was cancelled and the file restored. Non-Blocking
func LogRestore(file models.File, user models.User) {
	createLogEntry(categoryEdit, fmt.Sprintf("%s, ID %s, restored by %s (user #%d)", file.Name, file.Id, user.Name, user.Id), false)
}

// LogDeprecation adds a log entry to indicate that a deprecated feature is being used. Blocking
func LogDeprecation(dep deprecation.Deprecation) {
	createLogEntry(categoryWarning, "Deprecated feature: "+dep.Name, true)
	createLogEntry(categoryWarning, dep.Description, true)
	createLogEntry(categoryWarning, "See "+dep.DocUrl+" for more information.", true)
}

// DeleteLogs removes all logs before the cutoff timestamp and inserts a new log that the user
// deleted the previous logs
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
		userName, userId, getDate(time.Now()), GetIpAddress(r)), timestamp)
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

func isTrustedProxy(ip net.IP) bool {
	// Check against fixed IPs
	for _, trustedIP := range parsedTrustedIPs {
		if trustedIP.Equal(ip) {
			return true
		}
	}

	// Check against CIDR ranges
	for _, trustedNet := range parsedTrustedCIDRs {
		if trustedNet.Contains(ip) {
			return true
		}
	}

	return false
}

// GetIpAddress returns the IP address of the requester
func GetIpAddress(r *http.Request) string {

	if useCloudflare {
		cfIp := r.Header.Get("CF-Connecting-IP")
		if cfIp != "" {
			return cfIp
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}

	// Clean up if it is an IPv6 zone
	netIP := net.ParseIP(ip)

	// Check if the immediate connector is a Trusted Proxy and if yes, use their header for IP
	// Otherwise this returns the actual IP used for the connection
	if netIP != nil && isTrustedProxy(netIP) {

		// Check X-Forwarded-For
		// Ideally, use the last IP in the list if a proxy appends to it
		xff := r.Header.Get("X-FORWARDED-FOR")
		if xff != "" {
			ips := strings.Split(xff, ",")
			// Iterate from right to left, skip trusted proxies
			for i := len(ips) - 1; i >= 0; i-- {
				ipXff := strings.TrimSpace(ips[i])
				parsedIP := net.ParseIP(ipXff)
				if parsedIP != nil && !isTrustedProxy(parsedIP) {
					return ipXff
				}
			}
		}

		// Fallback to X-Real-Ip if XFF fails
		xri := r.Header.Get("X-REAL-IP")
		if net.ParseIP(xri) != nil {
			return xri
		}
	}

	return ip
}
