package webserver

/**
Handling of webserver and requests / uploads
*/

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/NYTimes/gziphandler"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/encryption"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/logging"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage"
	"github.com/forceu/gokapi/internal/webserver/api"
	"github.com/forceu/gokapi/internal/webserver/authentication"
	"github.com/forceu/gokapi/internal/webserver/authentication/oauth"
	"github.com/forceu/gokapi/internal/webserver/authentication/sessionmanager"
	"github.com/forceu/gokapi/internal/webserver/fileupload"
	"github.com/forceu/gokapi/internal/webserver/sse"
	"github.com/forceu/gokapi/internal/webserver/ssl"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"sort"
	"strings"
	templatetext "text/template"
	"time"
)

// TODO add 404 handler
// staticFolderEmbedded is the embedded version of the "static" folder
// This contains JS files, CSS, images etc
//
//go:embed web/static
var staticFolderEmbedded embed.FS

// templateFolderEmbedded is the embedded version of the "templates" folder
// This contains templates that Gokapi uses for creating the HTML output
//
//go:embed web/templates
var templateFolderEmbedded embed.FS

// wasmDownloadFile is the compiled binary of the wasm downloader
// Will be generated with go generate ./...
//
//go:embed web/main.wasm
var wasmDownloadFile embed.FS

// wasmE2EFile is the compiled binary of the wasm e2e encrypter
// Will be generated with go generate ./...
//
//go:embed web/e2e.wasm
var wasmE2EFile embed.FS

const timeOutWebserverRead = 12 * time.Hour
const timeOutWebserverWrite = 12 * time.Hour

// Variable containing all parsed templates
var templateFolder *template.Template

var imageExpiredPicture []byte

var srv http.Server

// Start the webserver on the port set in the config
func Start() {
	initTemplates(templateFolderEmbedded)
	webserverDir, _ := fs.Sub(staticFolderEmbedded, "web/static")
	var err error

	mux := http.NewServeMux()

	if helper.FolderExists("static") {
		fmt.Println("Found folder 'static', using local folder instead of internal static folder")
		mux.Handle("/", http.FileServer(http.Dir("static")))
	} else {
		mux.Handle("/", http.FileServer(http.FS(webserverDir)))
	}
	loadExpiryImage()

	mux.HandleFunc("/admin", requireLogin(showAdminMenu, true, false))
	mux.HandleFunc("/api/", processApi)
	mux.HandleFunc("/apiKeys", requireLogin(showApiAdmin, true, false))
	mux.HandleFunc("/changePassword", requireLogin(changePassword, true, true))
	mux.HandleFunc("/d", showDownload)
	mux.HandleFunc("/downloadFile", downloadFile)
	mux.HandleFunc("/e2eInfo", requireLogin(e2eInfo, false, false))
	mux.HandleFunc("/e2eSetup", requireLogin(showE2ESetup, true, false))
	mux.HandleFunc("/error", showError)
	mux.HandleFunc("/error-auth", showErrorAuth)
	mux.HandleFunc("/error-header", showErrorHeader)
	mux.HandleFunc("/error-oauth", showErrorIntOAuth)
	mux.HandleFunc("/forgotpw", forgotPassword)
	mux.HandleFunc("/hotlink/", showHotlink)
	mux.HandleFunc("/index", showIndex)
	mux.HandleFunc("/login", showLogin)
	mux.HandleFunc("/logs", requireLogin(showLogs, true, false))
	mux.HandleFunc("/logout", doLogout)
	mux.HandleFunc("/uploadChunk", requireLogin(uploadChunk, false, false))
	mux.HandleFunc("/uploadStatus", requireLogin(sse.GetStatusSSE, false, false))
	mux.HandleFunc("/users", requireLogin(showUserAdmin, true, false))
	mux.Handle("/main.wasm", gziphandler.GzipHandler(http.HandlerFunc(serveDownloadWasm)))
	mux.Handle("/e2e.wasm", gziphandler.GzipHandler(http.HandlerFunc(serveE2EWasm)))

	mux.HandleFunc("/d/{id}/{filename}", redirectFromFilename)
	mux.HandleFunc("/dh/{id}/{filename}", downloadFileWithNameInUrl)

	if configuration.Get().Authentication.Method == models.AuthenticationOAuth2 {
		oauth.Init(configuration.Get().ServerUrl, configuration.Get().Authentication)
		mux.HandleFunc("/oauth-login", oauth.HandlerLogin)
		mux.HandleFunc("/oauth-callback", oauth.HandlerCallback)
	}

	fmt.Println("Binding webserver to " + configuration.Get().Port)
	srv = http.Server{
		Addr:         configuration.Get().Port,
		ReadTimeout:  timeOutWebserverRead,
		WriteTimeout: timeOutWebserverWrite,
		Handler:      mux,
	}
	infoMessage := "Webserver can be accessed at " + configuration.Get().ServerUrl + "admin\nPress CTRL+C to stop Gokapi"
	if strings.Contains(configuration.Get().ServerUrl, "127.0.0.1") {
		if configuration.Get().UseSsl {
			infoMessage = strings.Replace(infoMessage, "http://", "https://", 1)
		} else {
			infoMessage = strings.Replace(infoMessage, "https://", "http://", 1)
		}
	}
	if configuration.Get().UseSsl {
		ssl.GenerateIfInvalidCert(configuration.Get().ServerUrl, false)
		fmt.Println(infoMessage)
		err = srv.ListenAndServeTLS(ssl.GetCertificateLocations())
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	} else {
		fmt.Println(infoMessage)
		err = srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}
}

func loadExpiryImage() {
	svgTemplate, err := templatetext.ParseFS(templateFolderEmbedded, "web/templates/expired_file_svg.tmpl")
	helper.Check(err)
	var buf bytes.Buffer
	err = svgTemplate.Execute(&buf, struct {
		PublicName string
	}{PublicName: configuration.Get().PublicName})
	helper.Check(err)
	imageExpiredPicture = buf.Bytes()
}

// Shutdown closes the webserver gracefully
func Shutdown() {
	sse.Shutdown()
	err := srv.Shutdown(context.Background())
	if err != nil {
		log.Println(err)
	}
}

// Initialises the templateFolder variable by scanning through all the templates.
// If a folder "templates" exists in the main directory, it is used.
// Otherwise, templateFolderEmbedded will be used.
func initTemplates(templateFolderEmbedded embed.FS) {
	var err error

	funcMap := template.FuncMap{
		"newAdminButtonContext": newAdminButtonContext,
	}
	if helper.FolderExists("templates") {
		fmt.Println("Found folder 'templates', using local folder instead of internal template folder")
		templateFolder, err = template.New("").Funcs(funcMap).ParseGlob("templates/*.tmpl")
		helper.Check(err)
	} else {
		templateFolder, err = template.New("").Funcs(funcMap).ParseFS(templateFolderEmbedded, "web/templates/*.tmpl")
		helper.Check(err)
	}
}

// Sends a redirect HTTP output to the client. Variable url is used to redirect to ./url
func redirect(w http.ResponseWriter, url string) {
	_, _ = io.WriteString(w, "<html><head><meta http-equiv=\"Refresh\" content=\"0; URL=./"+url+"\"></head></html>")
}

type redirectValues struct {
	FileId           string
	RedirectUrl      string
	Name             string
	Size             string
	PublicName       string
	BaseUrl          string
	PasswordRequired bool
}

// Handling of /id/?/? - used when filename shall be displayed, will redirect to regular download URL
func redirectFromFilename(w http.ResponseWriter, r *http.Request) {
	addNoCacheHeader(w)
	id := r.PathValue("id")
	file, ok := storage.GetFile(id)
	if !ok {
		redirect(w, "../../error")
		return
	}

	config := configuration.Get()
	err := templateFolder.ExecuteTemplate(w, "redirect_filename", redirectValues{
		FileId:           id,
		RedirectUrl:      "d",
		Name:             file.Name,
		Size:             file.Size,
		PublicName:       config.PublicName,
		BaseUrl:          config.ServerUrl,
		PasswordRequired: file.PasswordHash != ""})
	helper.CheckIgnoreTimeout(err)
}

// Handling of /main.wasm
func serveDownloadWasm(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Cache-Control", "public, max-age=100800") // 2 days
	w.Header().Set("content-type", "application/wasm")
	file, err := wasmDownloadFile.ReadFile("web/main.wasm")
	helper.Check(err)
	_, _ = w.Write(file)
}

// Handling of /e2e.wasm
func serveE2EWasm(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Cache-Control", "public, max-age=100800") // 2 days
	w.Header().Set("content-type", "application/wasm")
	file, err := wasmE2EFile.ReadFile("web/e2e.wasm")
	helper.Check(err)
	_, _ = w.Write(file)
}

// Handling of /logout
func doLogout(w http.ResponseWriter, r *http.Request) {
	authentication.Logout(w, r)
}

// Handling of /index and redirecting to globalConfig.RedirectUrl
func showIndex(w http.ResponseWriter, r *http.Request) {
	err := templateFolder.ExecuteTemplate(w, "index", genericView{RedirectUrl: configuration.Get().RedirectUrl, PublicName: configuration.Get().PublicName})
	helper.CheckIgnoreTimeout(err)
}

// Handling of /changePassword
func changePassword(w http.ResponseWriter, r *http.Request) {
	var errMessage string
	user, err := authentication.GetUserFromRequest(r)
	if err != nil {
		panic(err)
	}
	if !user.ResetPassword {
		redirect(w, "admin")
		return
	}
	err = r.ParseForm()
	if err != nil {
		fmt.Println("Invalid form data sent to server for /changePassword")
		fmt.Println(err)
		errMessage = "Invalid form data sent"
	} else {
		var ok bool
		var pwHash string

		pw := r.Form.Get("newpw")
		errMessage, pwHash, ok = validateNewPassword(pw, user)
		if ok {
			user.Password = pwHash
			user.ResetPassword = false
			database.SaveUser(user, false)
			redirect(w, "admin")
			return
		}
	}
	err = templateFolder.ExecuteTemplate(w, "changepw",
		genericView{PublicName: configuration.Get().PublicName,
			MinPasswordLength: configuration.MinLengthPassword,
			ErrorMessage:      errMessage})
	helper.CheckIgnoreTimeout(err)
}

func validateNewPassword(newPassword string, user models.User) (string, string, bool) {
	if len(newPassword) == 0 {
		return "", user.Password, false
	}
	if len(newPassword) < configuration.MinLengthPassword {
		return "Password is too short", user.Password, false
	}
	newPasswordHash := configuration.HashPassword(newPassword, false)
	if user.Password == newPasswordHash {
		return "New password has to be different from the old password", user.Password, false
	}
	return "", newPasswordHash, true
}

// Handling of /error
func showError(w http.ResponseWriter, r *http.Request) {
	const invalidFile = 0
	const noCipherSupplied = 1
	const wrongCipher = 2

	errorReason := invalidFile
	if r.URL.Query().Has("e2e") {
		errorReason = noCipherSupplied
	}
	if r.URL.Query().Has("key") {
		errorReason = wrongCipher
	}
	err := templateFolder.ExecuteTemplate(w, "error", genericView{ErrorId: errorReason, PublicName: configuration.Get().PublicName})
	helper.CheckIgnoreTimeout(err)
}

// Handling of /error-auth
func showErrorAuth(w http.ResponseWriter, r *http.Request) {
	err := templateFolder.ExecuteTemplate(w, "error_auth", genericView{PublicName: configuration.Get().PublicName})
	helper.CheckIgnoreTimeout(err)
}

// Handling of /error-header
func showErrorHeader(w http.ResponseWriter, r *http.Request) {
	err := templateFolder.ExecuteTemplate(w, "error_auth_header", genericView{PublicName: configuration.Get().PublicName})
	helper.CheckIgnoreTimeout(err)
}

// Handling of /error-oauth
func showErrorIntOAuth(w http.ResponseWriter, r *http.Request) {
	view := oauthErrorView{PublicName: configuration.Get().PublicName}
	view.IsAuthDenied = r.URL.Query().Get("isDenied") == "true"
	view.ErrorProvidedName = r.URL.Query().Get("error")
	view.ErrorProvidedMessage = r.URL.Query().Get("error_description")
	view.ErrorGenericMessage = r.URL.Query().Get("error_generic")
	err := templateFolder.ExecuteTemplate(w, "error_int_oauth", view)
	helper.CheckIgnoreTimeout(err)
}

// Handling of /forgotpw
func forgotPassword(w http.ResponseWriter, r *http.Request) {
	err := templateFolder.ExecuteTemplate(w, "forgotpw", genericView{PublicName: configuration.Get().PublicName})
	helper.CheckIgnoreTimeout(err)
}

// Handling of /api
// If user is authenticated, this menu lists all uploads and enables uploading new files
func showApiAdmin(w http.ResponseWriter, r *http.Request) {
	userId, err := authentication.GetUserFromRequest(r)
	if err != nil {
		panic(err)
	}
	view := (&UploadView{}).convertGlobalConfig(ViewAPI, userId)
	err = templateFolder.ExecuteTemplate(w, "api", view)
	helper.CheckIgnoreTimeout(err)
}

// Handling of /users
// If user is authenticated, this menu lists all users
func showUserAdmin(w http.ResponseWriter, r *http.Request) {
	userId, err := authentication.GetUserFromRequest(r)
	if err != nil {
		panic(err)
	}
	view := (&UploadView{}).convertGlobalConfig(ViewUsers, userId)
	if !view.ActiveUser.HasPermissionManageUsers() || configuration.Get().Authentication.Method == models.AuthenticationDisabled {
		redirect(w, "admin")
		return
	}
	err = templateFolder.ExecuteTemplate(w, "users", view)
	helper.CheckIgnoreTimeout(err)
}

// Handling of /api/
func processApi(w http.ResponseWriter, r *http.Request) {
	api.Process(w, r)
}

// Handling of /login
// Shows a login form. If not authenticated, client needs to wait for three seconds.
// If correct, a new session is created and the user is redirected to the admin menu
func showLogin(w http.ResponseWriter, r *http.Request) {
	_, ok := authentication.IsAuthenticated(w, r)
	if ok {
		redirect(w, "admin")
		return
	}
	if configuration.Get().Authentication.Method == models.AuthenticationHeader {
		redirect(w, "error-header")
		return
	}
	if configuration.Get().Authentication.Method == models.AuthenticationOAuth2 {
		// If user clicked logout, force consent
		if r.URL.Query().Has("consent") {
			redirect(w, "oauth-login?consent=true")
		} else {
			redirect(w, "oauth-login")
		}
		return
	}
	err := r.ParseForm()
	if err != nil {
		fmt.Println("Invalid form data sent to server for /login")
		fmt.Println(err)
		return
	}
	user := r.Form.Get("username")
	pw := r.Form.Get("password")
	failedLogin := false
	if pw != "" && user != "" {
		retrievedUser, validCredentials := authentication.IsCorrectUsernameAndPassword(user, pw)
		if validCredentials {
			sessionmanager.CreateSession(w, false, 0, retrievedUser.Id)
			redirect(w, "admin")
			return
		}
		select {
		case <-time.After(3 * time.Second):
		}
		failedLogin = true
	}
	err = templateFolder.ExecuteTemplate(w, "login", LoginView{
		IsFailedLogin: failedLogin,
		User:          user,
		IsAdminView:   false,
		PublicName:    configuration.Get().PublicName,
	})
	helper.CheckIgnoreTimeout(err)
}

// LoginView contains variables for the login template
type LoginView struct {
	IsFailedLogin  bool
	IsAdminView    bool
	IsDownloadView bool
	User           string
	PublicName     string
}

// Handling of /d
// Checks if a file exists for the submitted ID
// If it exists, a download form is shown or a password needs to be entered.
func showDownload(w http.ResponseWriter, r *http.Request) {
	addNoCacheHeader(w)
	keyId := queryUrl(w, r, "error")
	file, ok := storage.GetFile(keyId)
	if !ok {
		redirect(w, "error")
		return
	}

	config := configuration.Get()

	view := DownloadView{
		Name:               file.Name,
		Size:               file.Size,
		Id:                 file.Id,
		IsDownloadView:     true,
		EndToEndEncryption: file.Encryption.IsEndToEndEncrypted,
		PublicName:         config.PublicName,
		BaseUrl:            config.ServerUrl,
		IsFailedLogin:      false,
		UsesHttps:          configuration.UsesHttps(),
	}

	if file.RequiresClientDecryption() {
		view.ClientSideDecryption = true
		if !file.Encryption.IsEndToEndEncrypted {
			cipher, err := encryption.GetCipherFromFile(file.Encryption)
			helper.Check(err)
			view.Cipher = base64.StdEncoding.EncodeToString(cipher)
		}
	}

	if file.PasswordHash != "" {
		_ = r.ParseForm()
		enteredPassword := r.Form.Get("password")
		if configuration.HashPassword(enteredPassword, true) != file.PasswordHash && !isValidPwCookie(r, file) {
			if enteredPassword != "" {
				view.IsFailedLogin = true
				select {
				case <-time.After(1 * time.Second):
				}
			}
			view.IsPasswordView = true
			err := templateFolder.ExecuteTemplate(w, "download_password", view)
			helper.CheckIgnoreTimeout(err)
			return
		}
		if !isValidPwCookie(r, file) {
			writeFilePwCookie(w, file)
			// redirect so that there is no post data to be resent if user refreshes page
			redirect(w, "d?id="+file.Id)
			return
		}
	}
	err := templateFolder.ExecuteTemplate(w, "download", view)
	helper.CheckIgnoreTimeout(err)
}

// Handling of /hotlink/
// Hotlinks an image or returns a static error image if image has expired
func showHotlink(w http.ResponseWriter, r *http.Request) {
	hotlinkId := strings.Replace(r.URL.Path, "/hotlink/", "", 1)
	addNoCacheHeader(w)
	file, ok := storage.GetFileByHotlink(hotlinkId)
	if !ok {
		w.Header().Set("Content-Type", "image/svg+xml")
		_, _ = w.Write(imageExpiredPicture)
		return
	}
	storage.ServeFile(file, w, r, false)
}

// Handling of /e2eInfo
// User needs to be admin. Receives or stores end2end encryption info
func e2eInfo(w http.ResponseWriter, r *http.Request) {
	action, ok := r.URL.Query()["action"]
	if !ok || len(action) < 1 {
		responseError(w, errors.New("invalid action specified"))
		return
	}

	user, err := authentication.GetUserFromRequest(r)
	if err != nil {
		responseError(w, err)
		return
	}
	switch action[0] {
	case "get":
		getE2eInfo(w, user.Id)
	case "store":
		storeE2eInfo(w, r, user.Id)
	default:
		responseError(w, errors.New("invalid action specified"))
	}
}

func storeE2eInfo(w http.ResponseWriter, r *http.Request, userId int) {
	err := r.ParseForm()
	if err != nil {
		responseError(w, err)
		return
	}
	uploadedInfoBase64 := r.Form.Get("info")
	if uploadedInfoBase64 == "" {
		responseError(w, errors.New("empty info sent"))
		return
	}
	uploadedInfo, err := base64.StdEncoding.DecodeString(uploadedInfoBase64)
	if err != nil {
		responseError(w, err)
		return
	}
	var info models.E2EInfoEncrypted
	err = json.Unmarshal(uploadedInfo, &info)
	if err != nil {
		responseError(w, err)
		return
	}
	database.SaveEnd2EndInfo(info, userId)
	_, _ = w.Write([]byte("\"result\":\"OK\""))
}

func getE2eInfo(w http.ResponseWriter, userId int) {
	info := database.GetEnd2EndInfo(userId)
	bytesE2e, err := json.Marshal(info)
	helper.Check(err)
	_, _ = w.Write(bytesE2e)
}

// Checks if a file is associated with the GET parameter from the current URL
// Stops for 500ms to limit brute forcing if invalid key and redirects to redirectUrl
func queryUrl(w http.ResponseWriter, r *http.Request, redirectUrl string) string {
	keys, ok := r.URL.Query()["id"]
	if !ok || len(keys[0]) < configuration.Get().LengthId {
		select {
		case <-time.After(500 * time.Millisecond):
		}
		redirect(w, redirectUrl)
		return ""
	}
	return keys[0]
}

// Handling of /admin
// If user is authenticated, this menu lists all uploads and enables uploading new files
func showAdminMenu(w http.ResponseWriter, r *http.Request) {

	user, err := authentication.GetUserFromRequest(r)
	if err != nil {
		panic(err)
	}

	if configuration.Get().Encryption.Level == encryption.EndToEndEncryption {
		e2einfo := database.GetEnd2EndInfo(user.Id)
		if !e2einfo.HasBeenSetUp() {
			redirect(w, "e2eSetup")
			return
		}
	}
	err = templateFolder.ExecuteTemplate(w, "admin", (&UploadView{}).convertGlobalConfig(ViewMain, user))
	helper.CheckIgnoreTimeout(err)
}

// Handling of /logs
// If user is authenticated, this menu shows the stored logs
func showLogs(w http.ResponseWriter, r *http.Request) {
	user, err := authentication.GetUserFromRequest(r)
	if err != nil {
		panic(err)
	}
	view := (&UploadView{}).convertGlobalConfig(ViewLogs, user)
	if !view.ActiveUser.HasPermissionManageLogs() {
		redirect(w, "admin")
		return
	}
	err = templateFolder.ExecuteTemplate(w, "logs", view)
	helper.CheckIgnoreTimeout(err)
}

func showE2ESetup(w http.ResponseWriter, r *http.Request) {
	if configuration.Get().Encryption.Level != encryption.EndToEndEncryption {
		redirect(w, "admin")
		return
	}

	user, err := authentication.GetUserFromRequest(r)
	if err != nil {
		panic(err)
	}
	e2einfo := database.GetEnd2EndInfo(user.Id)
	err = templateFolder.ExecuteTemplate(w, "e2esetup", e2ESetupView{HasBeenSetup: e2einfo.HasBeenSetUp(), PublicName: configuration.Get().PublicName})
	helper.CheckIgnoreTimeout(err)
}

// DownloadView contains parameters for the download template
type DownloadView struct {
	Name                 string
	Size                 string
	Id                   string
	Cipher               string
	PublicName           string
	BaseUrl              string
	IsFailedLogin        bool
	IsAdminView          bool
	IsDownloadView       bool
	IsPasswordView       bool
	ClientSideDecryption bool
	EndToEndEncryption   bool
	UsesHttps            bool
}

type e2ESetupView struct {
	IsAdminView    bool
	IsDownloadView bool
	HasBeenSetup   bool
	PublicName     string
}

// UploadView contains parameters for the admin menu template
type UploadView struct {
	Items              []models.FileApiOutput
	ApiKeys            []models.ApiKey
	Users              []userInfo
	ActiveUser         models.User
	UserMap            map[int]*models.User
	ServerUrl          string
	Logs               string
	PublicName         string
	SystemKey          string
	IsAdminView        bool
	IsDownloadView     bool
	IsApiView          bool
	IsLogoutAvailable  bool
	IsUserTabAvailable bool
	EndToEndEncryption bool
	IncludeFilename    bool
	IsInternalAuth     bool
	MaxFileSize        int
	ActiveView         int
	ChunkSize          int
	MaxParallelUploads int
	TimeNow            int64
}

// getUserMap needs to return the map with pointers, otherwise template cannot call
// functions associated with it
func getUserMap() map[int]*models.User {
	result := make(map[int]*models.User)
	users := database.GetAllUsers()
	for _, user := range users {
		result[user.Id] = &user
	}
	return result
}

const (
	// ViewMain is the identifier for the main menu
	ViewMain = iota
	// ViewLogs is the identifier for the log viewer menu
	ViewLogs
	// ViewAPI is the identifier for the API menu
	ViewAPI
	// ViewUsers is the identifier for the user management menu
	ViewUsers
)

// Converts the globalConfig variable to an UploadView struct to pass the infos to
// the admin template
func (u *UploadView) convertGlobalConfig(view int, user models.User) *UploadView {
	var result []models.FileApiOutput
	var resultApi []models.ApiKey

	config := configuration.Get()
	u.IsInternalAuth = config.Authentication.Method == models.AuthenticationInternal
	u.ActiveUser = user
	u.UserMap = getUserMap()
	switch view {
	case ViewMain:
		for _, element := range database.GetAllMetadata() {
			if element.UserId != user.Id && !user.HasPermissionListOtherUploads() {
				continue
			}
			fileInfo, err := element.ToFileApiOutput(config.ServerUrl, config.IncludeFilename)
			helper.Check(err)
			result = append(result, fileInfo)
		}
		sort.Slice(result[:], func(i, j int) bool {
			if result[i].ExpireAt == result[j].ExpireAt {
				return result[i].Id > result[j].Id
			}
			return result[i].ExpireAt > result[j].ExpireAt
		})
	case ViewAPI:
		for _, apiKey := range database.GetAllApiKeys() {
			// Double-checking if user of API key exists
			// If the user was manually deleted from the database, this could lead to a crash
			// in the API view
			_, ok := u.UserMap[apiKey.UserId]
			if !ok {
				continue
			}
			if !apiKey.IsSystemKey {
				if apiKey.UserId == user.Id || user.HasPermissionManageApi() {
					resultApi = append(resultApi, apiKey)
				}
			}
		}
		sort.Slice(resultApi[:], func(i, j int) bool {
			if resultApi[i].LastUsed == resultApi[j].LastUsed {
				return resultApi[i].Id < resultApi[j].Id
			}
			return resultApi[i].LastUsed > resultApi[j].LastUsed
		})
	case ViewLogs:
		u.Logs, _ = logging.GetAll()
	case ViewUsers:
		uploadCounts := storage.GetUploadCounts()
		u.Users = make([]userInfo, 0)
		for _, userEntry := range database.GetAllUsers() {
			userWithUploads := userInfo{
				UploadCount: uploadCounts[userEntry.Id],
				User:        userEntry,
			}
			// Otherwise the user is not shown as online, if /users is opened as first page
			if userEntry.Id == user.Id {
				userWithUploads.User.LastOnline = time.Now().Unix()
			}
			u.Users = append(u.Users, userWithUploads)
		}
	}

	u.ServerUrl = config.ServerUrl
	u.Items = result
	u.PublicName = config.PublicName
	u.ApiKeys = resultApi
	u.TimeNow = time.Now().Unix()
	u.IsAdminView = true
	u.ActiveView = view
	u.MaxFileSize = config.MaxFileSizeMB
	u.IsLogoutAvailable = authentication.IsLogoutAvailable()
	u.IsUserTabAvailable = config.Authentication.Method != models.AuthenticationDisabled
	u.EndToEndEncryption = config.Encryption.Level == encryption.EndToEndEncryption
	u.MaxParallelUploads = config.MaxParallelUploads
	u.ChunkSize = config.ChunkSize
	u.IncludeFilename = config.IncludeFilename
	u.SystemKey = api.GetSystemKey(user.Id)
	return u
}

type userInfo struct {
	UploadCount int
	User        models.User
}

// Handling of /uploadChunk
// If the user is authenticated, this parses the uploaded chunk and stores it
func uploadChunk(w http.ResponseWriter, r *http.Request) {
	maxUpload := int64(configuration.Get().MaxFileSizeMB) * 1024 * 1024
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if r.ContentLength > maxUpload {
		responseError(w, storage.ErrorFileTooLarge)
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
	err := fileupload.ProcessNewChunk(w, r, false)
	responseError(w, err)
}

// Outputs an error in json format if err!=nil
func responseError(w http.ResponseWriter, err error) {
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, "{\"Result\":\"error\",\"ErrorMessage\":\""+err.Error()+"\"}")
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			log.Println(err)
		}
	}
}

// Handling of /dh/?/?
// Hotlinks a file and has the filename in the URL
func downloadFileWithNameInUrl(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	serveFile(id, false, w, r)
}

// Handling of /downloadFile
// Outputs the file to the user and reduces the download remaining count for the file
func downloadFile(w http.ResponseWriter, r *http.Request) {
	id := queryUrl(w, r, "error")
	serveFile(id, true, w, r)
}

func serveFile(id string, isRootUrl bool, w http.ResponseWriter, r *http.Request) {
	addNoCacheHeader(w)
	savedFile, ok := storage.GetFile(id)
	if !ok {
		if isRootUrl {
			redirect(w, "error")
		} else {
			redirect(w, "../../error")
		}
		return
	}
	if savedFile.PasswordHash != "" {
		if !(isValidPwCookie(r, savedFile)) {
			if isRootUrl {
				redirect(w, "d?id="+savedFile.Id)
			} else {
				redirect(w, "../../d?id="+savedFile.Id)
			}
			return
		}
	}
	storage.ServeFile(savedFile, w, r, true)
}

func requireLogin(next http.HandlerFunc, isUiCall, isPwChangeView bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		addNoCacheHeader(w)
		user, isLoggedIn := authentication.IsAuthenticated(w, r)
		if isLoggedIn {
			if user.ResetPassword && isUiCall && configuration.Get().Authentication.Method == models.AuthenticationInternal {
				if !isPwChangeView {
					redirect(w, "changePassword")
					return
				}
			}
			c := context.WithValue(r.Context(), "user", user)
			r = r.WithContext(c)
			next.ServeHTTP(w, r)
			return
		}
		if !isUiCall {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, "{\"Result\":\"error\",\"ErrorMessage\":\"Not authenticated\"}")
			return
		}
		redirect(w, "login")
	}
}

type adminButtonContext struct {
	CurrentFile models.FileApiOutput
	ActiveUser  *models.User
}

// Used internally in templates, to create buttons with user context
func newAdminButtonContext(file models.FileApiOutput, user models.User) adminButtonContext {
	return adminButtonContext{CurrentFile: file, ActiveUser: &user}
}

// Write a cookie if the user has entered a correct password for a password-protected file
func writeFilePwCookie(w http.ResponseWriter, file models.File) {
	http.SetCookie(w, &http.Cookie{
		Name:    "p" + file.Id,
		Value:   file.PasswordHash,
		Expires: time.Now().Add(5 * time.Minute),
	})
}

// Checks if a cookie contains the correct password hash for a password-protected file
// If incorrect, a 3-second delay is introduced unless the cookie was empty.
func isValidPwCookie(r *http.Request, file models.File) bool {
	cookie, err := r.Cookie("p" + file.Id)
	if err == nil {
		if cookie.Value == file.PasswordHash {
			return true
		}
		select {
		case <-time.After(3 * time.Second):
		}
	}
	return false
}

// Adds a header to disable external caching
func addNoCacheHeader(w http.ResponseWriter) {
	w.Header().Set("cdn-cache-control", "no-store, no-cache")
	w.Header().Set("Cloudflare-CDN-Cache-Control", "no-store, no-cache")
	w.Header().Set("cache-control", "no-store, no-cache")
	w.Header().Set("Pragma", "no-cache")
}

// A view containing parameters for a generic template
type genericView struct {
	IsAdminView       bool
	IsDownloadView    bool
	PublicName        string
	RedirectUrl       string
	ErrorMessage      string
	ErrorId           int
	MinPasswordLength int
}

// A view containing parameters for an oauth error
type oauthErrorView struct {
	IsAdminView          bool
	IsDownloadView       bool
	PublicName           string
	IsAuthDenied         bool
	ErrorGenericMessage  string
	ErrorProvidedName    string
	ErrorProvidedMessage string
}
