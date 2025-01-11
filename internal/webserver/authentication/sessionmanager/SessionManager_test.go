package sessionmanager

import (
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
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
	user, ok := IsValidSession(getRecorder(nil))
	test.IsEqualBool(t, ok, false)
	user, ok = IsValidSession(getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: "invalid"},
	}))
	test.IsEqualBool(t, ok, false)
	user, ok = IsValidSession(getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: ""},
	}))
	test.IsEqualBool(t, ok, false)
	user, ok = IsValidSession(getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: "expiredsession"},
	}))
	test.IsEqualBool(t, ok, false)
	user, ok = IsValidSession(getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: "validsession"},
	}))
	test.IsEqualBool(t, ok, true)
	_, ok = IsValidSession(getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: "validSessionInvalidUser"},
	}))
	test.IsEqualBool(t, ok, false)
	test.IsEqualInt(t, user.Id, 7)
	w, r, _, _ := getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: "needsRenewal"},
	})
	user, ok = IsValidSession(w, r, false, 1)
	cookies := w.Result().Cookies()
	test.IsEqualInt(t, len(cookies), 1)
	test.IsEqualString(t, cookies[0].Name, "session_token")
	session := cookies[0].Value
	test.IsEqualInt(t, len(session), 60)
	test.IsNotEqualString(t, session, "needsRenewal")
}

func TestCreateSession(t *testing.T) {
	w, _, _, _ := getRecorder(nil)
	CreateSession(w, false, 1, 5)
	cookies := w.Result().Cookies()
	test.IsEqualInt(t, len(cookies), 1)
	test.IsEqualString(t, cookies[0].Name, "session_token")
	newSession = cookies[0].Value
	test.IsEqualInt(t, len(newSession), 60)

	user, ok := IsValidSession(getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: newSession},
	}))
	test.IsEqualBool(t, ok, true)
	test.IsEqualInt(t, user.Id, 5)

	w, _, _, _ = getRecorder(nil)
	CreateSession(w, true, 20, 50)
	cookies = w.Result().Cookies()
	newOauthSession := cookies[0].Value

	var session models.Session
	session, ok = database.GetSession(newOauthSession)
	test.IsEqualBool(t, ok, true)
	isEqual := time.Now().Add(20*time.Hour).Unix()-session.ValidUntil < 10 &&
		time.Now().Add(20*time.Hour).Unix()-session.ValidUntil > -1
	test.IsEqualBool(t, isEqual, true)
}

func TestLogoutSession(t *testing.T) {
	user, ok := IsValidSession(getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: newSession},
	}))
	test.IsEqualBool(t, ok, true)
	test.IsEqualInt(t, user.Id, 5)
	w, r, _, _ := getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: newSession},
	})
	LogoutSession(w, r)
	_, ok = IsValidSession(getRecorder([]test.Cookie{{
		Name:  "session_token",
		Value: newSession},
	}))
	test.IsEqualBool(t, ok, false)
}
