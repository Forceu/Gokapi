package configupgrade

import (
	"fmt"
	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/models"
	"os"
)

// CurrentConfigVersion is the version of the configuration structure. Used for upgrading
const CurrentConfigVersion = 22

const minConfigVersion = 22

// DoUpgrade checks if an old version is present and updates it to the current version if required
func DoUpgrade(settings *models.Configuration, env *environment.Environment) bool {
	if settings.ConfigVersion < CurrentConfigVersion {
		updateConfig(settings, env)
		fmt.Printf("Successfully upgraded configuration to version %d\n", CurrentConfigVersion)
		settings.ConfigVersion = CurrentConfigVersion
		return true
	}
	return false
}

// Upgrades the settings if saved with a previous version
func updateConfig(settings *models.Configuration, env *environment.Environment) {
	// < v2.0.0
	if settings.ConfigVersion < minConfigVersion {
		fmt.Println("Please update to version 2.0.0 before running this version.")
		osExit(1)
		return
	}
}

var osExit = os.Exit
