package configupgrade

import (
	"encoding/json"
	"fmt"
	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"os"
	"strings"
)

// RequiresUpgradeV1ToV2 is an indicator for migrating the admin user to the database
//
// It will be removed in v2.1.0.
// Deprecated: This is a temporary solution
var RequiresUpgradeV1ToV2 = false

// LegacyPasswordHash is the hash, which was originally stored in
// AuthenticationConfig.Password and needs to be passed to the migration
//
// It will be removed in v2.1.0
// Deprecated: This is a temporary solution
var LegacyPasswordHash string

// LegacyUsersHeaderOauth is the user restriction from OAuth2 or Header authentication
//
//	and needs to be passed to the migration
//
// It will be removed in v2.1.0
// Deprecated: This is a temporary solution
var LegacyUsersHeaderOauth []string

// CurrentConfigVersion is the version of the configuration structure. Used for upgrading
const CurrentConfigVersion = 22

const minConfigVersion = 21

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
	if settings.ConfigVersion < minConfigVersion {
		fmt.Println("Please update to version 1.9.6 before running this version.")
		osExit(1)
		return
	}
	// < v2.0.0
	if settings.ConfigVersion < 22 {
		RequiresUpgradeV1ToV2 = true
		oldAuth, err := getLegacyAuth(env.ConfigPath)
		helper.Check(err)
		LegacyPasswordHash = oldAuth.Authentication.Password
		LegacyUsersHeaderOauth = make([]string, 0)

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
				settings.Authentication.Username = adminUser
			}
			if settings.Authentication.Method == models.AuthenticationOAuth2 && len(oldAuth.Authentication.OAuthUsers) > 0 {
				LegacyUsersHeaderOauth = oldAuth.Authentication.OAuthUsers
				settings.Authentication.OnlyRegisteredUsers = true
			}
			if settings.Authentication.Method == models.AuthenticationHeader && len(oldAuth.Authentication.HeaderUsers) > 0 {
				LegacyUsersHeaderOauth = oldAuth.Authentication.HeaderUsers
				settings.Authentication.OnlyRegisteredUsers = true
			}
			for _, user := range LegacyUsersHeaderOauth {
				if strings.Contains(user, "*") {
					fmt.Println("FAILED UPDATE")
					fmt.Println("--> If using Oauth or Header authentication and restricting the users, please remove any wildcards before upgrading.")
					fmt.Println("--> See the release notes for more information")
					osExit(1)
					return
				}
			}
		}
	}
}

func getLegacyAuth(configFile string) (legacyAuthentication, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return legacyAuthentication{}, err
	}
	decoder := json.NewDecoder(file)
	settings := legacyAuthentication{}
	err = decoder.Decode(&settings)
	if err != nil {
		return legacyAuthentication{}, err
	}
	return settings, nil
}

type legacyAuthentication struct {
	Authentication struct {
		Password    string   `json:"password"`
		HeaderUsers []string `json:"HeaderUsers"`
		OAuthUsers  []string `json:"OauthUsers"`
	}
}

var osExit = os.Exit
