package sessionmanager

import (
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var newSession string

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	configuration.Load()
	configuration.ConnectDatabase()
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func getRecorder(cookies []test.Cookie) (*httptest.ResponseRecorder, *http.Request, bool, int) {
	w, r := test.GetRecorder("GET", "/", cookies, nil, nil)
	return w, r, false, 1
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
	w, r, _, _ := getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: "needsRenewal"},
	})
	test.IsEqualBool(t, IsValidSession(w, r, false, 1), true)
	cookies := w.Result().Cookies()
	test.IsEqualInt(t, len(cookies), 1)
	test.IsEqualString(t, cookies[0].Name, "session_token")
	session := cookies[0].Value
	test.IsEqualInt(t, len(session), 60)
	test.IsNotEqualString(t, session, "needsRenewal")
}

func TestCreateSession(t *testing.T) {
	w, _, _, _ := getRecorder(nil)
	CreateSession(w, false, 1)
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
	w, r, _, _ := getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: newSession},
	})
	LogoutSession(w, r)
	test.IsEqualBool(t, IsValidSession(getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: newSession},
	})), false)
}
