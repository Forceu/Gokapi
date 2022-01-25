package oauth

import (
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/webserver/authentication"
	"testing"
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
