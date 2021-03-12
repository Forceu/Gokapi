package main

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed static
var staticFolderEmbedded embed.FS

//go:embed templates
var templateFolderEmbedded embed.FS

var templateFolder *template.Template

func startWebserver() {
	webserverDir, _ := fs.Sub(staticFolderEmbedded, "static")
	if folderExists("static") {
		http.Handle("/", http.FileServer(http.Dir("static")))
	} else {
		http.Handle("/", http.FileServer(http.FS(webserverDir)))
	}
	http.HandleFunc("/index", showIndex)
	http.HandleFunc("/d", showDownload)
	http.HandleFunc("/error", showError)
	http.HandleFunc("/login", showLogin)
	http.HandleFunc("/logout", doLogout)
	http.HandleFunc("/admin", showAdminMenu)
	http.HandleFunc("/upload", uploadFile)
	http.HandleFunc("/delete", deleteFile)
	http.HandleFunc("/downloadFile", downloadFile)
	http.HandleFunc("/forgotpw", forgotPassword)
	fmt.Println("Webserver started on " + globalConfig.Port )
	fmt.Println("Webserver can be accessed on " + globalConfig.ServerUrl + "admin")
	log.Fatal(http.ListenAndServe(globalConfig.Port, nil))
}

func initTemplates() {
	var err error
	if folderExists("templates") {
		templateFolder, err = template.ParseGlob("templates/*.tmpl")
		check(err)
	} else {
		templateFolder, err = template.ParseFS(templateFolderEmbedded, "templates/*.tmpl")
		check(err)
	}
}

func redirect(w http.ResponseWriter, url string) {
	_, _ = fmt.Fprint(w, "<head><meta http-equiv=\"Refresh\" content=\"0; URL=./"+url+"\"></head>")
}

func doLogout(w http.ResponseWriter, r *http.Request) {
	logoutSession(w, r)
	redirect(w, "login")
}

func showIndex(w http.ResponseWriter, r *http.Request) {
	err := templateFolder.ExecuteTemplate(w, "index", globalConfig.RedirectUrl)
	check(err)
}

func showError(w http.ResponseWriter, r *http.Request) {
	err := templateFolder.ExecuteTemplate(w, "error", nil)
	check(err)
}

func forgotPassword(w http.ResponseWriter, r *http.Request) {
	err := templateFolder.ExecuteTemplate(w, "forgotpw", nil)
	check(err)
}

func showLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	check(err)
	user := r.Form.Get("username")
	pw := r.Form.Get("password")
	failedLogin := false
	if pw != "" && user != "" {
		if strings.ToLower(user) == strings.ToLower(globalConfig.AdminName) && hashPassword(pw) == globalConfig.AdminPassword {
			createSession(w)
			redirect(w, "admin")
			return
		} else {
			time.Sleep(3 * time.Second)
			failedLogin = true
		}
	}
	err = templateFolder.ExecuteTemplate(w, "login", LoginView{
		IsFailedLogin: failedLogin,
		User:          user,
	})
	check(err)
}

type LoginView struct {
	IsFailedLogin bool
	User          string
}

func showDownload(w http.ResponseWriter, r *http.Request) {
	keyId := queryUrl(w, r, "error")
	if keyId == "" {
		return
	}
	file := globalConfig.Files[keyId]
	if file.ExpireAt < time.Now().Unix() || file.DownloadsRemaining < 1 {
		redirect(w, "error")
		return
	}
	err := templateFolder.ExecuteTemplate(w, "download", DownloadView{
		Name: file.Name,
		Size: file.Size,
		Id:   file.Id,
	})
	check(err)
}

func deleteFile(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(w, r, false) {
		return
	}
	keyId := queryUrl(w, r, "admin")
	if keyId == "" {
		return
	}
	item := globalConfig.Files[keyId]
	item.ExpireAt = 0
	globalConfig.Files[keyId] = item
	cleanUpOldFiles(false)
	redirect(w, "admin")
}

func queryUrl(w http.ResponseWriter, r *http.Request, redirectUrl string) string {
	keys, ok := r.URL.Query()["id"]
	if !ok || len(keys[0]) < 15 {
		redirect(w, redirectUrl)
		return ""
	}
	return keys[0]
}

func showAdminMenu(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(w, r, false) {
		return
	}
	err := templateFolder.ExecuteTemplate(w, "admin", (&UploadView{}).convertGlobalConfig())
	check(err)
}

type DownloadView struct {
	Name string
	Size string
	Id   string
}

type UploadView struct {
	Items            []FileList
	Url              string
	TimeNow          int64
	DefaultDownloads int
	DefaultExpiry    int
}

func (u *UploadView) convertGlobalConfig() *UploadView {
	var result []FileList
	for _, element := range globalConfig.Files {
		result = append(result, element)
	}
	sort.Slice(result[:], func(i, j int) bool {
		return result[i].ExpireAt > result[j].ExpireAt
	})
	u.Url = globalConfig.ServerUrl + "d?id="
	u.Items = result
	u.DefaultExpiry = globalConfig.DefaultExpiry
	u.DefaultDownloads = globalConfig.DefaultDownloads
	u.TimeNow = time.Now().Unix()
	return u
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(w, r, true) {
		return
	}
	err := r.ParseMultipartForm(20 * 1024 * 1024)
	responseError(w, err)
	allowedDownloads := r.Form.Get("allowedDownloads")
	expiryDays := r.Form.Get("expiryDays")
	allowedDownloadsInt, err := strconv.Atoi(allowedDownloads)
	if err != nil {
		allowedDownloadsInt = globalConfig.DefaultDownloads
	}
	expiryDaysInt, err := strconv.Atoi(expiryDays)
	if err != nil {
		expiryDaysInt = globalConfig.DefaultExpiry
	}
	globalConfig.DefaultExpiry = expiryDaysInt
	globalConfig.DefaultDownloads = allowedDownloadsInt
	file, handler, err := r.FormFile("file")
	responseError(w, err)
	result, err := createNewFile(&file, handler, time.Now().Add(time.Duration(expiryDaysInt)*time.Hour*24).Unix(), allowedDownloadsInt)
	responseError(w, err)
	defer file.Close()
	_, err = fmt.Fprint(w, result.toJsonResult())
	check(err)
}
func responseError(w http.ResponseWriter, err error) {
	if err != nil {
		fmt.Fprint(w, "{\"Result\":\"error\",\"ErrorMessage\":\""+err.Error()+"\"}")
		panic(err)
	}
}

func downloadFile(w http.ResponseWriter, r *http.Request) {
	keyId := queryUrl(w, r, "error")
	if keyId == "" {
		return
	}
	savedFile := globalConfig.Files[keyId]
	if savedFile.DownloadsRemaining == 0 || savedFile.ExpireAt < time.Now().Unix() || !fileExists("data/"+savedFile.SHA256) {
		redirect(w, "error")
		return
	}
	savedFile.DownloadsRemaining = savedFile.DownloadsRemaining - 1
	globalConfig.Files[keyId] = savedFile
	saveConfig()

	w.Header().Set("Content-Disposition", "attachment; filename=\""+savedFile.Name+"\"")
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	file, err := os.OpenFile("data/"+savedFile.SHA256, os.O_RDONLY, 0644)
	defer file.Close()
	check(err)
	_, err = io.Copy(w, file)
	check(err)
}

func isAuthenticated(w http.ResponseWriter, r *http.Request, isUpload bool) bool {
	if isValidSession(w, r) {
		return true
	}
	if isUpload {
		_, err := fmt.Fprint(w, "{\"Result\":\"error\",\"ErrorMessage\":\"Not authenticated\"}")
		check(err)
	} else {
		redirect(w, "login")
	}
	return false
}
