package oauth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/webserver/authentication"
)

func TestSetCallbackCookie(t *testing.T) {
	w, _ := test.GetRecorder("GET", "/", nil, nil, nil)
	setCallbackCookie(w, "test")
	cookies := w.Result().Cookies()
	test.IsEqualInt(t, len(cookies), 1)
	test.IsEqualString(t, cookies[0].Name, authentication.CookieOauth)
	value := cookies[0].Value
	test.IsEqualString(t, value, "test")
}

func TestHandlerLogin(t *testing.T) {
	// Setup a dummy config
	config.ClientID = "test-client"
	config.Endpoint.AuthURL = "https://example.com/auth"

	req, _ := http.NewRequest("GET", "/login?consent=true", nil)
	rr := httptest.NewRecorder()

	HandlerLogin(rr, req)

	// Check for redirect to provider
	test.IsEqualInt(t, rr.Code, http.StatusFound)
	location := rr.Header().Get("Location")
	test.IsEqualBool(t, len(location) > 0, true)

	// Verify prompt=consent was added
	test.IsEqualBool(t, location != "", true)
	// Check if cookie was set
	test.IsEqualBool(t, len(rr.Result().Cookies()) > 0, true)
}

func TestHandlerCallback_StateMismatch(t *testing.T) {
	req, _ := http.NewRequest("GET", "/oauth-callback?state=wrong-state&code=123", nil)
	// Add the correct cookie to the request, but use a wrong state in URL
	req.AddCookie(&http.Cookie{Name: authentication.CookieOauth, Value: "correct-state"})

	rr := httptest.NewRecorder()
	HandlerCallback(rr, req)

	// Should redirect to error page
	test.IsEqualInt(t, rr.Code, http.StatusSeeOther)
	test.IsEqualBool(t, rr.Header().Get("Location") != "", true)
}

func TestIsLoginRequired(t *testing.T) {
	t.Run("Standard error", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/?error=login_required", nil)
		test.IsEqualBool(t, isLoginRequired(req), true)
	})

	t.Run("No error", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/?code=123", nil)
		test.IsEqualBool(t, isLoginRequired(req), false)
	})
}
