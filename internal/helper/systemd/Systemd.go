package systemd

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
)

// InstallService installs Gokapi as a systemd service
func InstallService() {
	checkRunAsRoot()
	checkSystemdOs()

	fmt.Println("Installing Gokapi as a service...")

	// Check if the service file already exists
	if _, err := os.Stat("/usr/lib/systemd/system/gokapi.service"); err == nil {
		fmt.Println("Service file already exists. Reinstalling it")
	}

	// Find the path to the current executable and it's directory
	executablePath, err := os.Executable()
	executableDir := filepath.Dir(executablePath)

	// Get the name of the current user
	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("Error getting current user: ", err)
		os.Exit(5)

	}

	// Create the service file
	serviceFileContents := createSystemdFileContent(executablePath, executableDir, currentUser.Username)

	err = os.WriteFile("/usr/lib/systemd/system/gokapi.service", serviceFileContents, 0644)
	if err != nil {
		fmt.Println("Error writing service data to file: ", err)
		os.Exit(3)
	}

	systemctlCmd("daemon-reload")
	systemctlCmd("enable", "gokapi.service")
	systemctlCmd("start", "gokapi.service")

	fmt.Println("Service installed and started successfully.")
	fmt.Println("The Gokapi executable found at " + executablePath + " will now run on startup in the background.")
	fmt.Println("Please do not remove the executable file from that location or the service will not start.")

	// Exit the program
	os.Exit(0)

}

// UninstallService uninstalls Gokapi as a systemd service
func UninstallService() {
	checkRunAsRoot()
	checkSystemdOs()

	fmt.Println("Uninstalling Gokapi as a service...")

	// Check if the service file exists
	if _, err := os.Stat("/usr/lib/systemd/system/gokapi.service"); os.IsNotExist(err) {
		fmt.Println("Service does not exist in systemd. Nothing to uninstall.")
		os.Exit(3)
	}
	systemctlCmd("stop", "gokapi.service")
	systemctlCmd("disable", "gokapi.service")
	// Remove the service file
	fmt.Println("Removing the service file...")
	err := os.Remove("/usr/lib/systemd/system/gokapi.service")
	if err != nil {
		fmt.Println("Error removing service file: ", err)
		os.Exit(4)
	}

	systemctlCmd("daemon-reload")
	fmt.Println("Service uninstalled successfully.")

	// Exit the program
	os.Exit(0)
}

// checkRunAsRoot displays an error message and exits the program if not run as root
func checkRunAsRoot() {
	if os.Geteuid() != 0 {
		fmt.Println("This feature requires root privileges.")
		os.Exit(1)
	}
}

// checkSystemdOs displays an error message and exits the program if the OS is not systemd based
func checkSystemdOs() {
	if _, err := os.Stat("/usr/lib/systemd/system"); os.IsNotExist(err) {
		fmt.Println("This feature is only supported on systems using systemd.")
		os.Exit(2)
	}
}

// systemctlCmd runs the command systemctl with the provided arguments. It displays an error message and exits the program
// if an error is encountered
func systemctlCmd(arg ...string) {
	err := exec.Command("systemctl", arg...).Run()
	if err != nil {
		fmt.Println("Error executing systemctl "+arg[0]+": ", err)
		os.Exit(4)
	}
}

// createSystemdFileContent returns a byte array with the content of the systemd file to be written
func createSystemdFileContent(executablePath, executableDir, username string) []byte {
	return []byte(`[Unit]
Description=Gokapi
After=network.target

[Service]
ExecStart=` + executablePath + `
WorkingDirectory=` + executableDir + `
User=` + username + `
Restart=always

[Install]
WantedBy=multi-user.target`)
}
