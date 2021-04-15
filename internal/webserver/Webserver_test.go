package webserver

import (
	"Gokapi/internal/configuration"
	testconfiguration "Gokapi/internal/test"
	"Gokapi/pkg/test"
	"html/template"
	"io/fs"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(true)
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestEmbedFs(t *testing.T) {
	templates, err := template.ParseFS(templateFolderEmbedded, "web/templates/*.tmpl")
	if err != nil {
		t.Error("Unable to read templates")
	}
	if !strings.Contains(templates.DefinedTemplates(), "app_name") {
		t.Error("Unable to parse templates")
	}
	_, err = fs.Stat(staticFolderEmbedded, "web/static/expired.png")
	if err != nil {
		t.Error("Static webdir incomplete")
	}
}

func TestWebserverEmbedFs(t *testing.T) {
	configuration.Load()
	go Start()

	time.Sleep(1 * time.Second)
	// Index redirect
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/",
		RequiredContent: "<html><head><meta http-equiv=\"Refresh\" content=\"0; URL=./index\"></head></html>",
		IsHtml:          true,
	})
	// Index file
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/index",
		RequiredContent: configuration.ServerSettings.RedirectUrl,
		IsHtml:          true,
	})
	// CSS file
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/css/cover.css",
		RequiredContent: ".btn-secondary:hover",
	})
	// Login page
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/login",
		RequiredContent: "id=\"uname_hidden\"",
		IsHtml:          true,
	})
	// Admin without auth
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/admin",
		RequiredContent: "URL=./login\"",
		IsHtml:          true,
	})
	// Admin with auth
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/admin",
		RequiredContent: "Downloads remaining",
		IsHtml:          true,
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "validsession",
		}},
	})
	// Admin with expired session
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/admin",
		RequiredContent: "URL=./login\"",
		IsHtml:          true,
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "expiredsession",
		}},
	})
	// Admin with invalid auth
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/admin",
		RequiredContent: "URL=./login\"",
		IsHtml:          true,
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "invalid",
		}},
	})
	// Invalid link
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/d?id=123",
		RequiredContent: "URL=./error\"",
		IsHtml:          true,
	})
	// Error
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/error",
		RequiredContent: "this file cannot be found",
		IsHtml:          true,
	})
	// Forgot pw
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/forgotpw",
		RequiredContent: "--reset-pw",
		IsHtml:          true,
	})
	// Login correct
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/login",
		RequiredContent: "URL=./admin\"",
		IsHtml:          true,
		Method:          "POST",
		PostValues:      []test.PostBody{{"username", "test"}, {"password", "testtest"}},
	})
	// Login incorrect
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/login",
		RequiredContent: "Incorrect username or password",
		IsHtml:          true,
		Method:          "POST",
		PostValues:      []test.PostBody{{"username", "test"}, {"password", "incorrect"}},
	})
	// Login incorrect
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/login",
		RequiredContent: "Incorrect username or password",
		IsHtml:          true,
		Method:          "POST",
		PostValues:      []test.PostBody{{"username", "incorrect"}, {"password", "incorrect"}},
	})
	// Download hotlink
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/hotlink/PhSs6mFtf8O5YGlLMfNw9rYXx9XRNkzCnJZpQBi7inunv3Z4A.jpg",
		RequiredContent: "123",
	})
	// Download expired hotlink
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/hotlink/PhSs6mFtf8O5YGlLMfNw9rYXx9XRNkzCnJZpQBi7inunv3Z4A.jpg",
		RequiredContent: "Created with GIMP",
	})
	// Show download page no password
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/d?id=Wzol7LyY2QVczXynJtVo",
		IsHtml:          true,
		RequiredContent: "smallfile2",
	})
	// Download file no password
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/downloadFile?id=Wzol7LyY2QVczXynJtVo",
		RequiredContent: "789",
	})
	// Show download page expired file
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/d?id=Wzol7LyY2QVczXynJtVo",
		IsHtml:          true,
		RequiredContent: "URL=./error\"",
	})
	// Download expired file
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/downloadFile?id=Wzol7LyY2QVczXynJtVo",
		IsHtml:          true,
		RequiredContent: "URL=./error\"",
	})
	// Show download page password
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/d?id=jpLXGJKigM4hjtA6T6sN",
		IsHtml:          true,
		RequiredContent: "Password required",
	})
	// Show download page incorrect password
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/d?id=jpLXGJKigM4hjtA6T6sN",
		IsHtml:          true,
		RequiredContent: "Incorrect password!",
		Method:          "POST",
		PostValues:      []test.PostBody{{"password", "incorrect"}},
	})
	// Submit download page correct password
	cookies := test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/d?id=jpLXGJKigM4hjtA6T6sN",
		IsHtml:          true,
		RequiredContent: "URL=./d?id=jpLXGJKigM4hjtA6T6sN",
		Method:          "POST",
		PostValues:      []test.PostBody{{"password", "123"}},
	})
	pwCookie := ""
	for _, cookie := range cookies {
		if (*cookie).Name == "pjpLXGJKigM4hjtA6T6sN" {
			pwCookie = (*cookie).Value
			break
		}
	}
	if pwCookie == "" {
		t.Error("Cookie not set")
	}
	// Show download page correct password
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/d?id=jpLXGJKigM4hjtA6T6sN",
		IsHtml:          true,
		RequiredContent: "smallfile",
		Cookies:         []test.Cookie{{"pjpLXGJKigM4hjtA6T6sN", pwCookie}},
	})
	// Show download page incorrect password cookie
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/d?id=jpLXGJKigM4hjtA6T6sN",
		IsHtml:          true,
		RequiredContent: "Password required",
		Cookies:         []test.Cookie{{"pjpLXGJKigM4hjtA6T6sN", "invalid"}},
	})
	// Download incorrect password
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/downloadFile?id=jpLXGJKigM4hjtA6T6sN",
		IsHtml:          true,
		RequiredContent: "URL=./d?id=jpLXGJKigM4hjtA6T6sN",
		Cookies:         []test.Cookie{{"pjpLXGJKigM4hjtA6T6sN", "invalid"}},
	})
	// Download correct password
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/downloadFile?id=jpLXGJKigM4hjtA6T6sN",
		RequiredContent: "456",
		Cookies:         []test.Cookie{{"pjpLXGJKigM4hjtA6T6sN", pwCookie}},
	})
	// Delete file non-auth
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/delete?id=e4TjE7CokWK0giiLNxDL",
		IsHtml:          true,
		RequiredContent: "URL=./login",
	})
	// Delete file authorised
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/delete?id=e4TjE7CokWK0giiLNxDL",
		IsHtml:          true,
		RequiredContent: "URL=./admin",
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "validsession",
		}},
	})
	// Post upload unauthorized
	test.HttpPostRequest(t, "http://127.0.0.1:53843/upload", "test/fileupload.jpg", "file", "{\"Result\":\"error\",\"ErrorMessage\":\"Not authenticated\"}", []test.Cookie{})
	// Post upload authorized
	test.HttpPostRequest(t, "http://127.0.0.1:53843/upload", "test/fileupload.jpg", "file", "fileupload.jpg", []test.Cookie{{
		Name:  "session_token",
		Value: "validsession",
	}})
	// Logout
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/logout",
		RequiredContent: "URL=./login\"",
		IsHtml:          true,
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "validsession",
		}},
	})
	// Admin after logout
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/admin",
		RequiredContent: "URL=./login\"",
		IsHtml:          true,
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "validsession",
		}},
	})
}
