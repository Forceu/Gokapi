//go:build !linux

package systemd

import (
	"fmt"
	"os"
)

// InstallService installs Gokapi as a systemd service
func InstallService() {
	invalidOS()
}

// UninstallService uninstalls Gokapi as a systemd service
func UninstallService() {
	invalidOS()
}

// invalidOS displays an error message and exits the program, as systemd is not supported on Windows
func invalidOS() {
	fmt.Println("This feature is only supported on systems using systemd.")
	os.Exit(2)
}
