//go:build !integration && test
// +build !integration,test

package webserver

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/test"
	"Gokapi/internal/test/testconfiguration"
	"Gokapi/internal/webserver/authentication"
	"errors"
	"html/template"
	"io"
	"io/fs"
	"os"
	"strings"
	"testing"
	"time"
)

// Please note that if this test is run with go test -race, it will fail as there is a bug in html/template that
// causes data race. It will be fixed in Go 1.17, see https://github.com/golang/go/issues/39807

func TestMain(m *testing.M) {
	testconfiguration.Create(true)
	configuration.Load()
	go Start()
	time.Sleep(1 * time.Second)
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

func TestIndexRedirect(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/",
		RequiredContent: []string{"<html><head><meta http-equiv=\"Refresh\" content=\"0; URL=./index\"></head></html>"},
		IsHtml:          true,
	})
}
func TestIndexFile(t *testing.T) {
	t.Parallel()
	settings := configuration.GetServerSettings()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/index",
		RequiredContent: []string{settings.RedirectUrl},
		IsHtml:          true,
	})
	configuration.Release()
}
func TestStaticDirs(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/css/cover.css",
		RequiredContent: []string{".btn-secondary:hover"},
	})
}
func TestLogin(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/login",
		RequiredContent: []string{"id=\"uname_hidden\""},
		IsHtml:          true,
	})
}
func TestAdminNoAuth(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/admin",
		RequiredContent: []string{"URL=./login\""},
		IsHtml:          true,
	})
}
func TestAdminAuth(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/admin",
		RequiredContent: []string{"Downloads remaining"},
		IsHtml:          true,
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "validsession",
		}},
	})
}
func TestAdminExpiredAuth(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/admin",
		RequiredContent: []string{"URL=./login\""},
		IsHtml:          true,
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "expiredsession",
		}},
	})
}

func TestAdminRenewalAuth(t *testing.T) {
	t.Parallel()
	cookies := test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/admin",
		RequiredContent: []string{"Downloads remaining"},
		IsHtml:          true,
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "needsRenewal",
		}},
	})
	sessionCookie := "needsRenewal"
	for _, cookie := range cookies {
		if (*cookie).Name == "session_token" {
			sessionCookie = (*cookie).Value
			break
		}
	}
	if sessionCookie == "needsRenewal" {
		t.Error("Session not renewed")
	}
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/admin",
		RequiredContent: []string{"Downloads remaining"},
		IsHtml:          true,
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: sessionCookie,
		}},
	})
}

func TestAdminInvalidAuth(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/admin",
		RequiredContent: []string{"URL=./login\""},
		IsHtml:          true,
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "invalid",
		}},
	})
}

func TestInvalidLink(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/d?id=123",
		RequiredContent: []string{"URL=./error\""},
		IsHtml:          true,
	})
}
func TestError(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/error",
		RequiredContent: []string{"this file cannot be found"},
		IsHtml:          true,
	})
}
func TestForgotPw(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/forgotpw",
		RequiredContent: []string{"--reset-pw"},
		IsHtml:          true,
	})
}
func TestLoginCorrect(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/login",
		RequiredContent: []string{"URL=./admin\""},
		IsHtml:          true,
		Method:          "POST",
		PostValues:      []test.PostBody{{"username", "test"}, {"password", "testtest"}},
	})
}

func TestLoginIncorrectPassword(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/login",
		RequiredContent: []string{"Incorrect username or password"},
		IsHtml:          true,
		Method:          "POST",
		PostValues:      []test.PostBody{{"username", "test"}, {"password", "incorrect"}},
	})
}
func TestLoginIncorrectUsername(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/login",
		RequiredContent: []string{"Incorrect username or password"},
		IsHtml:          true,
		Method:          "POST",
		PostValues:      []test.PostBody{{"username", "incorrect"}, {"password", "incorrect"}},
	})
}

func TestLogout(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/admin",
		RequiredContent: []string{"Downloads remaining"},
		IsHtml:          true,
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "logoutsession",
		}},
	})
	// Logout
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/logout",
		RequiredContent: []string{"URL=./login\""},
		IsHtml:          true,
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "logoutsession",
		}},
	})
	// Admin after logout
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/admin",
		RequiredContent: []string{"URL=./login\""},
		IsHtml:          true,
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "logoutsession",
		}},
	})
}

func TestDownloadHotlink(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/hotlink/PhSs6mFtf8O5YGlLMfNw9rYXx9XRNkzCnJZpQBi7inunv3Z4A.jpg",
		RequiredContent: []string{"123"},
	})
	// Download expired hotlink
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/hotlink/PhSs6mFtf8O5YGlLMfNw9rYXx9XRNkzCnJZpQBi7inunv3Z4A.jpg",
		RequiredContent: []string{"Created with GIMP"},
	})
}

func TestDownloadNoPassword(t *testing.T) {
	t.Parallel()
	// Show download page
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/d?id=Wzol7LyY2QVczXynJtVo",
		IsHtml:          true,
		RequiredContent: []string{"smallfile2"},
	})
	// Download
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/downloadFile?id=Wzol7LyY2QVczXynJtVo",
		RequiredContent: []string{"789"},
	})
	// Show download page expired file
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/d?id=Wzol7LyY2QVczXynJtVo",
		IsHtml:          true,
		RequiredContent: []string{"URL=./error\""},
	})
	// Download expired file
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/downloadFile?id=Wzol7LyY2QVczXynJtVo",
		IsHtml:          true,
		RequiredContent: []string{"URL=./error\""},
	})
}

func TestDownloadPagePassword(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/d?id=jpLXGJKigM4hjtA6T6sN",
		IsHtml:          true,
		RequiredContent: []string{"Password required"},
	})
}
func TestDownloadPageIncorrectPassword(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/d?id=jpLXGJKigM4hjtA6T6sN",
		IsHtml:          true,
		RequiredContent: []string{"Incorrect password!"},
		Method:          "POST",
		PostValues:      []test.PostBody{{"password", "incorrect"}},
	})
}

func TestDownloadIncorrectPasswordCookie(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/d?id=jpLXGJKigM4hjtA6T6sN",
		IsHtml:          true,
		RequiredContent: []string{"Password required"},
		Cookies:         []test.Cookie{{"pjpLXGJKigM4hjtA6T6sN", "invalid"}},
	})
}

func TestDownloadIncorrectPassword(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/downloadFile?id=jpLXGJKigM4hjtA6T6sN",
		IsHtml:          true,
		RequiredContent: []string{"URL=./d?id=jpLXGJKigM4hjtA6T6sN"},
		Cookies:         []test.Cookie{{"pjpLXGJKigM4hjtA6T6sN", "invalid"}},
	})
}

func TestDownloadCorrectPassword(t *testing.T) {
	t.Parallel()
	// Submit download page correct password
	cookies := test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/d?id=jpLXGJKigM4hjtA6T6sN2",
		IsHtml:          true,
		RequiredContent: []string{"URL=./d?id=jpLXGJKigM4hjtA6T6sN2"},
		Method:          "POST",
		PostValues:      []test.PostBody{{"password", "123"}},
	})
	pwCookie := ""
	for _, cookie := range cookies {
		if (*cookie).Name == "pjpLXGJKigM4hjtA6T6sN2" {
			pwCookie = (*cookie).Value
			break
		}
	}
	if pwCookie == "" {
		t.Error("Cookie not set")
	}
	// Show download page correct password
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/d?id=jpLXGJKigM4hjtA6T6sN2",
		IsHtml:          true,
		RequiredContent: []string{"smallfile"},
		Cookies:         []test.Cookie{{"pjpLXGJKigM4hjtA6T6sN2", pwCookie}},
	})
	// Download correct password
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/downloadFile?id=jpLXGJKigM4hjtA6T6sN2",
		RequiredContent: []string{"456"},
		Cookies:         []test.Cookie{{"pjpLXGJKigM4hjtA6T6sN2", pwCookie}},
	})
}

func TestDeleteFileNonAuth(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/delete?id=e4TjE7CokWK0giiLNxDL",
		IsHtml:          true,
		RequiredContent: []string{"URL=./login"},
	})
}

func TestDeleteFileInvalidKey(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/delete",
		IsHtml:          true,
		RequiredContent: []string{"URL=./admin"},
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "validsession",
		}},
	})
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/delete?id=",
		IsHtml:          true,
		RequiredContent: []string{"URL=./admin"},
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "validsession",
		}},
	})
}

func TestPostUploadNoAuth(t *testing.T) {
	t.Parallel()
	test.HttpPostRequest(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/upload",
		UploadFileName:  "test/fileupload.jpg",
		UploadFieldName: "file",
		RequiredContent: []string{"{\"Result\":\"error\",\"ErrorMessage\":\"Not authenticated\"}"},
	})
}

func TestPostUpload(t *testing.T) {
	test.HttpPostRequest(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/upload",
		UploadFileName:  "test/fileupload.jpg",
		UploadFieldName: "file",
		RequiredContent: []string{"{\"Result\":\"OK\"", "\"Name\":\"fileupload.jpg\"", "\"SHA256\":\"a9993e364706816aba3e25717850c26c9cd0d89d\"", "DownloadsRemaining\":3"},
		ExcludedContent: []string{"\"Id\":\"\"", "HotlinkId\":\"\""},
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "validsession",
		}},
	})
}

func TestDeleteFile(t *testing.T) {
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/delete?id=e4TjE7CokWK0giiLNxDL",
		IsHtml:          true,
		RequiredContent: []string{"URL=./admin"},
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "validsession",
		}},
	})
}

func TestApiPageAuthorized(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/apiKeys",
		IsHtml:          true,
		RequiredContent: []string{"Click on the API key name to give it a new name."},
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "validsession",
		}},
	})
}
func TestApiPageNotAuthorized(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/apiKeys",
		IsHtml:          true,
		RequiredContent: []string{"URL=./login"},
		ExcludedContent: []string{"Click on the API key name to give it a new name."},
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "invalid",
		}},
	})
}

func TestNewApiKey(t *testing.T) {
	// Authorised
	settings := configuration.GetServerSettings()
	amountKeys := len(settings.ApiKeys)
	configuration.Release()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/apiNew",
		IsHtml:          true,
		RequiredContent: []string{"URL=./apiKeys"},
		ExcludedContent: []string{"URL=./login"},
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "validsession",
		}},
	})
	settings = configuration.GetServerSettings()
	amountKeysAfter := len(settings.ApiKeys)
	configuration.Release()
	test.IsEqualInt(t, amountKeysAfter, amountKeys+1)
	test.IsEqualInt(t, amountKeysAfter, 5)

	// Not authorised
	amountKeys++
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/apiNew",
		IsHtml:          true,
		RequiredContent: []string{"URL=./login"},
		ExcludedContent: []string{"URL=./apiKeys"},
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "invalid",
		}},
	})
	settings = configuration.GetServerSettings()
	amountKeysAfter = len(settings.ApiKeys)
	configuration.Release()
	test.IsEqualInt(t, amountKeysAfter, amountKeys)
	test.IsEqualInt(t, amountKeysAfter, 5)
}

func TestDeleteApiKey(t *testing.T) {
	// Not authorised
	settings := configuration.GetServerSettings()
	amountKeys := len(settings.ApiKeys)
	configuration.Release()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/apiDelete?id=jiREglQJW0bOqJakfjdVfe8T1EM8n8",
		IsHtml:          true,
		RequiredContent: []string{"URL=./login"},
		ExcludedContent: []string{"URL=./apiKeys"},
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "invalid",
		}},
	})
	settings = configuration.GetServerSettings()
	amountKeysAfter := len(settings.ApiKeys)
	test.IsEqualString(t, settings.ApiKeys["jiREglQJW0bOqJakfjdVfe8T1EM8n8"].Id, "jiREglQJW0bOqJakfjdVfe8T1EM8n8")
	configuration.Release()
	test.IsEqualInt(t, amountKeysAfter, amountKeys)
	test.IsEqualInt(t, amountKeysAfter, 5)

	// Authorised
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/apiDelete?id=jiREglQJW0bOqJakfjdVfe8T1EM8n8",
		IsHtml:          true,
		RequiredContent: []string{"URL=./apiKeys"},
		ExcludedContent: []string{"URL=./login"},
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "validsession",
		}},
	})
	settings = configuration.GetServerSettings()
	amountKeysAfter = len(settings.ApiKeys)
	test.IsEmpty(t, settings.ApiKeys["jiREglQJW0bOqJakfjdVfe8T1EM8n8"].Id)
	configuration.Release()
	test.IsEqualInt(t, amountKeysAfter, amountKeys-1)
	test.IsEqualInt(t, amountKeysAfter, 4)
}

func TestProcessApi(t *testing.T) {
	// Not authorised
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/api/files/list",
		RequiredContent: []string{"{\"Result\":\"error\",\"ErrorMessage\":\"Unauthorized\"}"},
		ExcludedContent: []string{"smallfile2"},
		ResultCode:      401,
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "invalid",
		}},
	})
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/api/files/list",
		RequiredContent: []string{"{\"Result\":\"error\",\"ErrorMessage\":\"Unauthorized\"}"},
		ExcludedContent: []string{"smallfile2"},
		ResultCode:      401,
		Headers:         []test.Header{{"apikey", "invalid"}},
	})

	// Authorised
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/api/files/list",
		RequiredContent: []string{"smallfile2"},
		ExcludedContent: []string{"Unauthorized"},
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "validsession",
		}},
	})
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/api/files/list",
		RequiredContent: []string{"smallfile2"},
		ExcludedContent: []string{"Unauthorized"},
		Headers:         []test.Header{{"apikey", "validkey"}},
	})
}

func TestDisableLogin(t *testing.T) {
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/admin",
		RequiredContent: []string{"URL=./login\""},
		IsHtml:          true,
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "invalid",
		}},
	})
	settings := configuration.GetServerSettings()
	settings.Authentication.Method = authentication.Disabled
	authentication.Init(settings.Authentication)
	configuration.Release()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/admin",
		RequiredContent: []string{"Downloads remaining"},
		IsHtml:          true,
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "invalid",
		}},
	})
	settings = configuration.GetServerSettings()
	settings.Authentication.Method = authentication.Internal
	authentication.Init(settings.Authentication)
	configuration.Release()
}

func TestResponseError(t *testing.T) {
	w, _ := test.GetRecorder("GET", "/", nil, nil, nil)
	err := errors.New("testerror")
	defer test.ExpectPanic(t)
	responseError(w, err)
	output, err := io.ReadAll(w.Result().Body)
	test.IsNil(t, err)
	test.IsEqualString(t, string(output), "{\"Result\":\"error\",\"ErrorMessage\":\""+err.Error()+"\"}")
}
