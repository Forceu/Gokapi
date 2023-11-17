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
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sort"
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
	"github.com/forceu/gokapi/internal/storage/processingstatus"
	"github.com/forceu/gokapi/internal/webserver/api"
	"github.com/forceu/gokapi/internal/webserver/authentication"
	"github.com/forceu/gokapi/internal/webserver/authentication/oauth"
	"github.com/forceu/gokapi/internal/webserver/authentication/sessionmanager"
	"github.com/forceu/gokapi/internal/webserver/fileupload"
	"github.com/forceu/gokapi/internal/webserver/guest"
	"github.com/forceu/gokapi/internal/webserver/ssl"
	"github.com/r3labs/sse/v2"
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
var sseServer *sse.Server

// Start the webserver on the port set in the config
func Start() {
	initTemplates(templateFolderEmbedded)
	webserverDir, _ := fs.Sub(staticFolderEmbedded, "web/static")
	var err error

	mux := http.NewServeMux()
	sseServer = sse.New()
	sseServer.CreateStream("changes")
	processingstatus.Init(sseServer)

	if helper.FolderExists("static") {
		fmt.Println("Found folder 'static', using local folder instead of internal static folder")
		mux.Handle("/", http.FileServer(http.Dir("static")))
	} else {
		mux.Handle("/", http.FileServer(http.FS(webserverDir)))
		helper.Check(err)
	}
	loadExpiryImage()

	mux.HandleFunc("/admin", requireLogin(showAdminMenu, false))
	mux.HandleFunc("/api/", processApi)
	mux.HandleFunc("/apiDelete", requireLogin(deleteApiKey, false))
	mux.HandleFunc("/apiKeys", requireLogin(showApiAdmin, false))
	mux.HandleFunc("/apiNew", requireLogin(newApiKey, false))
	mux.HandleFunc("/d", showDownload)
	mux.HandleFunc("/delete", requireLogin(deleteFile, false))
	mux.HandleFunc("/downloadFile", downloadFile)
	mux.HandleFunc("/e2eInfo", requireLogin(e2eInfo, true))
	mux.HandleFunc("/e2eSetup", requireLogin(showE2ESetup, false))
	mux.HandleFunc("/error-auth", showErrorAuth)
	mux.HandleFunc("/error", showError)
	mux.HandleFunc("/forgotpw", forgotPassword)
	mux.HandleFunc("/guest", requireValidGuestToken(showGuest, false))
	mux.HandleFunc("/guestTokens", requireLogin(showGuestTokenMenu, false))
	mux.HandleFunc("/hotlink/", showHotlink)
	mux.HandleFunc("/index", showIndex)
	mux.HandleFunc("/login", showLogin)
	mux.HandleFunc("/logout", doLogout)
	mux.HandleFunc("/logs", requireLogin(showLogs, false))
	mux.HandleFunc("/newGuestToken", requireLogin(newGuestToken, false))
	mux.HandleFunc("/guestTokenDelete", requireLogin(guestTokenDelete, false))
	mux.HandleFunc("/uploadChunk", requireLogin(uploadChunk, true))
	mux.HandleFunc("/uploadComplete", requireLogin(uploadComplete, true))
	mux.HandleFunc("/uploadStatus", requireLogin(sseServer.ServeHTTP, false))
	mux.HandleFunc("/guestUploadChunk", guestUploadChunk)
	mux.HandleFunc("/guestUploadComplete", guestUploadComplete)
	mux.HandleFunc("/guestUploadStatus", requireValidGuestToken(sseServer.ServeHTTP, false))
	mux.Handle("/main.wasm", gziphandler.GzipHandler(http.HandlerFunc(serveDownloadWasm)))
	mux.Handle("/e2e.wasm", gziphandler.GzipHandler(http.HandlerFunc(serveE2EWasm)))
	if configuration.Get().Authentication.Method == authentication.OAuth2 {
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
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	} else {
		fmt.Println(infoMessage)
		err = srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}
}

func loadExpiryImage() {
	svgTemplate, err := templatetext.ParseFS(templateFolderEmbedded, "web/templates/expired_file_svg.tmpl")
	helper.Check(err)
	var buf bytes.Buffer
	view := GenericView{}
	view.initView()
	err = svgTemplate.Execute(&buf, view)
	helper.Check(err)
	imageExpiredPicture = buf.Bytes()
}

// Shutdown closes the webserver gracefully
func Shutdown() {
	sseServer.Close()
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
	if helper.FolderExists("templates") {
		fmt.Println("Found folder 'templates', using local folder instead of internal template folder")
		templateFolder, err = template.ParseGlob("templates/*.tmpl")
		helper.Check(err)
	} else {
		templateFolder, err = template.ParseFS(templateFolderEmbedded, "web/templates/*.tmpl")
		helper.Check(err)
	}
}

// Sends a redirect HTTP output to the client. Variable url is used to redirect to ./url
func redirect(w http.ResponseWriter, url string) {
	_, _ = io.WriteString(w, "<html><head><meta http-equiv=\"Refresh\" content=\"0; URL=./"+url+"\"></head></html>")
}

// Handling of /main.wasm
func serveDownloadWasm(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Cache-Control", "public, max-age=100800") // 2 days
	w.Header().Set("content-type", "application/wasm")
	file, err := wasmDownloadFile.ReadFile("web/main.wasm")
	helper.Check(err)
	w.Write(file)
}

// Handling of /e2e.wasm
func serveE2EWasm(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Cache-Control", "public, max-age=100800") // 2 days
	w.Header().Set("content-type", "application/wasm")
	file, err := wasmE2EFile.ReadFile("web/e2e.wasm")
	helper.Check(err)
	w.Write(file)
}

// Handling of /logout
func doLogout(w http.ResponseWriter, r *http.Request) {
	authentication.Logout(w, r)
}

// Handling of /index and redirecting to globalConfig.RedirectUrl
func showIndex(w http.ResponseWriter, r *http.Request) {
	view := GenericView{RedirectUrl: configuration.Get().RedirectUrl}
	view.initView()
	err := templateFolder.ExecuteTemplate(w, "index", view)
	helper.Check(err)
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
	view := GenericView{ErrorId: errorReason}
	view.initView()
	err := templateFolder.ExecuteTemplate(w, "error", view)
	helper.Check(err)
}

// Handling of /error-auth
func showErrorAuth(w http.ResponseWriter, r *http.Request) {
	view := GenericView{}
	view.initView()
	err := templateFolder.ExecuteTemplate(w, "error_auth", view)
	helper.Check(err)
}

// Handling of /forgotpw
func forgotPassword(w http.ResponseWriter, r *http.Request) {
	view := GenericView{}
	view.initView()
	err := templateFolder.ExecuteTemplate(w, "forgotpw", view)
	helper.Check(err)
}

// Handling of /api
// If user is authenticated, this menu lists all uploads and enables uploading new files
func showApiAdmin(w http.ResponseWriter, r *http.Request) {
	view := &APIView{
		AdminView: AdminView{
			ActiveView: ViewAPI,
		},
	}
	view.initView()
	err := templateFolder.ExecuteTemplate(w, "api", view)
	helper.Check(err)
}

// Handling of /apiNew
func newApiKey(w http.ResponseWriter, r *http.Request) {
	api.NewKey()
	redirect(w, "apiKeys")
}

// Handling of /apiDelete
func deleteApiKey(w http.ResponseWriter, r *http.Request) {
	keys, ok := r.URL.Query()["id"]
	if ok {
		api.DeleteKey(keys[0])
	}
	redirect(w, "apiKeys")
}

// Handling of /api/
func processApi(w http.ResponseWriter, r *http.Request) {
	api.Process(w, r, configuration.Get().MaxMemory)
}

// Handling of /login
// Shows a login form. If not authenticated, client needs to wait for three seconds.
// If correct, a new session is created and the user is redirected to the admin menu
func showLogin(w http.ResponseWriter, r *http.Request) {
	if authentication.IsAuthenticated(w, r) {
		redirect(w, "admin")
		return
	}
	if configuration.Get().Authentication.Method == authentication.OAuth2 {
		redirect(w, "oauth-login")
		return
	}
	err := r.ParseForm()
	helper.Check(err)
	user := r.Form.Get("username")
	pw := r.Form.Get("password")
	failedLogin := false
	if pw != "" && user != "" {
		if authentication.IsCorrectUsernameAndPassword(user, pw) {
			sessionmanager.CreateSession(w)
			redirect(w, "admin")
			return
		}
		select {
		case <-time.After(3 * time.Second):
		}
		failedLogin = true
	}
	view := LoginView{
		IsFailedLogin: failedLogin,
		User:          user,
	}
	view.initView()
	err = templateFolder.ExecuteTemplate(w, "login", view)
	helper.Check(err)
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

	view := DownloadView{
		GenericView: GenericView{
			ViewType: ViewTypeDownload,
		},
		Name:               file.Name,
		Size:               file.Size,
		Id:                 file.Id,
		EndToEndEncryption: file.Encryption.IsEndToEndEncrypted,
		IsFailedLogin:      false,
		UsesHttps:          configuration.UsesHttps(),
	}
	(&view).initView()

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
			helper.Check(err)
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
	helper.Check(err)
}

// Handling of /hotlink/
// Hotlinks an image or returns a static error image if image has expired
func showHotlink(w http.ResponseWriter, r *http.Request) {
	addNoCacheHeader(w)
	hotlinkId := strings.Replace(r.URL.Path, "/hotlink/", "", 1)
	file, ok := storage.GetFileByHotlink(hotlinkId)
	if !ok {
		w.Header().Set("Content-Type", "image/svg+xml")
		_, err := w.Write(imageExpiredPicture)
		helper.Check(err)
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
	switch action[0] {
	case "get":
		getE2eInfo(w)
	case "store":
		storeE2eInfo(w, r)
	default:
		responseError(w, errors.New("invalid action specified"))
	}
}

func storeE2eInfo(w http.ResponseWriter, r *http.Request) {
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
	database.SaveEnd2EndInfo(info)
	_, _ = w.Write([]byte("\"result\":\"OK\""))
}

func getE2eInfo(w http.ResponseWriter) {
	info := database.GetEnd2EndInfo()
	bytes, err := json.Marshal(info)
	helper.Check(err)
	_, _ = w.Write(bytes)
}

// Handling of /delete
// User needs to be admin. Deletes the requested file
func deleteFile(w http.ResponseWriter, r *http.Request) {
	keyId := queryUrl(w, r, "admin")
	if keyId == "" {
		return
	}
	storage.DeleteFile(keyId, true)
	redirect(w, "admin")
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
	if configuration.Get().Encryption.Level == encryption.EndToEndEncryption {
		e2einfo := database.GetEnd2EndInfo()
		if !e2einfo.HasBeenSetUp() {
			redirect(w, "e2eSetup")
			return
		}
	}
	view := UploadView{
		AdminView: AdminView{
			ActiveView: ViewMain,
		},
	}
	view.initView()
	err := templateFolder.ExecuteTemplate(w, "admin", view)
	helper.Check(err)
}

// Handling of /guestTokens
// If user is authenticated, this menu lets the user create new guest tokens
func showGuestTokenMenu(w http.ResponseWriter, r *http.Request) {
	view := GuestTokenView{
		AdminView: AdminView{
			ActiveView: ViewGuestTokens,
		},
	}
	view.initView()
	err := templateFolder.ExecuteTemplate(w, "guesttokens", view)
	helper.Check(err)
}

// Handling of /logs
// If user is authenticated, this menu shows the stored logs
func showLogs(w http.ResponseWriter, r *http.Request) {
	view := LogView{
		AdminView: AdminView{
			ActiveView: ViewLogs,
		},
	}
	view.initView()
	err := templateFolder.ExecuteTemplate(w, "logs", view)
	helper.Check(err)
}

func showE2ESetup(w http.ResponseWriter, r *http.Request) {
	if configuration.Get().Encryption.Level != encryption.EndToEndEncryption {
		redirect(w, "admin")
		return
	}
	e2einfo := database.GetEnd2EndInfo()
	view := E2ESetupView{HasBeenSetup: e2einfo.HasBeenSetUp()}
	view.initView()
	err := templateFolder.ExecuteTemplate(w, "e2esetup", view)
	helper.Check(err)
}

// Guest Uploads
func showGuest(w http.ResponseWriter, r *http.Request) {
	addNoCacheHeader(w)
	tokenId := queryToken(w, r, "error")
	token, ok := database.GetGuestToken(tokenId)
	if !ok {
		redirect(w, "error")
		return
	}

	config := configuration.Get()

	view := GuestUploadView{
		GenericView: GenericView{
			ViewType: ViewTypeGuestUpload,
		},
		GuestToken:         token.Id,
		MaxFileSize:        config.MaxFileSizeMB,
		EndToEndEncryption: config.Encryption.Level == encryption.EndToEndEncryption,
	}
	(&view).initView()

	err := templateFolder.ExecuteTemplate(w, "guest", view)
	helper.Check(err)
}

func queryToken(w http.ResponseWriter, r *http.Request, redirectUrl string) string {
	config := configuration.Get()

	getTokens, ok := r.URL.Query()["token"]
	if ok && len(getTokens[0]) >= config.LengthId {
		return getTokens[0]
	}

	err := r.ParseForm()
	if err != nil {
		redirect(w, redirectUrl)
		return ""
	}
	formToken := r.Form.Get("token")
	if formToken != "" && len(formToken) >= config.LengthId {
		return formToken
	}
	select {
	case <-time.After(500 * time.Millisecond):
	}
	redirect(w, redirectUrl)
	return ""
}

func newGuestToken(w http.ResponseWriter, r *http.Request) {
	guest.NewToken()
	redirect(w, "guestTokens")
}

func guestTokenDelete(w http.ResponseWriter, r *http.Request) {
	tokenId := queryToken(w, r, "error")
	ok := guest.DeleteToken(tokenId)
	if !ok {
		redirect(w, "error")
		return
	}
	redirect(w, "guestTokens")
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

// Handling of /uploadComplete
// If the user is authenticated, this parses the uploaded chunk and stores it
func uploadComplete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	err := fileupload.CompleteChunk(w, r, false)
	responseError(w, err)
}

// Handling of /guestUploadChunk
// If the guesttoken is valid, this parses the uploaded chunk and stores it
func guestUploadChunk(w http.ResponseWriter, r *http.Request) {
	maxUpload := int64(configuration.Get().MaxFileSizeMB) * 1024 * 1024
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if r.ContentLength > maxUpload {
		responseError(w, storage.ErrorFileTooLarge)
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
	err := fileupload.ProcessNewChunk(w, r, false)
	responseError(w, err)

	// TODO: Update the guesttoken in the database
}

// Handling of /guestUploadComplete
// If the guesttoken is valid, this parses the uploaded chunk and stores it
func guestUploadComplete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	err := fileupload.CompleteChunk(w, r, false)
	responseError(w, err)

	// TODO: Update the guesttoken in the database
}

// Outputs an error in json format if err!=nil
func responseError(w http.ResponseWriter, err error) {
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, "{\"Result\":\"error\",\"ErrorMessage\":\""+err.Error()+"\"}")
		log.Println(err)
	}
}

// Outputs the file to the user and reduces the download remaining count for the file
func downloadFile(w http.ResponseWriter, r *http.Request) {
	addNoCacheHeader(w)
	keyId := queryUrl(w, r, "error")
	savedFile, ok := storage.GetFile(keyId)
	if !ok {
		redirect(w, "error")
		return
	}
	if savedFile.PasswordHash != "" {
		if !(isValidPwCookie(r, savedFile)) {
			redirect(w, "d?id="+savedFile.Id)
			return
		}
	}
	storage.ServeFile(savedFile, w, r, true)
}

func requireLogin(next http.HandlerFunc, isUpload bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		addNoCacheHeader(w)
		if authentication.IsAuthenticated(w, r) {
			next.ServeHTTP(w, r)
			return
		}
		if isUpload {
			_, err := io.WriteString(w, "{\"Result\":\"error\",\"ErrorMessage\":\"Not authenticated\"}")
			helper.Check(err)
			return
		}
		redirect(w, "login")
	}
}

func requireValidGuestToken(next http.HandlerFunc, isUpload bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		addNoCacheHeader(w)

		tokenId := queryToken(w, r, "error")
		_, ok := database.GetGuestToken(tokenId)

		if ok {
			next.ServeHTTP(w, r)
			return
		}
		if isUpload {
			_, err := io.WriteString(w, "{\"Result\":\"error\",\"ErrorMessage\":\"Invalid Guest Token\"}")
			helper.Check(err)
			return
		}
		redirect(w, "/")
	}
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
	w.Header().Set("cache-control", "no-store")
}

const (
	ViewTypeAdmin       string = "admin"
	ViewTypeLogin       string = "login"
	ViewTypeGuestUpload string = "guestupload"
	ViewTypeDownload    string = "download"
	ViewTypeError       string = "error"
)

// GenericView contains parameters for all templates
type GenericView struct {
	PublicName  string
	RedirectUrl string
	ViewType    string
	ErrorId     int
}

func (v *GenericView) initView() {
	v.PublicName = configuration.Get().PublicName
}

// LoginView contains parameters for the login template
type LoginView struct {
	GenericView
	IsFailedLogin bool
	User          string
}

// DownloadView contains parameters for the download template
type DownloadView struct {
	GenericView
	Name                 string
	Size                 string
	Id                   string
	Cipher               string
	IsFailedLogin        bool
	IsPasswordView       bool
	ClientSideDecryption bool
	EndToEndEncryption   bool
	UsesHttps            bool
}

type GuestUploadView struct {
	GenericView
	GuestToken         string
	MaxFileSize        int
	EndToEndEncryption bool
}

type E2ESetupView struct {
	GenericView
	HasBeenSetup bool
}

// AdminView contains parameters for the admin templates
type AdminView struct {
	GenericView
	ActiveView        int
	IsLogoutAvailable bool
	TimeNow           int64
}

const (
	ViewMain        int = 0
	ViewLogs        int = 1
	ViewAPI         int = 2
	ViewGuestTokens int = 3
)

func (v *AdminView) initView() {
	v.GenericView.initView()
	v.ViewType = ViewTypeAdmin
	v.IsLogoutAvailable = authentication.IsLogoutAvailable()
	v.TimeNow = time.Now().Unix()
}

// UploadView contains parameters for the upload admin template
type UploadView struct {
	AdminView
	Items                    []models.FileApiOutput
	Url                      string
	HotlinkUrl               string
	GenericHotlinkUrl        string
	DefaultPassword          string
	DefaultUnlimitedDownload bool
	DefaultUnlimitedTime     bool
	DefaultDownloads         int
	DefaultExpiry            int
	MaxFileSize              int
	EndToEndEncryption       bool
}

func (v *UploadView) initView() {
	v.AdminView.initView()

	config := configuration.Get()
	defaultValues := database.GetUploadDefaults()

	var result []models.FileApiOutput
	for _, element := range database.GetAllMetadata() {
		fileInfo, err := element.ToFileApiOutput()
		helper.Check(err)
		result = append(result, fileInfo)
	}
	sort.Slice(result[:], func(i, j int) bool {
		if result[i].ExpireAt == result[j].ExpireAt {
			return result[i].Id > result[j].Id
		}
		return result[i].ExpireAt > result[j].ExpireAt
	})
	v.Items = result
	v.Url = config.ServerUrl + "d?id="
	v.HotlinkUrl = config.ServerUrl + "hotlink/"
	v.GenericHotlinkUrl = config.ServerUrl + "downloadFile?id="
	v.DefaultPassword = defaultValues.Password
	v.DefaultUnlimitedDownload = defaultValues.UnlimitedDownload
	v.DefaultUnlimitedTime = defaultValues.UnlimitedTime
	v.DefaultDownloads = defaultValues.Downloads
	v.DefaultExpiry = defaultValues.TimeExpiry
	v.MaxFileSize = config.MaxFileSizeMB
	v.EndToEndEncryption = config.Encryption.Level == encryption.EndToEndEncryption
}

type LogView struct {
	AdminView
	Logs string
}

func (v *LogView) initView() {
	v.AdminView.initView()

	if helper.FileExists(logging.GetLogPath()) {
		content, err := os.ReadFile(logging.GetLogPath())
		helper.Check(err)
		v.Logs = string(content)
	} else {
		v.Logs = "Warning: Log file not found!"
	}
}

type GuestTokenView struct {
	AdminView
	GuestUploadUrl string
	GuestTokens    []models.GuestToken
}

func (v *GuestTokenView) initView() {
	v.AdminView.initView()

	config := configuration.Get()

	var result []models.GuestToken
	for _, element := range database.GetAllGuestTokens() {
		if element.LastUsed == 0 {
			element.LastUsedString = "Never"
		} else {
			element.LastUsedString = time.Unix(element.LastUsed, 0).Format("2006-01-02 15:04:05")
		}
		if element.ExpireAt == 0 {
			element.ExpireAtString = "Never"
		} else {
			element.ExpireAtString = time.Unix(element.ExpireAt, 0).Format("2006-01-02 15:04:05")
		}
		result = append(result, element)
	}
	sort.Slice(result[:], func(i, j int) bool {
		if result[i].LastUsed == result[j].LastUsed {
			return result[i].Id < result[j].Id
		}
		return result[i].LastUsed > result[j].LastUsed
	})

	v.GuestTokens = result
	v.GuestUploadUrl = config.ServerUrl + "guest?token="
}

type APIView struct {
	AdminView
	ApiKeys []models.ApiKey
}

func (v *APIView) initView() {
	v.AdminView.initView()

	var result []models.ApiKey

	for _, element := range database.GetAllApiKeys() {
		if element.LastUsed == 0 {
			element.LastUsedString = "Never"
		} else {
			element.LastUsedString = time.Unix(element.LastUsed, 0).Format("2006-01-02 15:04:05")
		}
		result = append(result, element)
	}
	sort.Slice(result[:], func(i, j int) bool {
		if result[i].LastUsed == result[j].LastUsed {
			return result[i].Id < result[j].Id
		}
		return result[i].LastUsed > result[j].LastUsed
	})

	v.ApiKeys = result
}
