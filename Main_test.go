package main

/**
Unit testing for whole project. At the moment coverage is low.
*/

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/environment"
	"Gokapi/internal/webserver"
	"bytes"
	"html/template"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func loadConfig() {
	os.Setenv("GOKAPI_CONFIG_DIR", "test")
	os.Mkdir("test", 0777)
	os.WriteFile("test/config.json", configTestFile, 0777)
	configuration.Load()
}

func isEqualString(t *testing.T, got, want string) {
	if got != want {
		t.Errorf("Assertion failed, got: %s, want: %s.", got, want)
	}
}

func isEqualBool(t *testing.T, got, want bool) {
	if got != want {
		t.Errorf("Assertion failed, got: %t, want: %t.", got, want)
	}
}
func isEqualInt(t *testing.T, got, want int) {
	if got != want {
		t.Errorf("Assertion failed, got: %d, want: %d.", got, want)
	}
}

func TestConfigLoad(t *testing.T) {
	loadConfig()
	isEqualString(t, configuration.Environment.ConfigDir, "test")
	isEqualString(t, configuration.ServerSettings.Port, "127.0.0.1:53843")
	isEqualString(t, configuration.ServerSettings.AdminName, "test")
	isEqualString(t, configuration.ServerSettings.AdminPassword, "10340aece68aa4fb14507ae45b05506026f276cf")
	isEqualString(t, configuration.ServerSettings.ServerUrl, "http://127.0.0.1:53843/")
	isEqualString(t, configuration.ServerSettings.AdminPassword, "10340aece68aa4fb14507ae45b05506026f276cf")
	isEqualString(t, configuration.HashPassword("testtest", false), "10340aece68aa4fb14507ae45b05506026f276cf")
}

func testHttpPage(t *testing.T, testUrl, requiredContent string, isHtml bool, authCookie string) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", testUrl, nil)
	if err != nil {
		t.Fatal(err)
	}
	if authCookie != "" {
		req.Header.Set("Cookie", "session_token="+authCookie)
	}
	resp, err := client.Do(req)

	if resp.StatusCode != 200 {
		t.Errorf("Status %d != 200", resp.StatusCode)
	}
	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if isHtml && !bytes.Contains(bs, []byte("</html>")) {
		t.Error(testUrl + ": Incorrect response")
	}
	if !bytes.Contains(bs, []byte(requiredContent)) {
		t.Error(testUrl + ": Incorrect response. Got:\n" + string(bs))
	}
	resp.Body.Close()
}

func TestWebserver(t *testing.T) {
	loadConfig()
	go webserver.Start(&StaticFolderEmbedded, &TemplateFolderEmbedded, false)

	time.Sleep(1 * time.Second)
	// Index redirect
	testHttpPage(t, "http://localhost:53843/", "<html><head><meta http-equiv=\"Refresh\" content=\"0; URL=./index\"></head></html>", true, "")
	// CSS file
	testHttpPage(t, "http://localhost:53843/css/cover.css", ".btn-secondary:hover", false, "")
	// Login page
	testHttpPage(t, "http://localhost:53843/login", "id=\"uname_hidden\"", true, "")
	// Admin without auth
	testHttpPage(t, "http://localhost:53843/admin", "URL=./login\"", true, "")
	// Admin with auth
	testHttpPage(t, "http://localhost:53843/admin", "Downloads remaining", true, "GubBwU9KVuuRmOTjUvlKSIl9MyBLumuql9NAHFps8hc0UdumD8lD7mdPRuK01ouU9rZ5a4JiWUeB5aJ")
	// Admin with invalid auth
	testHttpPage(t, "http://localhost:53843/admin", "URL=./login\"", true, "invalid")
}

func TestEmbedFs(t *testing.T) {
	templates, err := template.ParseFS(TemplateFolderEmbedded, "web/templates/*.tmpl")
	if err != nil {
		t.Error("Unable to read templates")
	}
	if !strings.Contains(templates.DefinedTemplates(), "app_name") {
		t.Error("Unable to parse templates")
	}
	_, err = fs.Stat(StaticFolderEmbedded, "web/static/expired.png")
	if err != nil {
		t.Error("Static webdir incomplete")
	}
}

func TestEnvLoad(t *testing.T) {
	os.Setenv("GOKAPI_CONFIG_DIR", "test")
	os.Setenv("GOKAPI_CONFIG_FILE", "test2")
	os.Setenv("GOKAPI_LOCALHOST", "yes")
	os.Setenv("GOKAPI_LENGTH_ID", "7")
	env := environment.New()
	isEqualString(t, env.ConfigPath, "test/test2")
	isEqualString(t, env.WebserverLocalhost, environment.IsTrue)
	isEqualInt(t, env.LengthId, 7)
	os.Setenv("GOKAPI_LENGTH_ID", "3")
	env = environment.New()
	isEqualInt(t, env.LengthId, 5)

}

var configTestFile = []byte(`{"Port":"127.0.0.1:53843","AdminName":"test","AdminPassword":"10340aece68aa4fb14507ae45b05506026f276cf","ServerUrl":"http://127.0.0.1:53843/","DefaultDownloads":3,"DefaultExpiry":20,"DefaultPassword":"123","RedirectUrl":"https://test.com/","Sessions":{"GubBwU9KVuuRmOTjUvlKSIl9MyBLumuql9NAHFps8hc0UdumD8lD7mdPRuK01ouU9rZ5a4JiWUeB5aJ":{"RenewAt":2147483646,"ValidUntil":2147483646}},"Files":{},"Hotlinks":{},"ConfigVersion":4,"SaltAdmin":"LW6fW4Pjv8GtdWVLSZD66gYEev6NAaXxOVBw7C","SaltFiles":"lL5wMTtnVCn5TPbpRaSe4vAQodWW0hgk00WCZE","LengthId":0,"DataDir":"test/data"}`)
