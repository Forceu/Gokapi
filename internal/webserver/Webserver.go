package webserver

/**
Handling of webserver and requests / uploads
*/

import (
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"github.com/NYTimes/gziphandler"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/encryption"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage"
	"github.com/forceu/gokapi/internal/webserver/api"
	"github.com/forceu/gokapi/internal/webserver/authentication"
	"github.com/forceu/gokapi/internal/webserver/authentication/oauth"
	"github.com/forceu/gokapi/internal/webserver/authentication/sessionmanager"
	"github.com/forceu/gokapi/internal/webserver/fileupload"
	"github.com/forceu/gokapi/internal/webserver/ssl"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// TODO add 404 handler

// staticFolderEmbedded is the embedded version of the "static" folder
// This contains JS files, CSS, images etc
//go:embed web/static
var staticFolderEmbedded embed.FS

// templateFolderEmbedded is the embedded version of the "templates" folder
// This contains templates that Gokapi uses for creating the HTML output
//go:embed web/templates
var templateFolderEmbedded embed.FS

// wasmFile is the compiled binary of the wasm downloader
// Will be generated with go generate ./...
//go:embed web/main.wasm
var wasmFile embed.FS

const timeOutWebserver = 12 * time.Hour

// Variable containing all parsed templates
var templateFolder *template.Template

var imageExpiredPicture []byte

const expiredFile = "static/expired.png"

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
		imageExpiredPicture, err = os.ReadFile(expiredFile)
		helper.Check(err)
	} else {
		mux.Handle("/", http.FileServer(http.FS(webserverDir)))
		imageExpiredPicture, err = fs.ReadFile(staticFolderEmbedded, "web/"+expiredFile)
		helper.Check(err)
	}
	mux.HandleFunc("/admin", requireLogin(showAdminMenu, false))
	mux.HandleFunc("/api/", processApi)
	mux.HandleFunc("/apiDelete", requireLogin(deleteApiKey, false))
	mux.HandleFunc("/apiKeys", requireLogin(showApiAdmin, false))
	mux.HandleFunc("/apiNew", requireLogin(newApiKey, false))
	mux.HandleFunc("/d", showDownload)
	mux.HandleFunc("/delete", requireLogin(deleteFile, false))
	mux.HandleFunc("/downloadFile", downloadFile)
	mux.HandleFunc("/error", showError)
	mux.HandleFunc("/forgotpw", forgotPassword)
	mux.HandleFunc("/hotlink/", showHotlink)
	mux.HandleFunc("/index", showIndex)
	mux.HandleFunc("/login", showLogin)
	mux.HandleFunc("/logout", doLogout)
	mux.HandleFunc("/uploadChunk", requireLogin(uploadChunk, true))
	mux.HandleFunc("/uploadComplete", requireLogin(uploadComplete, true))
	mux.HandleFunc("/error-auth", showErrorAuth)
	mux.Handle("/main.wasm", gziphandler.GzipHandler(http.HandlerFunc(serveWasm)))
	if configuration.Get().Authentication.Method == authentication.OAuth2 {
		oauth.Init(configuration.Get().ServerUrl, configuration.Get().Authentication)
		mux.HandleFunc("/oauth-login", oauth.HandlerLogin)
		mux.HandleFunc("/oauth-callback", oauth.HandlerCallback)
	}

	fmt.Println("Binding webserver to " + configuration.Get().Port)
	srv = http.Server{
		Addr:         configuration.Get().Port,
		ReadTimeout:  timeOutWebserver,
		WriteTimeout: timeOutWebserver,
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

// Shutdown closes the webserver gracefully
func Shutdown() {
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
func serveWasm(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Cache-Control", "public, max-age=100800") // 2 days
	w.Header().Set("content-type", "application/wasm")
	file, err := wasmFile.ReadFile("web/main.wasm")
	helper.Check(err)
	w.Write(file)
}

// Handling of /logout
func doLogout(w http.ResponseWriter, r *http.Request) {
	authentication.Logout(w, r)
}

// Handling of /index and redirecting to globalConfig.RedirectUrl
func showIndex(w http.ResponseWriter, r *http.Request) {
	err := templateFolder.ExecuteTemplate(w, "index", genericView{RedirectUrl: configuration.Get().RedirectUrl})
	helper.Check(err)
}

// Handling of /error
func showError(w http.ResponseWriter, r *http.Request) {
	err := templateFolder.ExecuteTemplate(w, "error", genericView{})
	helper.Check(err)
}

// Handling of /error-auth
func showErrorAuth(w http.ResponseWriter, r *http.Request) {
	err := templateFolder.ExecuteTemplate(w, "error_auth", genericView{})
	helper.Check(err)
}

// Handling of /forgotpw
func forgotPassword(w http.ResponseWriter, r *http.Request) {
	err := templateFolder.ExecuteTemplate(w, "forgotpw", genericView{})
	helper.Check(err)
}

// Handling of /api
// If user is authenticated, this menu lists all uploads and enables uploading new files
func showApiAdmin(w http.ResponseWriter, r *http.Request) {
	err := templateFolder.ExecuteTemplate(w, "api", (&UploadView{}).convertGlobalConfig(false))
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
		time.Sleep(3 * time.Second)
		failedLogin = true
	}
	err = templateFolder.ExecuteTemplate(w, "login", LoginView{
		IsFailedLogin: failedLogin,
		User:          user,
		IsAdminView:   false,
	})
	helper.Check(err)
}

// LoginView contains variables for the login template
type LoginView struct {
	IsFailedLogin bool
	User          string
	IsAdminView   bool
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
		Name:          file.Name,
		Size:          file.Size,
		Id:            file.Id,
		IsFailedLogin: false,
		UsesHttps:     configuration.UsesHttps(),
	}

	if storage.RequiresClientDecryption(file) {
		view.ClientSideDecryption = true
		cipher, err := encryption.GetCipherFromFile(file.Encryption)
		helper.Check(err)
		view.Cipher = base64.StdEncoding.EncodeToString(cipher)
	}

	if file.PasswordHash != "" {
		r.ParseForm()
		enteredPassword := r.Form.Get("password")
		if configuration.HashPassword(enteredPassword, true) != file.PasswordHash && !isValidPwCookie(r, file) {
			if enteredPassword != "" {
				view.IsFailedLogin = true
				time.Sleep(1 * time.Second)
			}
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
		w.Header().Set("Content-Type", "image/png")
		_, err := w.Write(imageExpiredPicture)
		helper.Check(err)
		return
	}
	storage.ServeFile(file, w, r, false)
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
		time.Sleep(500 * time.Millisecond)
		redirect(w, redirectUrl)
		return ""
	}
	return keys[0]
}

// Handling of /admin
// If user is authenticated, this menu lists all uploads and enables uploading new files
func showAdminMenu(w http.ResponseWriter, r *http.Request) {
	err := templateFolder.ExecuteTemplate(w, "admin", (&UploadView{}).convertGlobalConfig(true))
	helper.Check(err)
}

// DownloadView contains parameters for the download template
type DownloadView struct {
	Name                 string
	Size                 string
	Id                   string
	IsFailedLogin        bool
	IsAdminView          bool
	ClientSideDecryption bool
	UsesHttps            bool
	Cipher               string
}

// UploadView contains parameters for the admin menu template
type UploadView struct {
	Items                    []models.FileApiOutput
	ApiKeys                  []models.ApiKey
	Url                      string
	HotlinkUrl               string
	GenericHotlinkUrl        string
	TimeNow                  int64
	IsAdminView              bool
	IsMainView               bool
	IsApiView                bool
	MaxFileSize              int
	IsLogoutAvailable        bool
	DefaultDownloads         int
	DefaultExpiry            int
	DefaultPassword          string
	DefaultUnlimitedDownload bool
	DefaultUnlimitedTime     bool
}

// Converts the globalConfig variable to an UploadView struct to pass the infos to
// the admin template
func (u *UploadView) convertGlobalConfig(isMainView bool) *UploadView {
	var result []models.FileApiOutput
	var resultApi []models.ApiKey
	if isMainView {
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
	} else {
		for _, element := range database.GetAllApiKeys() {
			if element.LastUsed == 0 {
				element.LastUsedString = "Never"
			} else {
				element.LastUsedString = time.Unix(element.LastUsed, 0).Format("2006-01-02 15:04:05")
			}
			resultApi = append(resultApi, element)
		}
		sort.Slice(resultApi[:], func(i, j int) bool {
			if resultApi[i].LastUsed == resultApi[j].LastUsed {
				return resultApi[i].Id < resultApi[j].Id
			}
			return resultApi[i].LastUsed > resultApi[j].LastUsed
		})
	}
	u.Url = configuration.Get().ServerUrl + "d?id="
	u.HotlinkUrl = configuration.Get().ServerUrl + "hotlink/"
	u.GenericHotlinkUrl = configuration.Get().ServerUrl + "downloadFile?id="
	u.Items = result
	u.ApiKeys = resultApi
	u.TimeNow = time.Now().Unix()
	u.IsAdminView = true
	u.IsMainView = isMainView
	u.MaxFileSize = configuration.Get().MaxFileSizeMB
	u.IsLogoutAvailable = authentication.IsLogoutAvailable()
	defaultValues := database.GetUploadDefaults()
	u.DefaultDownloads = defaultValues.Downloads
	u.DefaultExpiry = defaultValues.TimeExpiry
	u.DefaultPassword = defaultValues.Password
	u.DefaultUnlimitedDownload = defaultValues.UnlimitedDownload
	u.DefaultUnlimitedTime = defaultValues.UnlimitedTime
	return u
}

// Handling of /uploadChunk
// If the user is authenticated, this parses the uploaded chunk and stores it
func uploadChunk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	err := fileupload.ProcessChunk(w, r)
	responseError(w, err)
}

// Handling of /uploadComplete
// If the user is authenticated, this parses the uploaded chunk and stores it
func uploadComplete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	err := fileupload.CompleteChunk(w, r, true)
	responseError(w, err)
}

// Outputs an error in json format
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

// Write a cookie if the user has entered a correct password for a password-protected file
func writeFilePwCookie(w http.ResponseWriter, file models.File) {
	http.SetCookie(w, &http.Cookie{
		Name:    "p" + file.Id,
		Value:   file.PasswordHash,
		Expires: time.Now().Add(5 * time.Minute),
	})
}

// Checks if a cookie contains the correct password hash for a password-protected file
// If incorrect, a 3 second delay is introduced unless the cookie was empty.
func isValidPwCookie(r *http.Request, file models.File) bool {
	cookie, err := r.Cookie("p" + file.Id)
	if err == nil {
		if cookie.Value == file.PasswordHash {
			return true
		}
		time.Sleep(3 * time.Second)
	}
	return false
}

// Adds a header to disable external caching
func addNoCacheHeader(w http.ResponseWriter) {
	w.Header().Set("cache-control", "no-store")
}

// A view containing parameters for a generic template
type genericView struct {
	IsAdminView bool
	RedirectUrl string
}
