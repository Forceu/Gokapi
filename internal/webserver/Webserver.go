package webserver

/**
Handling of webserver and requests / uploads
*/

import (
	"bytes"
	"context"
	"crypto/subtle"
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	templatetext "text/template"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/encryption"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/logging"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage"
	"github.com/forceu/gokapi/internal/storage/filerequest"
	"github.com/forceu/gokapi/internal/webserver/api"
	"github.com/forceu/gokapi/internal/webserver/authentication"
	"github.com/forceu/gokapi/internal/webserver/authentication/oauth"
	"github.com/forceu/gokapi/internal/webserver/authentication/sessionmanager"
	"github.com/forceu/gokapi/internal/webserver/authentication/tokengeneration"
	"github.com/forceu/gokapi/internal/webserver/favicon"
	"github.com/forceu/gokapi/internal/webserver/fileupload"
	"github.com/forceu/gokapi/internal/webserver/sse"
	"github.com/forceu/gokapi/internal/webserver/ssl"
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

const timeOutWebserverRead = 2 * time.Hour
const timeOutWebserverWrite = 12 * time.Hour

// templateFolder contains all parsed templates
var templateFolder *template.Template

// customStaticInfo is passed to all templates, so custom CSS or JS can be embedded
var customStaticInfo customStatic

// imageExpiredPicture is sent for an expired hotlink
var imageExpiredPicture []byte

// srv is the web server that is used for this module
var srv http.Server

// Start the webserver on the port set in the config
func Start() {
	initTemplates(templateFolderEmbedded)
	webserverDir, _ := fs.Sub(staticFolderEmbedded, "web/static")
	var err error

	mux := http.NewServeMux()
	loadCustomCssJsInfo(webserverDir)
	loadExpiryImage()

	mux.Handle("/", filesystemHandler(webserverDir))
	mux.HandleFunc("/auth/token", requireLogin(handleGenerateAuthToken, false, false))
	mux.HandleFunc("/admin", requireLogin(showAdminMenu, true, false))
	mux.HandleFunc("/api/", processApi)
	mux.HandleFunc("/apiKeys", requireLogin(showApiAdmin, true, false))
	mux.HandleFunc("/changePassword", requireLogin(changePassword, true, true))
	mux.HandleFunc("/d", showDownload)
	mux.HandleFunc("/downloadFile", downloadFile)
	mux.HandleFunc("/downloadPresigned", requireLogin(downloadPresigned, false, false))
	mux.HandleFunc("/e2eSetup", requireLogin(showE2ESetup, true, false))
	mux.HandleFunc("/error", showError)
	mux.HandleFunc("/error-auth", showErrorAuth)
	mux.HandleFunc("/error-header", showErrorHeader)
	mux.HandleFunc("/error-oauth", showErrorIntOAuth)
	mux.HandleFunc("/filerequests", requireLogin(showUploadRequest, true, false))
	mux.HandleFunc("/forgotpw", forgotPassword)
	mux.HandleFunc("/h/", showHotlink)
	mux.HandleFunc("/hotlink/", showHotlink) // backward compatibility
	mux.HandleFunc("/index", showIndex)
	mux.HandleFunc("/login", showLogin)
	mux.HandleFunc("/logs", requireLogin(showLogs, true, false))
	mux.HandleFunc("/logout", doLogout)
	mux.HandleFunc("/publicUpload", showPublicUpload)
	mux.HandleFunc("/uploadChunk", requireLogin(uploadChunk, false, false))
	mux.HandleFunc("/uploadStatus", requireLogin(sse.GetStatusSSE, false, false))
	mux.HandleFunc("/users", requireLogin(showUserAdmin, true, false))
	mux.Handle("/main.wasm", gziphandler.GzipHandler(http.HandlerFunc(serveDownloadWasm)))
	mux.Handle("/e2e.wasm", gziphandler.GzipHandler(http.HandlerFunc(serveE2EWasm)))
	mux.HandleFunc("/d/{id}/{filename}", redirectFromFilename)
	mux.HandleFunc("/dh/{id}/{filename}", downloadFileWithNameInUrl)

	addMuxForCustomContent(mux)

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

func filesystemHandler(webserverDir fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/favicon") {
			handleFavicon(w, r)
			return
		}
		http.FileServer(http.FS(webserverDir)).ServeHTTP(w, r)
	}
}

func handleFavicon(w http.ResponseWriter, r *http.Request) {
	icon := favicon.GetFavicon(r.URL.Path)
	_, _ = w.Write(icon)
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

// Handling of /id/?/? - used when filename shall be displayed, will redirect to the regular download URL
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
	err := templateFolder.ExecuteTemplate(w, "index", genericView{RedirectUrl: configuration.Get().RedirectUrl,
		PublicName:    configuration.Get().PublicName,
		CustomContent: customStaticInfo})
	helper.CheckIgnoreTimeout(err)
}

func handleGenerateAuthToken(w http.ResponseWriter, r *http.Request) {
	user, err := authentication.GetUserFromRequest(r)
	if err != nil {
		panic(err)
	}
	permString := r.Header.Get("permission")
	permission, err := models.ApiPermissionFromString(permString)
	if err != nil {
		http.Error(w, "Invalid permission", http.StatusBadRequest)
		return
	}
	token, expiry, err := tokengeneration.Generate(user, permission)
	if err != nil {
		http.Error(w, "Invalid permission", http.StatusBadRequest)
		return
	}
	_, _ = w.Write([]byte("{\"key\":\"" + token + "\",\"expiry\":" + strconv.FormatInt(expiry, 10) + "}"))
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
	config := configuration.Get()
	err = templateFolder.ExecuteTemplate(w, "changepw",
		genericView{PublicName: config.PublicName,
			MinPasswordLength: configuration.GetEnvironment().MinLengthPassword,
			ErrorMessage:      errMessage,
			CustomContent:     customStaticInfo})
	helper.CheckIgnoreTimeout(err)
}

func validateNewPassword(newPassword string, user models.User) (string, string, bool) {
	if len(newPassword) == 0 {
		return "", user.Password, false
	}
	if len(newPassword) < configuration.GetEnvironment().MinLengthPassword {
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
	const (
		invalidFile = iota
		noCipherSupplied
		wrongCipher
		invalidFileRequest
	)

	errorReason := invalidFile
	cardWidth := 18
	if r.URL.Query().Has("e2e") {
		errorReason = noCipherSupplied
		cardWidth = 25
	}
	if r.URL.Query().Has("key") {
		errorReason = wrongCipher
		cardWidth = 25
	}
	if r.URL.Query().Has("fr") {
		errorReason = invalidFileRequest
		cardWidth = 30
	}
	err := templateFolder.ExecuteTemplate(w, "error", genericView{
		ErrorId:        errorReason,
		ErrorCardWidth: cardWidth,
		PublicName:     configuration.Get().PublicName,
		CustomContent:  customStaticInfo})
	helper.CheckIgnoreTimeout(err)
}

// Handling of /error-auth
func showErrorAuth(w http.ResponseWriter, r *http.Request) {
	err := templateFolder.ExecuteTemplate(w, "error_auth", genericView{
		PublicName:    configuration.Get().PublicName,
		CustomContent: customStaticInfo})
	helper.CheckIgnoreTimeout(err)
}

// Handling of /error-header
func showErrorHeader(w http.ResponseWriter, r *http.Request) {
	err := templateFolder.ExecuteTemplate(w, "error_auth_header", genericView{
		PublicName:    configuration.Get().PublicName,
		CustomContent: customStaticInfo})
	helper.CheckIgnoreTimeout(err)
}

// Handling of /error-oauth
func showErrorIntOAuth(w http.ResponseWriter, r *http.Request) {
	view := oauthErrorView{PublicName: configuration.Get().PublicName,
		CustomContent: customStaticInfo}
	view.IsAuthDenied = r.URL.Query().Get("isDenied") == "true"
	view.ErrorProvidedName = r.URL.Query().Get("error")
	view.ErrorProvidedMessage = r.URL.Query().Get("error_description")
	view.ErrorGenericMessage = r.URL.Query().Get("error_generic")
	err := templateFolder.ExecuteTemplate(w, "error_int_oauth", view)
	helper.CheckIgnoreTimeout(err)
}

// Handling of /forgotpw
func forgotPassword(w http.ResponseWriter, r *http.Request) {
	err := templateFolder.ExecuteTemplate(w, "forgotpw", genericView{
		PublicName:    configuration.Get().PublicName,
		CustomContent: customStaticInfo})
	helper.CheckIgnoreTimeout(err)
}

// Handling of /filerequest
func showUploadRequest(w http.ResponseWriter, r *http.Request) {
	userId, err := authentication.GetUserFromRequest(r)
	if err != nil {
		panic(err)
	}
	view := (&AdminView{}).convertGlobalConfig(ViewFileRequests, userId)
	err = templateFolder.ExecuteTemplate(w, "uploadreq", view)
	helper.CheckIgnoreTimeout(err)
}

// Handling of /api
// If the user is authenticated, this menu lists all uploads and enables uploading new files
func showApiAdmin(w http.ResponseWriter, r *http.Request) {
	userId, err := authentication.GetUserFromRequest(r)
	if err != nil {
		panic(err)
	}
	view := (&AdminView{}).convertGlobalConfig(ViewAPI, userId)
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
	view := (&AdminView{}).convertGlobalConfig(ViewUsers, userId)
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
		CustomContent: customStaticInfo,
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
	CustomContent  customStatic
}

// Handling of /d
// Checks if a file exists for the submitted ID
// If it exists, a download form is shown, or a password needs to be entered.
func showDownload(w http.ResponseWriter, r *http.Request) {
	addNoCacheHeader(w)
	keyId := queryUrl(w, r, "id", "error")
	file, ok := storage.GetFile(keyId)
	if !ok || file.IsFileRequest() {
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
		CustomContent:      customStaticInfo,
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

// Handling of /h/ and /hotlink/
// Hotlinks an image or returns a static error image if image has expired
func showHotlink(w http.ResponseWriter, r *http.Request) {
	hotlinkId := strings.Replace(r.URL.Path, "/hotlink/", "", 1)
	hotlinkId = strings.Replace(hotlinkId, "/h/", "", 1)
	addNoCacheHeader(w)
	file, ok := storage.GetFileByHotlink(hotlinkId)
	if !ok || file.IsFileRequest() {
		w.Header().Set("Content-Type", "image/svg+xml")
		_, _ = w.Write(imageExpiredPicture)
		return
	}
	storage.ServeFile(file, w, r, false, true)
}

// Checks if a file is associated with the GET parameter from the current URL
// Stops for 500ms to limit brute forcing if invalid key and redirects to redirectUrl
func queryUrl(w http.ResponseWriter, r *http.Request, keyword string, redirectUrl string) string {
	keys, ok := r.URL.Query()[keyword]
	if !ok || len(keys[0]) < configuration.GetEnvironment().LengthId {
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

	config := configuration.Get()
	if config.Encryption.Level == encryption.EndToEndEncryption {
		e2einfo := database.GetEnd2EndInfo(user.Id)
		if !e2einfo.HasBeenSetUp() {
			redirect(w, "e2eSetup")
			return
		}
	}

	view := (&AdminView{}).convertGlobalConfig(ViewMain, user)
	if len(configuration.GetEnvironment().ActiveDeprecations) > 0 {
		if user.IsSuperAdmin() {
			view.ShowDeprecationNotice = true
		}
	}

	err = templateFolder.ExecuteTemplate(w, "admin", view)
	helper.CheckIgnoreTimeout(err)
}

// Handling of /logs
// If user is authenticated, this menu shows the stored logs
func showLogs(w http.ResponseWriter, r *http.Request) {
	user, err := authentication.GetUserFromRequest(r)
	if err != nil {
		panic(err)
	}
	view := (&AdminView{}).convertGlobalConfig(ViewLogs, user)
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
	err = templateFolder.ExecuteTemplate(w, "e2esetup", e2ESetupView{
		HasBeenSetup:  e2einfo.HasBeenSetUp(),
		PublicName:    configuration.Get().PublicName,
		CustomContent: customStaticInfo})
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
	CustomContent        customStatic
}

type e2ESetupView struct {
	IsAdminView    bool
	IsDownloadView bool
	HasBeenSetup   bool
	PublicName     string
	CustomContent  customStatic
}

// AdminView contains parameters for all admin-related pages
type AdminView struct {
	Items                 []models.FileApiOutput
	ApiKeys               []models.ApiKey
	Users                 []userInfo
	FileRequests          []models.FileRequest
	ActiveUser            models.User
	UserMap               map[int]*models.User
	ServerUrl             string
	Logs                  string
	PublicName            string
	IsAdminView           bool
	IsDownloadView        bool
	IsApiView             bool
	IsLogoutAvailable     bool
	IsUserTabAvailable    bool
	EndToEndEncryption    bool
	IncludeFilename       bool
	IsInternalAuth        bool
	ShowDeprecationNotice bool
	MaxFileSize           int
	ActiveView            int
	ChunkSize             int
	MaxParallelUploads    int
	MinLengthPassword     int
	FileRequestMaxFiles   int
	FileRequestMaxSize    int
	TimeNow               int64
	CustomContent         customStatic
}

// getUserMap needs to return the map with pointers; otherwise template cannot call
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
	// ViewFileRequests is the identifier for the file request menu
	ViewFileRequests
)

// Converts the globalConfig variable to an AdminView struct to pass the infos to
// the admin template
func (u *AdminView) convertGlobalConfig(view int, user models.User) *AdminView {
	var metaDataList []models.FileApiOutput
	var apiKeyList []models.ApiKey

	config := configuration.Get()
	u.IsInternalAuth = config.Authentication.Method == models.AuthenticationInternal
	u.ActiveUser = user
	u.UserMap = getUserMap()
	u.CustomContent = customStaticInfo
	switch view {
	case ViewMain:
		for _, element := range database.GetAllMetadata() {
			if element.UserId != user.Id && !user.HasPermissionListOtherUploads() {
				continue
			}
			fileInfo, err := element.ToFileApiOutput(config.ServerUrl, config.IncludeFilename)
			helper.Check(err)
			metaDataList = append(metaDataList, fileInfo)
		}
		metaDataList = sortMetaDataApi(metaDataList)
	case ViewAPI:
		for _, apiKey := range database.GetAllApiKeys() {
			// Double-checking if the owner of the API key exists
			// If the user was manually deleted from the database, this could lead to a crash
			// in the API view
			_, ok := u.UserMap[apiKey.UserId]
			if !ok {
				continue
			}
			if !apiKey.IsSystemKey && !apiKey.IsUploadRequestKey() {
				if apiKey.UserId == user.Id || user.HasPermissionManageApi() {
					apiKeyList = append(apiKeyList, apiKey)
				}
			}
		}
		apiKeyList = sortApiKeys(apiKeyList)
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
	case ViewFileRequests:
		for _, fileRequest := range filerequest.GetAll() {
			// Double-checking if the owner of the file request exists
			// If the user was manually deleted from the database, this could lead to a crash
			// in the file request view
			_, ok := u.UserMap[fileRequest.UserId]
			if !ok {
				continue
			}
			if fileRequest.UserId != user.Id && !user.HasPermissionListOtherUploads() {
				continue
			}
			fileRequest.Files = sortMetaData(fileRequest.Files)
			u.FileRequests = append(u.FileRequests, fileRequest)
			if !user.IsAdmin() {
				u.FileRequestMaxFiles = configuration.GetEnvironment().MaxFilesGuestUpload
				u.FileRequestMaxSize = configuration.GetEnvironment().MaxSizeGuestUploadMb
			}
		}
	}

	u.ServerUrl = config.ServerUrl
	u.Items = metaDataList
	u.PublicName = config.PublicName
	u.ApiKeys = apiKeyList
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
	return u
}

// sortMetaDataApi arranges the provided array so that Fies are sorted by most recent upload first and if that is equal
// then by most time remaining first. If that is equal, then sort by ID.
func sortMetaDataApi(input []models.FileApiOutput) []models.FileApiOutput {
	sort.Slice(input[:], func(i, j int) bool {
		if input[i].UploadDate != input[j].UploadDate {
			return input[i].UploadDate > input[j].UploadDate
		}
		if input[i].ExpireAt != input[j].ExpireAt {
			return input[i].ExpireAt > input[j].ExpireAt
		}
		return input[i].Id > input[j].Id
	})
	return input
}

// sortMetaData arranges the provided array so that Fies are sorted by most recent upload first then sort by ID.
// Currently only used for the files of File Requests, all others use sortMetaDataApi
func sortMetaData(input []models.File) []models.File {
	sort.Slice(input[:], func(i, j int) bool {
		if input[i].UploadDate != input[j].UploadDate {
			return input[i].UploadDate > input[j].UploadDate
		}
		return input[i].Id > input[j].Id
	})
	return input
}

// sortApiKeys arranges the provided array so that API keys are sorted by most recent usage first and if that is equal
// then by ID
func sortApiKeys(input []models.ApiKey) []models.ApiKey {
	sort.Slice(input[:], func(i, j int) bool {
		if input[i].LastUsed != input[j].LastUsed {
			return input[i].LastUsed > input[j].LastUsed
		}
		return input[i].Id < input[j].Id
	})
	return input
}

type userInfo struct {
	UploadCount int
	User        models.User
}

// Handling of /publicUpload
func showPublicUpload(w http.ResponseWriter, r *http.Request) {
	addNoCacheHeader(w)
	fileRequestId := queryUrl(w, r, "id", "error?fr")
	request, ok := filerequest.Get(fileRequestId)
	if !ok {
		redirect(w, "error?fr")
		return
	}
	if !request.IsUnlimitedTime() && request.Expiry < time.Now().Unix() {
		redirect(w, "error?fr")
		return
	}
	if !request.IsUnlimitedFiles() && request.UploadedFiles >= request.MaxFiles {
		redirect(w, "error?fr")
		return
	}
	apiKey := queryUrl(w, r, "key", "error?fr")
	if subtle.ConstantTimeCompare([]byte(request.ApiKey), []byte(apiKey)) != 1 {
		redirect(w, "error?fr")
		return
	}

	config := configuration.Get()

	view := publicUploadView{
		PublicName:    config.PublicName,
		ChunkSize:     config.ChunkSize,
		MaxServerSize: config.MaxFileSizeMB,
		FileRequest:   &request,
		CustomContent: customStaticInfo,
	}

	err := templateFolder.ExecuteTemplate(w, "publicUpload", view)
	helper.CheckIgnoreTimeout(err)
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
	err := fileupload.ProcessNewChunk(w, r, false, "")
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
	id := queryUrl(w, r, "id", "error")
	serveFile(id, true, w, r)
}

// Handling of /downloadPresigned
// Outputs the file to the user and reduces the download remaining count for the file, if requested
func downloadPresigned(w http.ResponseWriter, r *http.Request) {
	presignKey, ok := r.URL.Query()["key"]
	if !ok {
		responseError(w, storage.ErrorInvalidPresign)
		return
	}
	presign, ok := database.GetPresignedUrl(presignKey[0])
	if !ok || presign.Expiry < time.Now().Unix() {
		responseError(w, storage.ErrorInvalidPresign)
		return
	}
	files := make([]models.File, 0)
	for _, file := range presign.FileIds {
		storedFile, ok := storage.GetFile(file)
		if !ok {
			responseError(w, storage.ErrorFileNotFound)
			return
		}
		files = append(files, storedFile)
	}
	database.DeletePresignedUrl(presign.Id)

	if len(files) == 1 {
		storage.ServeFile(files[0], w, r, true, false)
		return
	}
	storage.ServeFilesAsZip(files, presign.Filename, w, r)
}

func serveFile(id string, isRootUrl bool, w http.ResponseWriter, r *http.Request) {
	addNoCacheHeader(w)
	savedFile, ok := storage.GetFile(id)
	if !ok || savedFile.IsFileRequest() {
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
	storage.ServeFile(savedFile, w, r, true, true)
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
			r = authentication.SetUserInRequest(r, user)
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
	ErrorCardWidth    int
	MinPasswordLength int
	CustomContent     customStatic
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
	CustomContent        customStatic
}

// A view containing parameters for the public upload page
type publicUploadView struct {
	IsAdminView    bool
	IsDownloadView bool
	PublicName     string
	ChunkSize      int
	MaxServerSize  int
	CustomContent  customStatic
	FileRequest    *models.FileRequest
}
