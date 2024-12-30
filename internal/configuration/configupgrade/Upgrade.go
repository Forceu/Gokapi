package configupgrade

import (
	"fmt"
	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/models"
	"os"
)

// CurrentConfigVersion is the version of the configuration structure. Used for upgrading
const CurrentConfigVersion = 22

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
	// < v1.9.0
	if settings.ConfigVersion < 21 {
		fmt.Println("Please update to version 1.9.6 before running this version.")
		osExit(1)
		return
	}
	// < v2.0.0
	if settings.ConfigVersion < 22 {
		if settings.Authentication.Method == models.AuthenticationOAuth2 || settings.Authentication.Method == models.AuthenticationHeader {
			adminUser := os.Getenv("GOKAPI_ADMIN_USER")
			if adminUser == "" {
				fmt.Println("FAILED UPDATE")
				fmt.Println("--> If using Oauth or Header authentication, please set the env variable GOKAPI_ADMIN_USER to the value of the expected user name / email")
				fmt.Println("--> See the release notes for more information")
				osExit(1)
				return
			} else {
				fmt.Println("Setting admin user to " + adminUser)
				if settings.Authentication.Method == models.AuthenticationOAuth2 {
					settings.Authentication.OAuthAdminUser = adminUser
				} else {
					settings.Authentication.HeaderAdminUser = adminUser
				}
			}
		}
	}
}

var osExit = os.Exit
