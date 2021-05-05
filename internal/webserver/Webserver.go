package webserver

/**
Handling of webserver and requests / uploads
*/

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/helper"
	"Gokapi/internal/models"
	"Gokapi/internal/storage"
	"Gokapi/internal/webserver/sessionmanager"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
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
	webserverPort          string
	webserverExtUrl        string
	webserverRedirectUrl   string
	webserverAdminName     string
	webserverAdminPassword string
)

// Start the webserver on the port set in the config
func Start() {
	initLocalVariables()
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
	http.HandleFunc("/index", showIndex)
	http.HandleFunc("/d", showDownload)
	http.HandleFunc("/hotlink/", showHotlink)
	http.HandleFunc("/error", showError)
	http.HandleFunc("/login", showLogin)
	http.HandleFunc("/logout", doLogout)
	http.HandleFunc("/admin", showAdminMenu)
	http.HandleFunc("/upload", uploadFile)
	http.HandleFunc("/delete", deleteFile)
	http.HandleFunc("/downloadFile", downloadFile)
	http.HandleFunc("/forgotpw", forgotPassword)
	fmt.Println("Binding webserver to " + webserverPort)
	fmt.Println("Webserver can be accessed at " + webserverExtUrl + "admin")
	srv := &http.Server{
		Addr:         webserverPort,
		ReadTimeout:  timeOutWebserver,
		WriteTimeout: timeOutWebserver,
	}
	log.Fatal(srv.ListenAndServe())
}

func initLocalVariables() {
	settings := configuration.GetServerSettings()
	webserverPort = settings.Port
	webserverExtUrl = settings.ServerUrl
	webserverRedirectUrl = settings.RedirectUrl
	webserverAdminName = settings.AdminName
	webserverAdminPassword = settings.AdminPassword
	configuration.Release()
}

// Initialises the templateFolder variable by scanning through all the templates.
// If a folder "templates" exists in the main directory, it is used.
// Otherwise templateFolderEmbedded will be used.
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
	_, _ = fmt.Fprint(w, "<html><head><meta http-equiv=\"Refresh\" content=\"0; URL=./"+url+"\"></head></html>")
}

// Handling of /logout
func doLogout(w http.ResponseWriter, r *http.Request) {
	sessionmanager.LogoutSession(w, r)
	redirect(w, "login")
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

// Handling of /forgotpw
func forgotPassword(w http.ResponseWriter, r *http.Request) {
	err := templateFolder.ExecuteTemplate(w, "forgotpw", genericView{})
	helper.Check(err)
}

// Handling of /login
// Shows a login form. If username / pw combo is incorrect, client needs to wait for three seconds.
// If correct, a new session is created and the user is redirected to the admin menu
func showLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	helper.Check(err)
	user := r.Form.Get("username")
	pw := r.Form.Get("password")
	failedLogin := false
	if pw != "" && user != "" {
		if strings.ToLower(user) == strings.ToLower(webserverAdminName) && configuration.HashPassword(pw, false) == webserverAdminPassword {
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
	if !isAuthenticated(w, r, false) {
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
	if !isAuthenticated(w, r, false) {
		return
	}
	err := templateFolder.ExecuteTemplate(w, "admin", (&UploadView{}).convertGlobalConfig())
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
	Items            []models.File
	Url              string
	HotlinkUrl       string
	TimeNow          int64
	DefaultDownloads int
	DefaultExpiry    int
	DefaultPassword  string
	IsAdminView      bool
	IsMainView       bool
	IsApiView        bool
}

// Converts the globalConfig variable to an UploadView struct to pass the infos to
// the admin template
func (u *UploadView) convertGlobalConfig() *UploadView {
	var result []models.File
	settings := configuration.GetServerSettings()
	for _, element := range settings.Files {
		result = append(result, element)
	}
	sort.Slice(result[:], func(i, j int) bool {
		if result[i].ExpireAt == result[j].ExpireAt {
			return result[i].Id > result[j].Id
		}
		return result[i].ExpireAt > result[j].ExpireAt
	})
	u.Url = settings.ServerUrl + "d?id="
	u.HotlinkUrl = settings.ServerUrl + "hotlink/"
	u.DefaultPassword = settings.DefaultPassword
	u.Items = result
	u.DefaultExpiry = settings.DefaultExpiry
	u.DefaultDownloads = settings.DefaultDownloads
	u.TimeNow = time.Now().Unix()
	u.IsAdminView = true
	u.IsMainView = true
	configuration.Release()
	return u
}

// Handling of /upload
// If the user is authenticated, this parses the uploaded file from the Multipart Form and
// adds it to the system.
func uploadFile(w http.ResponseWriter, r *http.Request) {
	addNoCacheHeader(w)
	if !isAuthenticated(w, r, true) {
		return
	}
	err := r.ParseMultipartForm(20 * 1024 * 1024)
	responseError(w, err)
	allowedDownloads := r.Form.Get("allowedDownloads")
	expiryDays := r.Form.Get("expiryDays")
	password := r.Form.Get("password")
	allowedDownloadsInt, err := strconv.Atoi(allowedDownloads)
	settings := configuration.GetServerSettings()
	if err != nil {
		allowedDownloadsInt = settings.DefaultDownloads
	}
	expiryDaysInt, err := strconv.Atoi(expiryDays)
	if err != nil {
		expiryDaysInt = settings.DefaultExpiry
	}
	settings.DefaultExpiry = expiryDaysInt
	settings.DefaultDownloads = allowedDownloadsInt
	settings.DefaultPassword = password
	configuration.Release()
	file, header, err := r.FormFile("file")
	responseError(w, err)
	result, err := storage.NewFile(file, header, time.Now().Add(time.Duration(expiryDaysInt)*time.Hour*24).Unix(), allowedDownloadsInt, password)
	responseError(w, err)
	defer file.Close()
	_, err = fmt.Fprint(w, result.ToJsonResult(webserverExtUrl))
	helper.Check(err)
}

// Outputs an error in json format
func responseError(w http.ResponseWriter, err error) {
	if err != nil {
		fmt.Fprint(w, "{\"Result\":\"error\",\"ErrorMessage\":\""+err.Error()+"\"}")
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

// Checks if the user is logged in as an admin
func isAuthenticated(w http.ResponseWriter, r *http.Request, isUpload bool) bool {
	if sessionmanager.IsValidSession(w, r) {
		return true
	}
	if isUpload {
		_, err := fmt.Fprint(w, "{\"Result\":\"error\",\"ErrorMessage\":\"Not authenticated\"}")
		helper.Check(err)
	} else {
		redirect(w, "login")
	}
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
