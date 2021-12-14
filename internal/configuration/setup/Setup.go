package setup

import (
	"Gokapi/internal/configuration"
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
	"time"
)

// webserverDir is the embedded version of the "static" folder
// This contains JS files, CSS, images etc for the setup
//go:embed static
var webserverDirEmb embed.FS
var srv http.Server

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
	return false, errors.New("could not convert " + value + " to bool")
}

func getFormValueInt(formObjects *[]jsonFormObject, key string) (int, error) {
	value, err := getFormValueString(formObjects, key)
	if err != nil {
		return 0, err
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New("could not convert " + value + " to int")
	}
	return result, nil
}

func toConfiguration(formObjects *[]jsonFormObject) (configuration.Configuration, error) {
	var err error
	parsedEnv := environment.New()

	result := configuration.Configuration{
		DefaultDownloads: 1,
		DefaultExpiry:    14,
		MaxFileSizeMB:    parsedEnv.MaxFileSize,
		LengthId:         parsedEnv.LengthId,
		MaxMemory:        parsedEnv.MaxMemory,
		DataDir:          parsedEnv.DataDir,
		SaltFiles:        helper.GenerateRandomString(30),
		SaltAdmin:        helper.GenerateRandomString(30),
		Sessions:         make(map[string]models.Session),
		Files:            make(map[string]models.File),
		Hotlinks:         make(map[string]models.Hotlink),
		DownloadStatus:   make(map[string]models.DownloadStatus),
		ApiKeys:          make(map[string]models.ApiKey),
		ConfigVersion:    configuration.CurrentConfigVersion,
	}

	result.AdminName, err = getFormValueString(formObjects, "auth_username")
	if err != nil {
		return configuration.Configuration{}, err
	}

	result.AdminPassword, err = getFormValueString(formObjects, "auth_pw")
	if err != nil {
		return configuration.Configuration{}, err
	}
	result.AdminPassword = configuration.HashPasswordCustomSalt(result.AdminPassword, result.SaltAdmin)

	port, err := getFormValueInt(formObjects, "port")
	if err != nil {
		return configuration.Configuration{}, err
	}
	bindLocalhost, err := getFormValueBool(formObjects, "localhost_sel")
	if err != nil {
		return configuration.Configuration{}, err
	}
	if bindLocalhost {
		result.Port = "127.0.0.1:" + strconv.Itoa(port)
	} else {
		result.Port = ":" + strconv.Itoa(port)
	}

	result.ServerUrl, err = getFormValueString(formObjects, "url")
	if err != nil {
		return configuration.Configuration{}, err
	}
	result.RedirectUrl, err = getFormValueString(formObjects, "url_redirection")
	if err != nil {
		return configuration.Configuration{}, err
	}
	result.LoginHeaderKey, err = getFormValueString(formObjects, "auth_headerkey")
	if err != nil {
		return configuration.Configuration{}, err
	}
	result.UseSsl, err = getFormValueBool(formObjects, "ssl_sel")
	if err != nil {
		return configuration.Configuration{}, err
	}

	result.AuthenticationMethod, err = getFormValueInt(formObjects, "authentication_sel")
	if err != nil {
		return configuration.Configuration{}, err
	}

	return result, nil
}

type setupResponse2 struct {
	AuthHeaderUsers      string `json:"auth_header_users"`
	StorageMethod        bool   `json:"storage_sel"`
	S3Bucket             string `json:"&s3_bucket"`
	S3Region             string `json:"s3_region"`
	S3Api                string `json:"s3_api"`
	S3Secret             string `json:"s3_secret"`
	S3Endpoint           string `json:"s3_endpoint"`
}

// Handling of /setup
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
