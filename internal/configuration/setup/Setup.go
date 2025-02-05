package setup

import (
	"bufio"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/cloudconfig"
	"github.com/forceu/gokapi/internal/configuration/configupgrade"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/configuration/database/dbabstraction"
	"github.com/forceu/gokapi/internal/encryption"
	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage/filesystem/s3filesystem/aws"
	"github.com/forceu/gokapi/internal/webserver/authentication"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
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

var srv http.Server
var isInitialSetup = true
var credentialUsername string
var credentialPassword string

// debugDisableAuth can be set to true for testing purposes. It will disable the
// password requirement for accessing the setup page
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
	credentialUsername = helper.GenerateRandomString(6)
	credentialPassword = helper.GenerateRandomString(10)
	fmt.Println()
	fmt.Println("###################################################################")
	fmt.Println("Use the following credentials for modifying the configuration:")
	fmt.Println("Username: " + credentialUsername)
	fmt.Println("Password: " + credentialPassword)
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
			usernameMatch := authentication.IsEqualStringConstantTime(enteredUser, credentialUsername)
			passwordMatch := authentication.IsEqualStringConstantTime(enteredPw, credentialPassword)
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
	if debugDisableAuth {
		srv.Addr = "127.0.0.1:" + port
		fmt.Println("Authentication is disabled by debug flag. Setup only accessible by localhost")
		fmt.Println("Please open http://127.0.0.1:" + port + "/setup to setup Gokapi.")
	} else {
		fmt.Println("Please open http://" + resolveHostIp() + ":" + port + "/setup to setup Gokapi.")
	}
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		if isErrorAddressAlreadyInUse(err) {
			fmt.Println("This port is already in use. Use parameter -p or env variable GOKAPI_PORT to change the port.")
		}
		log.Fatalf("Setup Webserver: %v", err)
	}
	err = srv.Serve(listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Setup Webserver: %v", err)
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
	if errors.Is(errErrno, syscall.EADDRINUSE) {
		return true
	}
	const WSAEADDRINUSE = 10048
	//noinspection GoBoolExpressions
	if runtime.GOOS == "windows" && errErrno == WSAEADDRINUSE {
		return true
	}
	return false
}

func resolveHostIp() string {
	const localhost = "127.0.0.1"
	if environment.IsDockerInstance() {
		return localhost
	}
	netInterfaceAddresses, err := net.InterfaceAddrs()
	if err != nil {
		return localhost
	}

	for _, netInterfaceAddress := range netInterfaceAddresses {
		networkIp, ok := netInterfaceAddress.(*net.IPNet)
		if ok && !networkIp.IP.IsLoopback() && networkIp.IP.To4() != nil {
			ip := networkIp.IP.String()
			return ip
		}
	}
	return localhost
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
		valueHidden, err2 := getFormValueString(formObjects, key+".unchecked")
		if err2 != nil {
			return false, err
		}
		value = valueHidden
	}
	if value == "0" || value == "false" {
		return false, nil
	}
	if value == "1" || value == "true" {
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

type authSettings struct {
	UserInternalAuth          string
	UserOAuth                 string
	UserHeader                string
	PasswordInternalAuth      string
	OnlyRegisteredUsersOAuth  bool
	OnlyRegisteredUsersHeader bool
}

func toConfiguration(formObjects *[]jsonFormObject) (models.Configuration, *cloudconfig.CloudConfig, configuration.End2EndReconfigParameters, authSettings, error) {
	var err error
	var e2eConfig configuration.End2EndReconfigParameters
	parsedEnv := environment.New()

	result := models.Configuration{
		MaxFileSizeMB:      parsedEnv.MaxFileSize,
		LengthId:           parsedEnv.LengthId,
		MaxMemory:          parsedEnv.MaxMemory,
		DataDir:            parsedEnv.DataDir,
		MaxParallelUploads: parsedEnv.MaxParallelUploads,
		ChunkSize:          parsedEnv.ChunkSizeMB,
		ConfigVersion:      configupgrade.CurrentConfigVersion,
		Authentication:     models.AuthenticationConfig{},
	}
	authInfo := authSettings{}

	if isInitialSetup {
		result.Authentication.SaltFiles = helper.GenerateRandomString(30)
		result.Authentication.SaltAdmin = helper.GenerateRandomString(30)
	} else {
		result.Authentication = configuration.Get().Authentication
	}

	err = parseDatabaseSettings(&result, formObjects)
	if err != nil {
		return models.Configuration{}, nil, configuration.End2EndReconfigParameters{}, authSettings{}, err
	}

	err = parseBasicAuthSettings(&result, &authInfo, formObjects)
	if err != nil {
		return models.Configuration{}, nil, configuration.End2EndReconfigParameters{}, authSettings{}, err
	}

	err = parseOAuthSettings(&result, &authInfo, formObjects)
	if err != nil {
		return models.Configuration{}, nil, configuration.End2EndReconfigParameters{}, authSettings{}, err
	}

	err = parseHeaderAuthSettings(&result, &authInfo, formObjects)
	if err != nil {
		return models.Configuration{}, nil, configuration.End2EndReconfigParameters{}, authSettings{}, err
	}

	err = parseServerSettings(&result, formObjects)
	if err != nil {
		return models.Configuration{}, nil, configuration.End2EndReconfigParameters{}, authSettings{}, err
	}

	e2eConfig, err = parseEncryptionAndDelete(&result, formObjects)
	if err != nil {
		return models.Configuration{}, nil, configuration.End2EndReconfigParameters{}, authSettings{}, err
	}

	var cloudSettings *cloudconfig.CloudConfig
	cloudSettings, err = parseCloudSettings(formObjects)
	if err != nil {
		return models.Configuration{}, nil, configuration.End2EndReconfigParameters{}, authSettings{}, err
	}

	switch result.Authentication.Method {
	case models.AuthenticationInternal:
		result.Authentication.Username = authInfo.UserInternalAuth
	case models.AuthenticationOAuth2:
		result.Authentication.Username = authInfo.UserOAuth
		result.Authentication.OnlyRegisteredUsers = authInfo.OnlyRegisteredUsersOAuth
	case models.AuthenticationHeader:
		result.Authentication.Username = authInfo.UserHeader
		result.Authentication.OnlyRegisteredUsers = authInfo.OnlyRegisteredUsersHeader
	case models.AuthenticationDisabled:
		result.Authentication.Username = "admin@gokapi"
	}
	result.Authentication.Username = strings.ToLower(result.Authentication.Username)

	return result, cloudSettings, e2eConfig, authInfo, nil
}

func parseDatabaseSettings(result *models.Configuration, formObjects *[]jsonFormObject) error {
	dbType, err := getFormValueInt(formObjects, "dbtype_sel")
	if err != nil {
		return err
	}
	err = checkForAllDbValues(formObjects)
	if err != nil {
		return err
	}
	switch dbType {
	case dbabstraction.TypeSqlite:
		location, err := getFormValueString(formObjects, "sqlite_location")
		if err != nil {
			return err
		}
		result.DatabaseUrl = "sqlite://" + location
		return nil
	case dbabstraction.TypeRedis:
		host, err := getFormValueString(formObjects, "redis_location")
		if err != nil {
			return err
		}
		prefix, err := getFormValueString(formObjects, "redis_prefix")
		if err != nil {
			return err
		}
		rUser, err := getFormValueString(formObjects, "redis_user")
		if err != nil {
			return err
		}
		rPassword, err := getFormValueString(formObjects, "redis_password")
		if err != nil {
			return err
		}
		useSsl, err := getFormValueBool(formObjects, "redis_ssl_sel")
		if err != nil {
			return err
		}
		dbUrl := url.URL{
			Scheme: "redis",
			Host:   host,
		}
		query := url.Values{}
		if prefix != "" {
			query.Set("prefix", prefix)
		}
		if useSsl {
			query.Set("ssl", "true")
		}
		if rUser != "" || rPassword != "" {
			dbUrl.User = url.UserPassword(rUser, rPassword)
		}
		dbUrl.RawQuery = query.Encode()
		result.DatabaseUrl = dbUrl.String()
		return nil
	default:
		return errors.New("unsupported database selected")
	}
}

// checkForAllDbValues tests if all values were passed, even if they were not required for this particular database
// This is done to ensure that no invalid form was passed and makes testing easier
func checkForAllDbValues(formObjects *[]jsonFormObject) error {
	expectedValues := []string{"dbtype_sel", "sqlite_location", "redis_location", "redis_prefix", "redis_user", "redis_password"}
	for _, value := range expectedValues {
		_, err := getFormValueString(formObjects, value)
		if err != nil {
			return err
		}
	}
	_, err := getFormValueBool(formObjects, "redis_ssl_sel")
	return err
}

func parseBasicAuthSettings(result *models.Configuration, authInfo *authSettings, formObjects *[]jsonFormObject) error {
	username, err := getFormValueString(formObjects, "auth_username")
	if err != nil {
		return err
	}
	authInfo.UserInternalAuth = username

	pw, err := getFormValueString(formObjects, "auth_pw")
	if err != nil {
		return err
	}
	// Password is not displayed in reconf setup, but a placeholder "unc". If this is submitted as a password, the
	// old password is kept
	if isInitialSetup {
		result.Authentication.SaltAdmin = helper.GenerateRandomString(30)
	}
	if isInitialSetup || pw != "unc" {
		authInfo.PasswordInternalAuth = configuration.HashPasswordCustomSalt(pw, result.Authentication.SaltAdmin)
	}
	return nil
}

func parseOAuthSettings(result *models.Configuration, authInfo *authSettings, formObjects *[]jsonFormObject) error {
	var err error
	result.Authentication.OAuthProvider, err = getFormValueString(formObjects, "oauth_provider")
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

	result.Authentication.OAuthRecheckInterval, err = getFormValueInt(formObjects, "oauth_recheck_interval")
	if err != nil {
		return err
	}

	username, err := getFormValueString(formObjects, "oauth_admin_user")
	if err != nil {
		return err
	}
	authInfo.UserOAuth = username

	authInfo.OnlyRegisteredUsersOAuth, err = getFormValueBool(formObjects, "oauth_only_registered_users")
	if err != nil {
		return err
	}

	restrictGroups, err := getFormValueBool(formObjects, "oauth_restrict_groups")
	if err != nil {
		return err
	}
	oauthAllowedGroups, err := getFormValueString(formObjects, "oauth_allowed_groups")
	if err != nil {
		return err
	}
	scopeGroups, err := getFormValueString(formObjects, "oauth_scope_groups")
	if err != nil {
		return err
	}
	if restrictGroups {
		result.Authentication.OAuthGroupScope = scopeGroups
		result.Authentication.OAuthGroups = splitAndTrim(oauthAllowedGroups)
	} else {
		result.Authentication.OAuthGroups = []string{}
		result.Authentication.OAuthGroupScope = ""
	}

	return nil
}

func parseHeaderAuthSettings(result *models.Configuration, authInfo *authSettings, formObjects *[]jsonFormObject) error {
	username, err := getFormValueString(formObjects, "auth_header_admin")
	if err != nil {
		return err
	}
	authInfo.UserHeader = username
	result.Authentication.HeaderKey, err = getFormValueString(formObjects, "auth_headerkey")
	if err != nil {
		return err
	}

	authInfo.OnlyRegisteredUsersHeader, err = getFormValueBool(formObjects, "auth_header_only_registered_users")
	if err != nil {
		return err
	}
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

	result.PublicName, err = getFormValueString(formObjects, "public_name")
	if err != nil {
		return err
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
	result.SaveIp, err = getFormValueBool(formObjects, "logip_sel")
	if err != nil {
		return err
	}
	result.IncludeFilename, err = getFormValueBool(formObjects, "showfilename_sel")
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
	awsConfig := cloudconfig.CloudConfig{}
	proxyDownload, err := getFormValueString(formObjects, "storage_sel_proxy")
	if err != nil {
		return nil, err
	}
	awsConfig.Aws.ProxyDownload = proxyDownload == "proxy"
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

func parseEncryptionAndDelete(result *models.Configuration, formObjects *[]jsonFormObject) (configuration.End2EndReconfigParameters, error) {
	var e2eConfig configuration.End2EndReconfigParameters
	encLevel, err := parseEncryptionLevel(formObjects)
	if err != nil {
		return e2eConfig, err
	}

	generateNewEncConfig := true

	if !isInitialSetup {
		generateNewEncConfig, err = encryptionHasChanged(encLevel, formObjects)
		if err != nil {
			return configuration.End2EndReconfigParameters{}, err
		}
		if encLevel == encryption.EndToEndEncryption {
			deleteE2eInfo, _ := getFormValueString(formObjects, "cleare2e")
			if deleteE2eInfo == "true" {
				e2eConfig.DeleteEnd2EndEncryption = true
			}
		}
	}

	if !generateNewEncConfig {
		result.Encryption = configuration.Get().Encryption
		return e2eConfig, nil
	}
	if !isInitialSetup {
		e2eConfig.DeleteEncryptedStorage = true
	}

	result.Encryption = models.Encryption{}
	if encLevel == encryption.LocalEncryptionStored || encLevel == encryption.FullEncryptionStored {
		cipher, err := encryption.GetRandomCipher()
		if err != nil {
			return configuration.End2EndReconfigParameters{}, err
		}
		result.Encryption.Cipher = cipher
	}

	masterPw, err := getFormValueString(formObjects, "enc_pw")
	if err != nil {
		return configuration.End2EndReconfigParameters{}, err
	}
	if encLevel == encryption.LocalEncryptionInput || encLevel == encryption.FullEncryptionInput {
		result.Encryption.Salt = helper.GenerateRandomString(30)
		result.Encryption.ChecksumSalt = helper.GenerateRandomString(30)
		if len(masterPw) < configuration.MinLengthPassword {
			return configuration.End2EndReconfigParameters{}, errors.New("password is less than " + strconv.Itoa(configuration.MinLengthPassword) + " characters long")
		}
		result.Encryption.Checksum = encryption.PasswordChecksum(masterPw, result.Encryption.ChecksumSalt)
	}
	result.Encryption.Level = encLevel
	return e2eConfig, nil
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
	IsInitialSetup     bool
	LocalhostOnly      bool
	HasAwsFeature      bool
	IsDocker           bool
	S3EnvProvided      bool
	IsDataNotMounted   bool
	IsConfigNotMounted bool
	Port               int
	OAuthGroups        string
	Auth               models.AuthenticationConfig
	Settings           models.Configuration
	CloudSettings      cloudconfig.CloudConfig
	DatabaseSettings   models.DbConnection
	ProtectedUrls      []string
}

func (v *setupView) loadFromConfig() {
	v.IsInitialSetup = isInitialSetup
	if environment.IsDockerInstance() {
		v.IsDocker = true
		v.IsDataNotMounted = !isVolumeMounted("/app/data")
		v.IsConfigNotMounted = !isVolumeMounted("/app/config")
	}
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
	v.OAuthGroups = strings.Join(settings.Authentication.OAuthGroups, ";")

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
	env := environment.New()
	v.S3EnvProvided = env.IsAwsProvided()

	dbSettings, err := database.ParseUrl(settings.DatabaseUrl, false)
	helper.Check(err)
	v.DatabaseSettings = dbSettings
}

func isVolumeMounted(path string) bool {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) > 1 && fields[1] == path {
			return true
		}
	}
	return false
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
	_, _ = w.Write([]byte("Server is in maintenance mode, please try again in a few minutes."))
}

// Handling of /setupResult
func handleResult(w http.ResponseWriter, r *http.Request) {
	setupResult, err := inputToJsonForm(r)
	if err != nil {
		outputError(w, err)
		return
	}

	newConfig, cloudSettings, e2eConfig, authInfo, err := toConfiguration(&setupResult)
	if err != nil {
		outputError(w, err)
		return
	}
	configuration.LoadFromSetup(newConfig, cloudSettings, e2eConfig, authInfo.PasswordInternalAuth)
	w.WriteHeader(200)
	_, _ = w.Write([]byte("{ \"result\": \"OK\"}"))
	go func() {
		time.Sleep(1500 * time.Millisecond)
		err = srv.Shutdown(context.Background())
		if err != nil {
			fmt.Println(err)
		}
	}()
}

func outputError(w http.ResponseWriter, err error) {
	w.WriteHeader(500)
	_, _ = w.Write([]byte("{ \"result\": \"Error\", \"error\": \"" + err.Error() + "\"}"))
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
	Bucket      string `json:"bucket"`
	Region      string `json:"region"`
	ApiKey      string `json:"apikey"`
	ApiSecret   string `json:"apisecret"`
	Endpoint    string `json:"endpoint"`
	GokapiUrl   string `json:"exturl"`
	EnvProvided bool   `json:"isEnvProvided"`
}

// Handling of /testaws
func handleTestAws(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var t testAwsRequest
	err := decoder.Decode(&t)
	if err != nil {
		_, _ = w.Write([]byte("Error: " + err.Error()))
		return
	}
	var awsConfig models.AwsConfig

	if !t.EnvProvided {
		awsConfig = models.AwsConfig{
			Bucket:    t.Bucket,
			Region:    t.Region,
			KeyId:     t.ApiKey,
			KeySecret: t.ApiSecret,
			Endpoint:  t.Endpoint,
		}
	} else {
		env := environment.New()
		awsConfig = models.AwsConfig{
			Bucket:    env.AwsBucket,
			Region:    env.AwsRegion,
			KeyId:     env.AwsKeyId,
			KeySecret: env.AwsKeySecret,
			Endpoint:  env.AwsEndpoint,
		}
	}
	ok, err := aws.IsValidLogin(awsConfig)
	if err != nil {
		handleAwsError(w, err, "Unable to login. ")
		return
	}
	if !ok {
		_, _ = w.Write([]byte("Error: Invalid or incomplete credentials provided"))
		return
	}
	aws.Init(awsConfig)
	ok, err = aws.IsCorsCorrectlySet(t.Bucket, t.GokapiUrl)
	aws.LogOut()
	if err != nil {
		handleAwsError(w, err, "Could not get CORS settings. ")
		return
	}
	if !ok {
		_, _ = w.Write([]byte("Test OK. WARNING: CORS settings do not allow encrypted downloads."))
		return
	}
	_, _ = w.Write([]byte("All tests OK."))
}

func handleAwsError(w http.ResponseWriter, err error, prefix string) {
	var awsErr awserr.Error
	isAwsErr := errors.As(err, &awsErr)
	if isAwsErr {
		switch awsErr.Code() {
		case s3.ErrCodeNoSuchBucket:
			_, _ = w.Write([]byte("Invalid bucket or regions provided, bucket does not exist."))
		case "Forbidden":
			_, _ = w.Write([]byte("Unable to log in, invalid credentials."))
		case "RequestError":
			_, _ = w.Write([]byte("Unable to connect to server, check endpoint."))
		case "SerializationError":
			_, _ = w.Write([]byte("Invalid response received by server, check endpoint."))
		default:
			_, _ = w.Write([]byte(prefix + "Error " + awsErr.Code() + ": " + err.Error()))
		}
	} else {
		_, _ = w.Write([]byte(prefix + "Error: " + err.Error()))
	}
}
