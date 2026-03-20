//go:build !integration && test

package webserver

import (
	"bufio"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage/processingstatus"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"github.com/forceu/gokapi/internal/webserver/authentication"
	"github.com/forceu/gokapi/internal/webserver/authentication/csrftoken"
	"github.com/forceu/gokapi/internal/webserver/ratelimiter"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(true)
	configuration.Load()
	configuration.ConnectDatabase()
	authentication.Init(configuration.Get().Authentication)
	go Start()
	time.Sleep(1 * time.Second)
	ratelimiter.SetUnitTestMode(true)
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestEmbedFs(t *testing.T) {
	funcMap := template.FuncMap{
		"newAdminButtonContext": newAdminButtonContext,
	}
	templates, err := template.New("").Funcs(funcMap).ParseFS(templateFolderEmbedded, "web/templates/*.tmpl")
	if err != nil {
		t.Error("Unable to read templates")
		return
	}
	if !strings.Contains(templates.DefinedTemplates(), "header") {
		t.Error("Unable to parse templates")
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
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/index",
		RequiredContent: []string{configuration.Get().RedirectUrl},
		IsHtml:          true,
	})
}
func TestStaticDirs(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/css/cover.css",
		RequiredContent: []string{".btn-secondary:hover"},
	})
}

func postValues(username, password, csrf string) []test.PostBody {
	return []test.PostBody{
		{Key: "username", Value: username},
		{Key: "password", Value: password},
		{Key: "csrf-token", Value: csrf},
	}
}

func cookieValue(cookies []*http.Cookie, name string) string {
	for _, c := range cookies {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

func TestLogin(t *testing.T) {
	const loginUrl = "http://localhost:53843/login"

	// GET /login shows the login form
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             loginUrl,
		IsHtml:          true,
		ResultCode:      http.StatusOK,
		RequiredContent: []string{"id=\"uname_hidden\""},
	})

	postConfig := test.HttpTestConfig{
		Url:             loginUrl,
		IsHtml:          true,
		Method:          "POST",
		ResultCode:      http.StatusOK,
		RequiredContent: []string{"id=\"uname_hidden\"", "Incorrect username or password"},
		ExcludedContent: []string{"URL=./admin"},
	}

	// POST with wrong username and password shows error
	postConfig.PostValues = postValues("invalid", "invalid", csrftoken.Generate())
	test.HttpPostRequest(t, postConfig)

	// POST with correct username but wrong password shows error
	postConfig.PostValues = postValues("test", "invalid", csrftoken.Generate())
	test.HttpPostRequest(t, postConfig)

	// POST with correct credentials but invalid CSRF token shows error
	postConfig.PostValues = postValues("test", "adminadmin", "invalid")
	postConfig.RequiredContent = []string{"id=\"uname_hidden\"", "The login page was open too long and expired. Please try again."}
	test.HttpPostRequest(t, postConfig)

	// GET /login with OAuth2 enabled redirects to oauth-login
	oauthConfig := configuration.Get()
	oauthConfig.Authentication.Method = models.AuthenticationOAuth2
	oauthConfig.Authentication.OAuthProvider = "http://test.com"
	oauthConfig.Authentication.OAuthClientSecret = "secret"
	oauthConfig.Authentication.OAuthClientId = "client"
	authentication.Init(oauthConfig.Authentication)
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:         loginUrl,
		ResultCode:  http.StatusTemporaryRedirect,
		RedirectUrl: "oauth-login",
	})
	configuration.Get().Authentication.Method = models.AuthenticationInternal
	authentication.Init(configuration.Get().Authentication)

	// POST with valid credentials returns a redirect to admin and sets a session cookie
	postConfig.RequiredContent = nil
	postConfig.ExcludedContent = nil
	postConfig.IsHtml = false
	postConfig.ResultCode = http.StatusTemporaryRedirect
	postConfig.RedirectUrl = "admin"
	postConfig.PostValues = postValues("test", "adminadmin", csrftoken.Generate())
	cookies := test.HttpPostRequest(t, postConfig)
	session := cookieValue(cookies, "session_token")
	test.IsNotEqualString(t, session, "")

	// Visiting /login with a valid session redirects to admin
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:         loginUrl,
		ResultCode:  http.StatusTemporaryRedirect,
		RedirectUrl: "admin",
		Cookies:     []test.Cookie{{Name: "session_token", Value: session}},
	})
}

func TestAdminNoAuth(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:         "http://localhost:53843/admin",
		RedirectUrl: "login",
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
		Url:         "http://localhost:53843/admin",
		RedirectUrl: "login",
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
		Url:         "http://localhost:53843/admin",
		RedirectUrl: "login",
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "invalid",
		}},
	})
}

func TestInvalidLink(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:                "http://localhost:53843/d?id=123",
		IgnoreRedirectParm: true,
		RedirectUrl:        "error",
	})
}

func TestError(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/error",
		RequiredContent: []string{"The link may have expired or the file has been downloaded too many times"},
		IsHtml:          true,
	})
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/error?e2e",
		RequiredContent: []string{"This file is encrypted, but no key was provided"},
		IsHtml:          true,
	})
}

func TestForgotPw(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/forgotpw",
		RequiredContent: []string{"--reconfigure"},
		IsHtml:          true,
	})
}

func TestLoginCorrect(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:         "http://localhost:53843/login",
		RedirectUrl: "admin",
		Method:      "POST",
		PostValues:  []test.PostBody{{"username", "test"}, {"password", "adminadmin"}, {"csrf-token", csrftoken.Generate()}},
	})
}

func TestLoginIncorrectPassword(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/login",
		RequiredContent: []string{"Incorrect username or password"},
		IsHtml:          true,
		Method:          "POST",
		PostValues:      []test.PostBody{{"username", "test"}, {"password", "incorrect"}, {"csrf-token", csrftoken.Generate()}},
	})
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/login",
		RequiredContent: []string{"The login page was open too long and expired. Please try again."},
		IsHtml:          true,
		Method:          "POST",
		PostValues:      []test.PostBody{{"username", "test"}, {"password", "incorrect"}, {"csrf-token", "incorrect"}},
	})
}
func TestLoginIncorrectUsername(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/login",
		RequiredContent: []string{"Incorrect username or password"},
		IsHtml:          true,
		Method:          "POST",
		PostValues:      []test.PostBody{{"username", "incorrect"}, {"password", "incorrect"}, {"csrf-token", csrftoken.Generate()}},
	})
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/login",
		RequiredContent: []string{"The login page was open too long and expired. Please try again."},
		IsHtml:          true,
		Method:          "POST",
		PostValues:      []test.PostBody{{"username", "incorrect"}, {"password", "incorrect"}, {"csrf-token", "incorrect"}},
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
		Url:         "http://localhost:53843/logout",
		RedirectUrl: "login",
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "logoutsession",
		}},
	})
	// Admin after logout
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:         "http://localhost:53843/admin",
		RedirectUrl: "login",
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
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/h/wjqlzpq2.jpg",
		RequiredContent: []string{"123"},
	})
	// Download expired hotlink
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/hotlink/PhSs6mFtf8O5YGlLMfNw9rYXx9XRNkzCnJZpQBi7inunv3Z4A.jpg",
		RequiredContent: []string{"The requested file has expired"},
	})
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/h/wjqlzpq2.jpg",
		RequiredContent: []string{"The requested file has expired"},
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
		Url:                "http://127.0.0.1:53843/d?id=Wzol7LyY2QVczXynJtVo",
		IgnoreRedirectParm: true,
		RedirectUrl:        "error",
	})
	// Download expired file
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:                "http://127.0.0.1:53843/downloadFile?id=Wzol7LyY2QVczXynJtVo",
		IgnoreRedirectParm: true,
		RedirectUrl:        "error",
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
		Url:         "http://127.0.0.1:53843/downloadFile?id=jpLXGJKigM4hjtA6T6sN",
		RedirectUrl: "d?id=jpLXGJKigM4hjtA6T6sN",
		Cookies:     []test.Cookie{{"pjpLXGJKigM4hjtA6T6sN", "invalid"}},
	})
}

func TestDownloadCorrectPassword(t *testing.T) {
	t.Parallel()
	// Submit download page correct password
	cookies := test.HttpPageResult(t, test.HttpTestConfig{
		Url:         "http://127.0.0.1:53843/d?id=jpLXGJKigM4hjtA6T6sN2",
		RedirectUrl: "d?id=jpLXGJKigM4hjtA6T6sN2",
		Method:      "POST",
		PostValues:  []test.PostBody{{"password", "123"}},
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

func TestPostUploadNoAuth(t *testing.T) {
	t.Parallel()
	test.HttpPostUploadRequest(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/uploadChunk",
		UploadFileName:  "test/fileupload.jpg",
		ResultCode:      http.StatusUnauthorized,
		UploadFieldName: "file",

		RequiredContent: []string{"{\"Result\":\"error\",\"ErrorMessage\":\"Not authenticated\"}"},
	})
}

func TestPostUpload(t *testing.T) {
	// Open the SSE connection
	req, err := http.NewRequest("GET", "http://127.0.0.1:53843/uploadStatus", nil)
	test.IsNil(t, err)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cookie", "session_token=validsession")

	resp, err := http.DefaultClient.Do(req)
	test.IsNil(t, err)
	defer resp.Body.Close()

	test.IsEqualInt(t, resp.StatusCode, http.StatusOK)
	scanner := bufio.NewScanner(resp.Body)

	test.HttpPostUploadRequest(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/uploadChunk",
		UploadFileName:  "test/fileupload.jpg",
		UploadFieldName: "file",
		PostValues: []test.PostBody{{
			Key:   "dztotalfilesize",
			Value: "50",
		}, {
			Key:   "dzchunkbyteoffset",
			Value: "0",
		}, {
			Key:   "dzuuid",
			Value: "eeng4ier3Taen7a",
		}},
		RequiredContent: []string{"{\"result\":\"OK\"}"},
		ExcludedContent: []string{"\"Id\":\"\"", "HotlinkId\":\"\"", "ErrorMessage"},
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "validsession",
		}},
	})
	go func() {
		time.Sleep(200 * time.Millisecond)
		test.HttpPostRequest(t, test.HttpTestConfig{
			Url: "http://127.0.0.1:53843/api/chunk/complete",
			Headers: []test.Header{
				{"apikey", "validkeyid7"},
				{"uuid", "eeng4ier3Taen7a"},
				{"filename", "fileupload.jpg"},
				{"filecontenttype", "test-content"},
				{"filesize", "50"},
				{"nonblocking", "true"},
			},
			RequiredContent: []string{"{\"result\":\"OK\"}"},
			Cookies: []test.Cookie{{
				Name:  "session_token",
				Value: "validsession",
			}},
		})
	}()

	var receivedStatus eventUploadStatus
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data:") {
			message := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			t.Log(message)
			err = json.Unmarshal([]byte(message), &receivedStatus)
			test.IsNil(t, err)
			if receivedStatus.UploadStatus == processingstatus.StatusFinished {
				break
			}
		}
	}
	test.IsEqualInt(t, receivedStatus.UploadStatus, processingstatus.StatusFinished)
	test.IsNotEmpty(t, receivedStatus.FileId)
	err = scanner.Err()
	test.IsNil(t, err)
	file, ok := database.GetMetaDataById(receivedStatus.FileId)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, file.Name, "fileupload.jpg")
}

// Originally declared in Sse, but should not be public
type eventUploadStatus struct {
	Event        string `json:"event"`
	ChunkId      string `json:"chunk_id"`
	FileId       string `json:"file_id"`
	ErrorMessage string `json:"error_message"`
	UploadStatus int    `json:"upload_status"`
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
		RedirectUrl:     "login",
		ResultCode:      http.StatusTemporaryRedirect,
		ExcludedContent: []string{"Click on the API key name to give it a new name."},
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "invalid",
		}},
	})
}

func TestProcessApi(t *testing.T) {
	// Not authorised
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/api/files/list",
		RequiredContent: []string{`{"Result":"error","ErrorMessage":"Unauthorized","ErrorCode":2}`},
		ExcludedContent: []string{"smallfile2"},
		ResultCode:      401,
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "invalid",
		}},
	})
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/api/files/list",
		RequiredContent: []string{`{"Result":"error","ErrorMessage":"Unauthorized","ErrorCode":2}`},
		ExcludedContent: []string{"smallfile2"},
		ResultCode:      401,
		Headers:         []test.Header{{"apikey", "invalid"}},
	})

	// Valid session does not grant API access
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://127.0.0.1:53843/api/files/list",
		RequiredContent: []string{`{"Result":"error","ErrorMessage":"Unauthorized","ErrorCode":2}`},
		ExcludedContent: []string{"smallfile2"},
		ResultCode:      401,
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
		Url:         "http://localhost:53843/admin",
		RedirectUrl: "login",
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "invalid",
		}},
	})
	configuration.Get().Authentication.Method = models.AuthenticationDisabled
	authentication.Init(configuration.Get().Authentication)
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:             "http://localhost:53843/admin",
		RequiredContent: []string{"Downloads remaining"},
		IsHtml:          true,
		Cookies: []test.Cookie{{
			Name:  "session_token",
			Value: "invalid",
		}},
	})
	configuration.Get().Authentication.Method = models.AuthenticationInternal
	authentication.Init(configuration.Get().Authentication)
}

func TestResponseError(t *testing.T) {
	w, _ := test.GetRecorder("GET", "/", nil, nil, nil)
	responseError(w, errors.New("testerror"))
	test.IsEqualInt(t, w.Result().StatusCode, 400)
	test.ResponseBodyContains(t, w, "testerror")
}

func TestServeWasmDownloader(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:    "http://localhost:53843/main.wasm",
		IsHtml: false,
	})
}
func TestServeWasmE2E(t *testing.T) {
	t.Parallel()
	test.HttpPageResult(t, test.HttpTestConfig{
		Url:    "http://localhost:53843/e2e.wasm",
		IsHtml: false,
	})
}
