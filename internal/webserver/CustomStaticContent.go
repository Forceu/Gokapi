package webserver

import (
	"bufio"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/NYTimes/gziphandler"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/webserver/favicon"
)

const pathCustomFolder = "custom/"
const pathCustomCss = pathCustomFolder + "custom.css"
const pathCustomPublicJs = pathCustomFolder + "public.js"
const pathCustomAdminJs = pathCustomFolder + "admin.js"
const pathCustomVersioning = pathCustomFolder + "version.txt"
const pathCustomFavicon = pathCustomFolder + "favicon.png"

type customStatic struct {
	Version            string
	CustomFolderExists bool
	UseCustomCss       bool
	UseCustomPublicJs  bool
	UseCustomAdminJs   bool
}

func loadCustomCssJsInfo(webserverDir fs.FS) {
	customStaticInfo = customStatic{}
	folderExists := helper.FolderExists(pathCustomFolder)
	customStaticInfo.CustomFolderExists = folderExists
	favicon.Init(pathCustomFavicon, webserverDir)
	if !folderExists {
		return
	}
	customStaticInfo.Version = strconv.Itoa(readCustomStaticVersion())
	customStaticInfo.UseCustomCss = helper.FileExists(pathCustomCss)
	customStaticInfo.UseCustomPublicJs = helper.FileExists(pathCustomPublicJs)
	customStaticInfo.UseCustomAdminJs = helper.FileExists(pathCustomAdminJs)
}

func addMuxForCustomContent(mux *http.ServeMux) {
	if !customStaticInfo.CustomFolderExists {
		return
	}
	fmt.Println("Serving custom static content")
	// Serve the user-created "custom" folder to /custom
	mux.Handle("/custom/", http.StripPrefix("/custom/", http.FileServer(http.Dir(pathCustomFolder))))
	// Allow versioning to prevent caching old versions
	if customStaticInfo.UseCustomCss {
		mux.Handle("/custom/custom.v"+customStaticInfo.Version+".css", gziphandler.GzipHandler(http.HandlerFunc(serveCustomCss)))
	}
	if customStaticInfo.UseCustomPublicJs {
		mux.Handle("/custom/public.v"+customStaticInfo.Version+".js", gziphandler.GzipHandler(http.HandlerFunc(serveCustomPublicJs)))
	}
	if customStaticInfo.UseCustomAdminJs {
		mux.Handle("/custom/admin.v"+customStaticInfo.Version+".js", gziphandler.GzipHandler(http.HandlerFunc(serveCustomAdminJs)))
	}
}

func serveCustomCss(w http.ResponseWriter, r *http.Request) {
	serveCustomFile(pathCustomCss, w, r)
}
func serveCustomPublicJs(w http.ResponseWriter, r *http.Request) {
	serveCustomFile(pathCustomPublicJs, w, r)
}
func serveCustomAdminJs(w http.ResponseWriter, r *http.Request) {
	serveCustomFile(pathCustomAdminJs, w, r)
}

func serveCustomFile(filePath string, w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Cache-Control", "public, max-age=100800") // 2 days
	http.ServeFile(w, r, filePath)
}

func readCustomStaticVersion() int {
	if !helper.FileExists(pathCustomVersioning) {
		return 0
	}
	file, err := os.Open(pathCustomVersioning)
	if err != nil {
		fmt.Println(err)
		return 0
	}
	defer file.Close()
	sc := bufio.NewScanner(file)
	if !sc.Scan() {
		return 0
	}
	line := strings.TrimSpace(sc.Text())
	version, err := strconv.Atoi(line)
	if err != nil {
		fmt.Println("Content of " + pathCustomVersioning + " must be numerical")
	}
	return version
}
