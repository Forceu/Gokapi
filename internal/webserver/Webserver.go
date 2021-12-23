package webserver

/**
Handling of webserver and requests / uploads
*/

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/helper"
	"Gokapi/internal/models"
	"Gokapi/internal/storage"
	"Gokapi/internal/webserver/api"
	"Gokapi/internal/webserver/authentication"
	"Gokapi/internal/webserver/authentication/oauth"
	"Gokapi/internal/webserver/fileupload"
	"Gokapi/internal/webserver/sessionmanager"
	"Gokapi/internal/webserver/ssl"
	"embed"
	"fmt"
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

const timeOutWebserver = 12 * time.Hour

// Variable containing all parsed templates
var templateFolder *template.Template

var imageExpiredPicture []byte

const expiredFile = "static/expired.png"

var (
	webserverRedirectUrl string
	webserverMaxMemory   int
)

// Start the webserver on the port set in the config
func Start() {
	settings := configuration.GetServerSettingsReadOnly()
	configuration.ReleaseReadOnly()
	webserverRedirectUrl = settings.RedirectUrl
	webserverMaxMemory = settings.MaxMemory

	initTemplates(templateFolderEmbedded)
	webserverDir, _ := fs.Sub(staticFolderEmbedded, "web/static")
	var err error
	if helper.FolderExists("static") {
		fmt.Println("Found folder 'static', using local folder instead of internal static folder")
		http.Handle("/", http.FileServer(http.Dir("static")))
		imageExpiredPicture, err = os.ReadFile(expiredFile)
		helper.Check(err)
	} else {
		http.Handle("/", http.FileServer(http.FS(webserverDir)))
		imageExpiredPicture, err = fs.ReadFile(staticFolderEmbedded, "web/"+expiredFile)
		helper.Check(err)
	}
	http.HandleFunc("/admin", showAdminMenu)
	http.HandleFunc("/api/", processApi)
	http.HandleFunc("/apiDelete", deleteApiKey)
	http.HandleFunc("/apiKeys", showApiAdmin)
	http.HandleFunc("/apiNew", newApiKey)
	http.HandleFunc("/d", showDownload)
	http.HandleFunc("/delete", deleteFile)
	http.HandleFunc("/downloadFile", downloadFile)
	http.HandleFunc("/error", showError)
	http.HandleFunc("/forgotpw", forgotPassword)
	http.HandleFunc("/hotlink/", showHotlink)
	http.HandleFunc("/index", showIndex)
	http.HandleFunc("/login", showLogin)
	http.HandleFunc("/logout", doLogout)
	http.HandleFunc("/upload", uploadFile)
	http.HandleFunc("/error-auth", showErrorAuth)
	if settings.Authentication.Method == models.AuthenticationOAuth2 {
		oauth.Init(settings.ServerUrl, settings.Authentication)
		http.HandleFunc("/oauth-login", oauth.HandlerLogin)
		http.HandleFunc("/oauth-callback", oauth.HandlerCallback)
	}

	fmt.Println("Binding webserver to " + settings.Port)
	srv := &http.Server{
		Addr:         settings.Port,
		ReadTimeout:  timeOutWebserver,
		WriteTimeout: timeOutWebserver,
	}
	infoMessage := "Webserver can be accessed at " + settings.ServerUrl + "admin"
	if strings.Contains(settings.ServerUrl, "127.0.0.1") {
		if settings.UseSsl {
			infoMessage = strings.Replace(infoMessage, "http://", "https://", 1)
		} else {
			infoMessage = strings.Replace(infoMessage, "https://", "http://", 1)
		}
	}
	if settings.UseSsl {
		ssl.GenerateIfInvalidCert(settings.ServerUrl, false)
		fmt.Println(infoMessage)
		log.Fatal(srv.ListenAndServeTLS(ssl.GetCertificateLocations()))
	} else {
		fmt.Println(infoMessage)
		log.Fatal(srv.ListenAndServe())
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

// Handling of /logout
func doLogout(w http.ResponseWriter, r *http.Request) {
	authentication.Logout(w, r)
}

// Handling of /index and redirecting to globalConfig.RedirectUrl
func showIndex(w http.ResponseWriter, r *http.Request) {
	err := templateFolder.ExecuteTemplate(w, "index", genericView{RedirectUrl: webserverRedirectUrl})
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
	addNoCacheHeader(w)
	if !isAuthenticatedOrRedirect(w, r, false) {
		return
	}
	err := templateFolder.ExecuteTemplate(w, "api", (&UploadView{}).convertGlobalConfig(false))
	helper.Check(err)
}

// Handling of /apiNew
func newApiKey(w http.ResponseWriter, r *http.Request) {
	addNoCacheHeader(w)
	if !isAuthenticatedOrRedirect(w, r, false) {
		return
	}
	api.NewKey()
	redirect(w, "apiKeys")
}

// Handling of /apiDelete
func deleteApiKey(w http.ResponseWriter, r *http.Request) {
	addNoCacheHeader(w)
	if !isAuthenticatedOrRedirect(w, r, false) {
		return
	}
	keys, ok := r.URL.Query()["id"]
	if ok {
		api.DeleteKey(keys[0])
	}
	redirect(w, "apiKeys")
}

// Handling of /api/
func processApi(w http.ResponseWriter, r *http.Request) {
	api.Process(w, r, webserverMaxMemory)
}

// Handling of /login
// Shows a login form. If username / pw combo is incorrect, client needs to wait for three seconds.
// If correct, a new session is created and the user is redirected to the admin menu
func showLogin(w http.ResponseWriter, r *http.Request) {
	if authentication.IsAuthenticated(w, r) {
		redirect(w, "admin")
		return
	}
	if authentication.GetMethod() == models.AuthenticationOAuth2 {
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
			sessionmanager.CreateSession(w, nil)
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
		w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
		_, err := w.Write(imageExpiredPicture)
		helper.Check(err)
		return
	}
	storage.ServeFile(file, w, r, false)
}

// Handling of /delete
// User needs to be admin. Deletes the requested file
func deleteFile(w http.ResponseWriter, r *http.Request) {
	addNoCacheHeader(w)
	if !isAuthenticatedOrRedirect(w, r, false) {
		return
	}
	keyId := queryUrl(w, r, "admin")
	if keyId == "" {
		return
	}
	storage.DeleteFile(keyId)
	redirect(w, "admin")
}

// Checks if a file is associated with the GET parameter from the current URL
// Stops for 500ms to limit brute forcing if invalid key and redirects to redirectUrl
func queryUrl(w http.ResponseWriter, r *http.Request, redirectUrl string) string {
	keys, ok := r.URL.Query()["id"]
	if !ok || len(keys[0]) < configuration.GetLengthId() {
		time.Sleep(500 * time.Millisecond)
		redirect(w, redirectUrl)
		return ""
	}
	return keys[0]
}

// Handling of /admin
// If user is authenticated, this menu lists all uploads and enables uploading new files
func showAdminMenu(w http.ResponseWriter, r *http.Request) {
	addNoCacheHeader(w)
	if !isAuthenticatedOrRedirect(w, r, false) {
		return
	}
	err := templateFolder.ExecuteTemplate(w, "admin", (&UploadView{}).convertGlobalConfig(true))
	helper.Check(err)
}

// DownloadView contains parameters for the download template
type DownloadView struct {
	Name          string
	Size          string
	Id            string
	IsFailedLogin bool
	IsAdminView   bool
}

// UploadView contains parameters for the admin menu template
type UploadView struct {
	Items             []models.File
	ApiKeys           []models.ApiKey
	Url               string
	HotlinkUrl        string
	TimeNow           int64
	DefaultDownloads  int
	DefaultExpiry     int
	DefaultPassword   string
	IsAdminView       bool
	IsMainView        bool
	IsApiView         bool
	MaxFileSize       int
	IsLogoutAvailable bool
}

// Converts the globalConfig variable to an UploadView struct to pass the infos to
// the admin template
func (u *UploadView) convertGlobalConfig(isMainView bool) *UploadView {
	var result []models.File
	var resultApi []models.ApiKey
	settings := configuration.GetServerSettingsReadOnly()
	if isMainView {
		for _, element := range settings.Files {
			result = append(result, element)
		}
		sort.Slice(result[:], func(i, j int) bool {
			if result[i].ExpireAt == result[j].ExpireAt {
				return result[i].Id > result[j].Id
			}
			return result[i].ExpireAt > result[j].ExpireAt
		})
	} else {
		for _, element := range settings.ApiKeys {
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
	u.Url = settings.ServerUrl + "d?id="
	u.HotlinkUrl = settings.ServerUrl + "hotlink/"
	u.DefaultPassword = settings.DefaultPassword
	u.Items = result
	u.DefaultExpiry = settings.DefaultExpiry
	u.ApiKeys = resultApi
	u.DefaultDownloads = settings.DefaultDownloads
	u.TimeNow = time.Now().Unix()
	u.IsAdminView = true
	u.IsMainView = isMainView
	u.MaxFileSize = settings.MaxFileSizeMB
	u.IsLogoutAvailable = authentication.IsLogoutAvailable()
	configuration.ReleaseReadOnly()
	return u
}

// Handling of /upload
// If the user is authenticated, this parses the uploaded file from the Multipart Form and
// adds it to the system.
func uploadFile(w http.ResponseWriter, r *http.Request) {
	addNoCacheHeader(w)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if !isAuthenticatedOrRedirect(w, r, true) {
		return
	}
	err := fileupload.Process(w, r, true, webserverMaxMemory)
	responseError(w, err)
}

// Outputs an error in json format
func responseError(w http.ResponseWriter, err error) {
	if err != nil {
		_, _ = io.WriteString(w, "{\"Result\":\"error\",\"ErrorMessage\":\""+err.Error()+"\"}")
		helper.Check(err)
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

// Checks if the user is logged in as an admin. Redirects to login page if not authenticated
func isAuthenticatedOrRedirect(w http.ResponseWriter, r *http.Request, isUpload bool) bool {
	if authentication.IsAuthenticated(w, r) {
		return true
	}
	if isUpload {
		_, err := io.WriteString(w, "{\"Result\":\"error\",\"ErrorMessage\":\"Not authenticated\"}")
		helper.Check(err)
		return false
	}
	redirect(w, "login")
	return false
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
