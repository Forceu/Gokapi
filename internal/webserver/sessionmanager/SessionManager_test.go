package sessionmanager

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/test"
	"Gokapi/internal/test/testconfiguration"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var newSession string

func TestMain(m *testing.M) {
	testconfiguration.Create(true)
	configuration.Load()
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func getRecorder(cookies []test.Cookie) (*httptest.ResponseRecorder, *http.Request) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	if cookies != nil {
		for _, cookie := range cookies {
			r.AddCookie(&http.Cookie{
				Name:  cookie.Name,
				Value: cookie.Value,
				Path:  "/",
			})
		}
	}
	return w, r
}

func TestIsValidSession(t *testing.T) {
	test.IsEqualBool(t, IsValidSession(getRecorder(nil)), false)
	test.IsEqualBool(t, IsValidSession(getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: "invalid"},
	})), false)
	test.IsEqualBool(t, IsValidSession(getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: "expiredsession"},
	})), false)
	test.IsEqualBool(t, IsValidSession(getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: "validsession"},
	})), true)
	w, r := getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: "needsRenewal"},
	})
	test.IsEqualBool(t, IsValidSession(w, r), true)
	cookies := w.Result().Cookies()
	test.IsEqualInt(t, len(cookies), 1)
	test.IsEqualString(t, cookies[0].Name, "session_token")
	session := cookies[0].Value
	test.IsEqualInt(t, len(session), 60)
	test.IsNotEqualString(t, session, "needsRenewal")
}

func TestCreateSession(t *testing.T) {
	w, _ := getRecorder(nil)
	CreateSession(w, false)
	cookies := w.Result().Cookies()
	test.IsEqualInt(t, len(cookies), 1)
	test.IsEqualString(t, cookies[0].Name, "session_token")
	newSession = cookies[0].Value
	test.IsEqualInt(t, len(newSession), 60)
	test.IsEqualBool(t, IsValidSession(getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: newSession},
	})), true)
}

func TestLogoutSession(t *testing.T) {
	test.IsEqualBool(t, IsValidSession(getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: newSession},
	})), true)
	LogoutSession(getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: newSession},
	}))
	test.IsEqualBool(t, IsValidSession(getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: newSession},
	})), false)
}
