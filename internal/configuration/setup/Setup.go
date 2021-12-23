package setup

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/configuration/cloudconfig"
	"Gokapi/internal/environment"
	"Gokapi/internal/helper"
	"Gokapi/internal/models"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// webserverDir is the embedded version of the "static" folder
// This contains JS files, CSS, images etc for the setup
//go:embed static
var webserverDirEmb embed.FS
var srv http.Server

// TODO Validation client side (eg url trailing slash, portnumber)

func RunIfFirstStart() {
	if !configuration.Exists() {
		startSetupWebserver()
	}
}

func startSetupWebserver() {
	port := environment.GetPort()
	webserverDir, _ := fs.Sub(webserverDirEmb, "static")
	http.Handle("/setup/", http.FileServer(http.FS(webserverDir)))
	http.HandleFunc("/setup/setupResult", handleResult)

	srv = http.Server{
		Addr:         ":" + port,
		ReadTimeout:  2 * time.Minute,
		WriteTimeout: 2 * time.Minute,
	}
	fmt.Println("Please open http://" + resolveHostIp() + ":" + port + "/setup to setup Gokapi.")
	// always returns error. ErrServerClosed on graceful close
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("ListenAndServe(): %v", err)
	}
}

func resolveHostIp() string {
	netInterfaceAddresses, err := net.InterfaceAddrs()
	if err != nil {
		return "[your server IP]"
	}

	for _, netInterfaceAddress := range netInterfaceAddresses {
		networkIp, ok := netInterfaceAddress.(*net.IPNet)
		if ok && !networkIp.IP.IsLoopback() && networkIp.IP.To4() != nil {
			ip := networkIp.IP.String()
			return ip
		}
	}
	return "[your server IP]"
}

type jsonFormObject struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func getFormValueString(formObjects *[]jsonFormObject, key string) (string, error) {
	for _, formObject := range *formObjects {
		if formObject.Name == key {
			return formObject.Value, nil
		}
	}
	return "", errors.New("missing value in submitted setup: " + key)
}

func getFormValueBool(formObjects *[]jsonFormObject, key string) (bool, error) {
	value, err := getFormValueString(formObjects, key)
	if err != nil {
		return false, err
	}
	if value == "0" {
		return false, nil
	}
	if value == "1" {
		return true, nil
	}
	return false, errors.New("could not convert " + key + " to bool, got: " + value)
}

func getFormValueInt(formObjects *[]jsonFormObject, key string) (int, error) {
	value, err := getFormValueString(formObjects, key)
	if err != nil {
		return 0, err
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New("could not convert " + key + " to int, got: " + value)
	}
	return result, nil
}

func toConfiguration(formObjects *[]jsonFormObject) (models.Configuration, error) {
	var err error
	parsedEnv := environment.New()

	result := models.Configuration{
		DefaultDownloads: 1,
		DefaultExpiry:    14,
		MaxFileSizeMB:    parsedEnv.MaxFileSize,
		LengthId:         parsedEnv.LengthId,
		MaxMemory:        parsedEnv.MaxMemory,
		DataDir:          parsedEnv.DataDir,
		Sessions:         make(map[string]models.Session),
		Files:            make(map[string]models.File),
		Hotlinks:         make(map[string]models.Hotlink),
		DownloadStatus:   make(map[string]models.DownloadStatus),
		ApiKeys:          make(map[string]models.ApiKey),
		ConfigVersion:    configuration.CurrentConfigVersion,
		Authentication: models.AuthenticationConfig{
			SaltAdmin: helper.GenerateRandomString(30),
			SaltFiles: helper.GenerateRandomString(30),
		},
	}

	err = parseBasicAuthSettings(&result, formObjects)
	if err != nil {
		return models.Configuration{}, err
	}

	err = parseOAuthSettings(&result, formObjects)
	if err != nil {
		return models.Configuration{}, err
	}

	err = parseHeaderAuthSettings(&result, formObjects)
	if err != nil {
		return models.Configuration{}, err
	}

	err = parseServerSettings(&result, formObjects)
	if err != nil {
		return models.Configuration{}, err
	}

	return result, nil
}

func parseBasicAuthSettings(result *models.Configuration, formObjects *[]jsonFormObject) error {
	var err error
	result.Authentication.Username, err = getFormValueString(formObjects, "auth_username")
	if err != nil {
		return err
	}

	result.Authentication.Password, err = getFormValueString(formObjects, "auth_pw")
	if err != nil {
		return err
	}
	result.Authentication.Password = configuration.HashPasswordCustomSalt(result.Authentication.Password, result.Authentication.SaltAdmin)
	return nil
}

func parseOAuthSettings(result *models.Configuration, formObjects *[]jsonFormObject) error {
	var err error
	result.Authentication.OauthProvider, err = getFormValueString(formObjects, "oauth_provider")
	if err != nil {
		return err
	}

	result.Authentication.OAuthClientId, err = getFormValueString(formObjects, "oauth_id")
	if err != nil {
		return err
	}

	result.Authentication.OAuthClientSecret, err = getFormValueString(formObjects, "oauth_secret")
	if err != nil {
		return err
	}

	oauthAllowedUsers, err := getFormValueString(formObjects, "oauth_header_users")
	if err != nil {
		return err
	}
	result.Authentication.OauthUsers = strings.Split(oauthAllowedUsers, ";")
	return nil
}

func parseHeaderAuthSettings(result *models.Configuration, formObjects *[]jsonFormObject) error {
	var err error
	result.Authentication.HeaderKey, err = getFormValueString(formObjects, "auth_headerkey")
	if err != nil {
		return err
	}

	headerAllowedUsers, err := getFormValueString(formObjects, "auth_header_users")
	if err != nil {
		return err
	}
	result.Authentication.HeaderUsers = strings.Split(headerAllowedUsers, ";")

	return nil
}

func parseServerSettings(result *models.Configuration, formObjects *[]jsonFormObject) error {
	var err error
	port, err := getFormValueInt(formObjects, "port")
	if err != nil {
		return err
	}
	bindLocalhost, err := getFormValueBool(formObjects, "localhost_sel")
	if err != nil {
		return err
	}
	if bindLocalhost {
		result.Port = "127.0.0.1:" + strconv.Itoa(port)
	} else {
		result.Port = ":" + strconv.Itoa(port)
	}

	result.ServerUrl, err = getFormValueString(formObjects, "url")
	if err != nil {
		return err
	}
	result.RedirectUrl, err = getFormValueString(formObjects, "url_redirection")
	if err != nil {
		return err
	}
	result.UseSsl, err = getFormValueBool(formObjects, "ssl_sel")
	if err != nil {
		return err
	}

	result.Authentication.Method, err = getFormValueInt(formObjects, "authentication_sel")
	if err != nil {
		return err
	}

	useCloud, err := getFormValueString(formObjects, "storage_sel")
	if err != nil {
		return err
	}
	if useCloud == "cloud" {
		err = writeCloudConfig(formObjects, result)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeCloudConfig(formObjects *[]jsonFormObject, config *models.Configuration) error {
	var err error
	awsConfig := cloudconfig.CloudConfig{}
	awsConfig.Aws.Bucket, err = getFormValueString(formObjects, "s3_bucket")
	if err != nil {
		return err
	}
	awsConfig.Aws.Region, err = getFormValueString(formObjects, "s3_region")
	if err != nil {
		return err
	}
	awsConfig.Aws.KeyId, err = getFormValueString(formObjects, "s3_api")
	if err != nil {
		return err
	}
	awsConfig.Aws.KeySecret, err = getFormValueString(formObjects, "s3_secret")
	if err != nil {
		return err
	}
	awsConfig.Aws.Endpoint, err = getFormValueString(formObjects, "s3_endpoint")
	if err != nil {
		return err
	}

	err = cloudconfig.Write(awsConfig)
	if err != nil {
		return err
	}

	return nil
}

// Handling of /setupResult
func handleResult(w http.ResponseWriter, r *http.Request) {
	reader, _ := io.ReadAll(r.Body)
	fmt.Println(string(reader))
	var setupResult []jsonFormObject
	err := json.Unmarshal(reader, &setupResult)
	if err != nil {
		outputError(w, err)
		return
	}
	newConfig, err := toConfiguration(&setupResult)
	if err != nil {
		outputError(w, err)
		return
	}
	configuration.LoadFromSetup(newConfig)
	w.WriteHeader(200)
	w.Write([]byte("{ \"result\": \"OK\"}"))
	go func() {
		time.Sleep(1 * time.Second)
		srv.Shutdown(context.Background())
	}()
}

func outputError(w http.ResponseWriter, err error) {
	w.WriteHeader(500)
	w.Write([]byte("{ \"result\": \"Error\", \"error\": \"" + err.Error() + "\"}"))
}
