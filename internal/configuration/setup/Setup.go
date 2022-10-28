package setup

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/cloudconfig"
	"github.com/forceu/gokapi/internal/configuration/configupgrade"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/encryption"
	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage"
	"github.com/forceu/gokapi/internal/storage/cloudstorage/aws"
	"github.com/forceu/gokapi/internal/webserver/authentication"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// webserverDir is the embedded version of the "static" folder
// This contains JS files, CSS, images etc for the setup
//
//go:embed static
var webserverDirEmb embed.FS

// templateFolderEmbedded is the embedded version of the "templates" folder
// This contains templates that Gokapi uses for creating the HTML output
//
//go:embed templates
var templateFolderEmbedded embed.FS

// protectedUrls contains a list of URLs that need to be protected if authentication is disabled.
// This list will be displayed during the setup
var protectedUrls = []string{"/admin", "/apiDelete", "/apiKeys", "/apiNew", "/delete", "/e2eInfo", "/e2eSetup", "/uploadChunk", "/uploadComplete"}

var srv http.Server
var isInitialSetup = true
var username string
var password string

// statusChannel is only used for testing to indicate to the unit test that the server has been started or shut down.
var statusChannel chan bool = nil

const debugDisableAuth = false

// RunIfFirstStart checks if config files exist and if not start a blocking webserver for setup
func RunIfFirstStart() {
	if !configuration.Exists() {
		isInitialSetup = true
		startSetupWebserver()
	}
}

// RunConfigModification starts a blocking webserver for reconfiguration setup
func RunConfigModification() {
	isInitialSetup = false
	username = helper.GenerateRandomString(6)
	password = helper.GenerateRandomString(10)
	fmt.Println()
	fmt.Println("###################################################################")
	fmt.Println("Use the following credentials for modifying the configuration:")
	fmt.Println("Username: " + username)
	fmt.Println("Password: " + password)
	fmt.Println("###################################################################")
	fmt.Println()
	startSetupWebserver()
}

// basicAuth adds authentication middleware used for reconfiguration setup
func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// No auth required on initial setup
		if isInitialSetup || debugDisableAuth {
			next.ServeHTTP(w, r)
			return
		}

		enteredUser, enteredPw, ok := r.BasicAuth()
		if ok {
			usernameMatch := authentication.IsEqualStringConstantTime(enteredUser, username)
			passwordMatch := authentication.IsEqualStringConstantTime(enteredPw, password)
			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}
		time.Sleep(time.Second)
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
}

func startSetupWebserver() {
	port := strconv.Itoa(environment.New().WebserverPort)
	webserverDir, _ := fs.Sub(webserverDirEmb, "static")

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleShowMaintenance)
	mux.Handle("/setup/", http.FileServer(http.FS(webserverDir)))
	mux.HandleFunc("/setup/start", basicAuth(handleShowSetup))
	mux.HandleFunc("/setup/setupResult", basicAuth(handleResult))
	mux.HandleFunc("/setup/testaws", basicAuth(handleTestAws))

	srv = http.Server{
		Addr:         ":" + port,
		ReadTimeout:  2 * time.Minute,
		WriteTimeout: 2 * time.Minute,
		Handler:      mux,
	}
	fmt.Println("Please open http://" + resolveHostIp() + ":" + port + "/setup to setup Gokapi.")
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		if isErrorAddressAlreadyInUse(err) {
			fmt.Println("This port is already in use. Use parameter -p or env variable GOKAPI_PORT to change the port.")
		}
		log.Fatalf("Setup Webserver: %v", err)
	}
	if statusChannel != nil {
		go func() {
			// Normally only one second should be enough, however slow Github action runners require more time
			time.Sleep(5 * time.Second)
			statusChannel <- true
		}()
	}
	err = srv.Serve(listener)
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("Setup Webserver: %v", err)
	}
	if statusChannel != nil {
		statusChannel <- false
	}
}

func isErrorAddressAlreadyInUse(err error) bool {
	var eOsSyscall *os.SyscallError
	if !errors.As(err, &eOsSyscall) {
		return false
	}
	var errErrno syscall.Errno
	if !errors.As(eOsSyscall, &errErrno) {
		return false
	}
	if errErrno == syscall.EADDRINUSE {
		return true
	}
	const WSAEADDRINUSE = 10048
	if runtime.GOOS == "windows" && errErrno == WSAEADDRINUSE {
		return true
	}
	return false
}

func resolveHostIp() string {
	netInterfaceAddresses, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	for _, netInterfaceAddress := range netInterfaceAddresses {
		networkIp, ok := netInterfaceAddress.(*net.IPNet)
		if ok && !networkIp.IP.IsLoopback() && networkIp.IP.To4() != nil {
			ip := networkIp.IP.String()
			return ip
		}
	}
	return "127.0.0.1"
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

func toConfiguration(formObjects *[]jsonFormObject) (models.Configuration, *cloudconfig.CloudConfig, error) {
	var err error
	parsedEnv := environment.New()

	result := models.Configuration{
		MaxFileSizeMB:  parsedEnv.MaxFileSize,
		LengthId:       parsedEnv.LengthId,
		MaxMemory:      parsedEnv.MaxMemory,
		DataDir:        parsedEnv.DataDir,
		ConfigVersion:  configupgrade.CurrentConfigVersion,
		Authentication: models.AuthenticationConfig{},
	}

	if isInitialSetup {
		result.Authentication.SaltFiles = helper.GenerateRandomString(30)
		result.Authentication.SaltAdmin = helper.GenerateRandomString(30)
	} else {
		result.Authentication = configuration.Get().Authentication
	}

	err = parseBasicAuthSettings(&result, formObjects)
	if err != nil {
		return models.Configuration{}, nil, err
	}

	err = parseOAuthSettings(&result, formObjects)
	if err != nil {
		return models.Configuration{}, nil, err
	}

	err = parseHeaderAuthSettings(&result, formObjects)
	if err != nil {
		return models.Configuration{}, nil, err
	}

	err = parseServerSettings(&result, formObjects)
	if err != nil {
		return models.Configuration{}, nil, err
	}

	err = parseEncryptionAndDelete(&result, formObjects)
	if err != nil {
		return models.Configuration{}, nil, err
	}

	var cloudSettings *cloudconfig.CloudConfig
	cloudSettings, err = parseCloudSettings(formObjects)
	if err != nil {
		return models.Configuration{}, nil, err
	}

	return result, cloudSettings, nil
}

func parseBasicAuthSettings(result *models.Configuration, formObjects *[]jsonFormObject) error {
	var err error
	result.Authentication.Username, err = getFormValueString(formObjects, "auth_username")
	if err != nil {
		return err
	}

	pw, err := getFormValueString(formObjects, "auth_pw")
	if err != nil {
		return err
	}
	// Password is not displayed in reconf setup, but a placeholder "unc". If this is submitted as a password, the
	// old password is kept
	if isInitialSetup || pw != "unc" {
		result.Authentication.SaltAdmin = helper.GenerateRandomString(30)
		result.Authentication.Password = configuration.HashPasswordCustomSalt(pw, result.Authentication.SaltAdmin)
	}
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
	result.Authentication.OauthUsers = splitAndTrim(oauthAllowedUsers)
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
	result.Authentication.HeaderUsers = splitAndTrim(headerAllowedUsers)
	return nil
}

func parseServerSettings(result *models.Configuration, formObjects *[]jsonFormObject) error {
	var err error
	port, err := getFormValueInt(formObjects, "port")
	if err != nil {
		return err
	}
	port = verifyPortNumber(port)
	bindLocalhost := false
	if !environment.IsDockerInstance() {
		bindLocalhost, err = getFormValueBool(formObjects, "localhost_sel")
		if err != nil {
			return err
		}
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
	if result.Authentication.Method < 0 || result.Authentication.Method > 3 {
		return errors.New("invalid authentication mode provided")
	}

	result.ServerUrl = addTrailingSlash(result.ServerUrl)

	picturesAlwaysLocal, err := getFormValueString(formObjects, "storage_sel_image")
	if err != nil {
		return err
	}
	result.PicturesAlwaysLocal = picturesAlwaysLocal == "local"
	return nil
}

func parseCloudSettings(formObjects *[]jsonFormObject) (*cloudconfig.CloudConfig, error) {
	useCloud, err := getFormValueString(formObjects, "storage_sel")
	if err != nil {
		return nil, err
	}
	if useCloud == "cloud" {
		return getCloudConfig(formObjects)
	}
	return nil, nil
}

func getCloudConfig(formObjects *[]jsonFormObject) (*cloudconfig.CloudConfig, error) {
	var err error
	awsConfig := cloudconfig.CloudConfig{}
	awsConfig.Aws.Bucket, err = getFormValueString(formObjects, "s3_bucket")
	if err != nil {
		return nil, err
	}
	awsConfig.Aws.Region, err = getFormValueString(formObjects, "s3_region")
	if err != nil {
		return nil, err
	}
	awsConfig.Aws.KeyId, err = getFormValueString(formObjects, "s3_api")
	if err != nil {
		return nil, err
	}
	awsConfig.Aws.KeySecret, err = getFormValueString(formObjects, "s3_secret")
	if err != nil {
		return nil, err
	}
	awsConfig.Aws.Endpoint, err = getFormValueString(formObjects, "s3_endpoint")
	if err != nil {
		return nil, err
	}
	return &awsConfig, nil
}

func encryptionHasChanged(encLevel int, formObjects *[]jsonFormObject) (bool, error) {
	if encLevel != configuration.Get().Encryption.Level {
		return true, nil
	}
	if encLevel == encryption.LocalEncryptionInput || encLevel == encryption.FullEncryptionInput {
		masterPw, err := getFormValueString(formObjects, "enc_pw")
		if err != nil {
			return true, err
		}
		return masterPw != "unc", nil
	}
	return false, nil
}

func parseEncryptionAndDelete(result *models.Configuration, formObjects *[]jsonFormObject) error {
	encLevel, err := parseEncryptionLevel(formObjects)
	if err != nil {
		return err
	}

	generateNewEncConfig := true

	if !isInitialSetup {
		generateNewEncConfig, err = encryptionHasChanged(encLevel, formObjects)
		if err != nil {
			return err
		}
		if encLevel == encryption.EndToEndEncryption {
			deleteE2eInfo, _ := getFormValueString(formObjects, "cleare2e")
			if deleteE2eInfo == "true" {
				database.DeleteEnd2EndInfo()
			}
		}
	}

	if !generateNewEncConfig {
		result.Encryption = configuration.Get().Encryption
		return nil
	}
	if !isInitialSetup {
		storage.DeleteAllEncrypted()
	}

	result.Encryption = models.Encryption{}
	if encLevel == encryption.LocalEncryptionStored || encLevel == encryption.FullEncryptionStored {
		cipher, err := encryption.GetRandomCipher()
		if err != nil {
			return err
		}
		result.Encryption.Cipher = cipher
	}

	masterPw, err := getFormValueString(formObjects, "enc_pw")
	if err != nil {
		return err
	}
	if encLevel == encryption.LocalEncryptionInput || encLevel == encryption.FullEncryptionInput {
		result.Encryption.Salt = helper.GenerateRandomString(30)
		result.Encryption.ChecksumSalt = helper.GenerateRandomString(30)
		const minLength = 8
		if len(masterPw) < minLength {
			return errors.New("password is less than " + strconv.Itoa(minLength) + " characters long")
		}
		result.Encryption.Checksum = encryption.PasswordChecksum(masterPw, result.Encryption.ChecksumSalt)
	}
	result.Encryption.Level = encLevel
	return nil
}

func parseEncryptionLevel(formObjects *[]jsonFormObject) (int, error) {
	encLevelStr, err := getFormValueString(formObjects, "encrypt_sel")
	if err != nil {
		return 0, err
	}
	encLevel, err := strconv.Atoi(encLevelStr)
	if err != nil {
		return 0, err
	}
	if encLevel < encryption.NoEncryption || encLevel > encryption.EndToEndEncryption {
		return 0, errors.New("invalid encryption level selected")
	}
	return encLevel, nil
}

func inputToJsonForm(r *http.Request) ([]jsonFormObject, error) {
	reader, _ := io.ReadAll(r.Body)
	var setupResult []jsonFormObject
	err := json.Unmarshal(reader, &setupResult)
	if err != nil {
		return nil, err
	}
	return setupResult, nil
}

func splitAndTrim(input string) []string {
	arr := strings.Split(input, ";")
	var result []string
	for i := range arr {
		arr[i] = strings.TrimSpace(arr[i])
		if arr[i] != "" {
			result = append(result, arr[i])
		}
	}
	return result
}

type setupView struct {
	IsInitialSetup bool
	LocalhostOnly  bool
	HasAwsFeature  bool
	IsDocker       bool
	Port           int
	OAuthUsers     string
	HeaderUsers    string
	Auth           models.AuthenticationConfig
	Settings       models.Configuration
	CloudSettings  cloudconfig.CloudConfig
	ProtectedUrls  []string
}

func (v *setupView) loadFromConfig() {
	v.IsInitialSetup = isInitialSetup
	v.IsDocker = environment.IsDockerInstance()
	v.HasAwsFeature = aws.IsIncludedInBuild
	v.ProtectedUrls = protectedUrls
	if isInitialSetup {
		return
	}
	configuration.Load()
	settings := configuration.Get()
	v.Settings = *settings
	v.Auth = settings.Authentication
	v.CloudSettings, _ = cloudconfig.Load()
	v.OAuthUsers = strings.Join(settings.Authentication.OauthUsers, ";")
	v.HeaderUsers = strings.Join(settings.Authentication.HeaderUsers, ";")

	if strings.Contains(settings.Port, "localhost") || strings.Contains(settings.Port, "127.0.0.1") {
		v.LocalhostOnly = true
	}
	portArray := strings.SplitAfter(settings.Port, ":")
	port, err := strconv.Atoi(portArray[len(portArray)-1])
	if err == nil {
		v.Port = port
	} else {
		v.Port = environment.DefaultPort
	}
}

// Handling of /start
func handleShowSetup(w http.ResponseWriter, r *http.Request) {
	templateFolder, err := template.ParseFS(templateFolderEmbedded, "templates/*.tmpl")
	helper.Check(err)
	view := setupView{}
	view.loadFromConfig()
	err = templateFolder.ExecuteTemplate(w, "setup", view)
	helper.Check(err)
}

func handleShowMaintenance(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Server is in maintenance mode, please try again in a few minutes."))
}

// Handling of /setupResult
func handleResult(w http.ResponseWriter, r *http.Request) {
	setupResult, err := inputToJsonForm(r)
	if err != nil {
		outputError(w, err)
		return
	}

	newConfig, cloudSettings, err := toConfiguration(&setupResult)
	if err != nil {
		outputError(w, err)
		return
	}
	configuration.LoadFromSetup(newConfig, cloudSettings, isInitialSetup)
	w.WriteHeader(200)
	w.Write([]byte("{ \"result\": \"OK\"}"))
	go func() {
		time.Sleep(1500 * time.Millisecond)
		srv.Shutdown(context.Background())
	}()
}

func outputError(w http.ResponseWriter, err error) {
	w.WriteHeader(500)
	w.Write([]byte("{ \"result\": \"Error\", \"error\": \"" + err.Error() + "\"}"))
}

// Adds a / character to the end of a URL if it does not exist
func addTrailingSlash(url string) string {
	if !strings.HasSuffix(url, "/") {
		return url + "/"
	}
	return url
}

func verifyPortNumber(port int) int {
	if port < 0 || port > 65535 {
		return environment.DefaultPort
	}
	return port
}

type testAwsRequest struct {
	Bucket    string `json:"bucket"`
	Region    string `json:"region"`
	ApiKey    string `json:"apikey"`
	ApiSecret string `json:"apisecret"`
	Endpoint  string `json:"endpoint"`
	GokapiUrl string `json:"exturl"`
}

// Handling of /testaws
func handleTestAws(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var t testAwsRequest
	err := decoder.Decode(&t)
	if err != nil {
		w.Write([]byte("Error: " + err.Error()))
		return
	}
	awsconfig := models.AwsConfig{
		Bucket:    t.Bucket,
		Region:    t.Region,
		KeyId:     t.ApiKey,
		KeySecret: t.ApiSecret,
		Endpoint:  t.Endpoint,
	}
	ok, err := aws.IsValidLogin(awsconfig)
	if err != nil {
		w.Write([]byte("Error: " + err.Error()))
		return
	}
	if !ok {
		w.Write([]byte("Error: Invalid or incomplete credentials provided"))
		return
	}
	aws.Init(awsconfig)
	ok, err = aws.IsCorsCorrectlySet(t.Bucket, t.GokapiUrl)
	aws.LogOut()
	if err != nil {
		w.Write([]byte("Could not check CORS settings. Error: " + err.Error()))
		return
	}
	if !ok {
		w.Write([]byte("Test OK, however CORS settings do not allow encrypted downloads."))
		return
	}
	w.Write([]byte("Test OK, CORS settings allow encrypted downloads."))
}
