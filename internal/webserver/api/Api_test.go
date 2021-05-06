package api

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/test"
	"Gokapi/internal/test/testconfiguration"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(true)
	configuration.Load()
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

var newKeyId string

func TestNewKey(t *testing.T) {
	newKeyId = NewKey()
	settings := configuration.GetServerSettings()
	configuration.Release()
	test.IsEqualString(t, settings.ApiKeys[newKeyId].FriendlyName, "Unnamed key")
}

func TestDeleteKey(t *testing.T) {
	settings := configuration.GetServerSettings()
	configuration.Release()
	test.IsEqualString(t, settings.ApiKeys[newKeyId].FriendlyName, "Unnamed key")
	result := DeleteKey(newKeyId)
	test.IsEqualString(t, settings.ApiKeys[newKeyId].FriendlyName, "")
	test.IsEqualBool(t, result, true)
	result = DeleteKey("invalid")
	test.IsEqualBool(t, result, false)
}

func TestIsValidApiKey(t *testing.T) {
	settings := configuration.GetServerSettings()
	configuration.Release()
	test.IsEqualBool(t, isValidKey("", false), false)
	test.IsEqualBool(t, isValidKey("invalid", false), false)
	test.IsEqualBool(t, isValidKey("validkey", false), true)
	test.IsEqualBool(t, settings.ApiKeys["validkey"].LastUsed == 0, true)
	test.IsEqualBool(t, isValidKey("validkey", true), true)
	test.IsEqualBool(t, settings.ApiKeys["validkey"].LastUsed == 0, false)
}

func TestProcess(t *testing.T) {
	w, r := getRecorder("GET", "/api/auth/friendlyname", nil, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "{\"Result\":\"error\",\"ErrorMessage\":\"Unauthorized\"}")
	w, r = getRecorder("GET", "/api/invalid", nil, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "Unauthorized")
	w, r = getRecorder("GET", "/api/invalid", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}})
	Process(w, r)
	test.ResponseBodyContains(t, w, "Invalid request")
	w, r = getRecorder("GET", "/api/invalid", []test.Cookie{{
		Name:  "session_token",
		Value: "validsession",
	}}, nil)
	Process(w, r)
	test.ResponseBodyContains(t, w, "Invalid request")
}

func TestChangeFriendlyName(t *testing.T) {
	settings := configuration.GetServerSettings()
	configuration.Release()
	w, r := getRecorder("GET", "/api/auth/friendlyname", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}})
	Process(w, r)
	test.ResponseBodyContains(t, w, "Invalid api key provided.")
	w, r = getRecorder("GET", "/api/auth/friendlyname", nil, []test.Header{{
		Name: "apikey", Value: "validkey"}, {
		Name: "apiKeyToModify", Value: "validkey"}})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	test.IsEqualString(t, settings.ApiKeys["validkey"].FriendlyName, "Unnamed key")
	w, r = getRecorder("GET", "/api/auth/friendlyname", nil, []test.Header{{
		Name: "apikey", Value: "validkey"}, {
		Name: "apiKeyToModify", Value: "validkey"}, {
		Name: "friendlyName", Value: "NewName"}})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	test.IsEqualString(t, settings.ApiKeys["validkey"].FriendlyName, "NewName")
	w = httptest.NewRecorder()
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
}

func TestDeleteFile(t *testing.T) {
	settings := configuration.GetServerSettings()
	configuration.Release()
	w, r := getRecorder("GET", "/api/files/delete", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}})
	Process(w, r)
	test.ResponseBodyContains(t, w, "Invalid id provided.")
	w, r = getRecorder("GET", "/api/files/delete", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}, {
		Name:  "id",
		Value: "invalid",
	},
	})
	Process(w, r)
	test.ResponseBodyContains(t, w, "Invalid id provided.")
	test.IsEqualString(t, settings.Files["jpLXGJKigM4hjtA6T6sN2"].Id, "jpLXGJKigM4hjtA6T6sN2")
	w, r = getRecorder("GET", "/api/files/delete", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}, {
		Name:  "id",
		Value: "jpLXGJKigM4hjtA6T6sN2",
	},
	})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	test.IsEqualString(t, settings.Files["jpLXGJKigM4hjtA6T6sN2"].Id, "")
}

func TestList(t *testing.T) {
	w, r := getRecorder("GET", "/api/files/list", nil, []test.Header{{
		Name:  "apikey",
		Value: "validkey",
	}})
	Process(w, r)
	test.IsEqualInt(t, w.Code, 200)
	test.ResponseBodyContains(t, w, "picture.jpg")
}

func getRecorder(method, target string, cookies []test.Cookie, headers []test.Header) (*httptest.ResponseRecorder, *http.Request) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(method, target, nil)
	if cookies != nil {
		for _, cookie := range cookies {
			r.AddCookie(&http.Cookie{
				Name:  cookie.Name,
				Value: cookie.Value,
				Path:  "/",
			})
		}
	}
	if headers != nil {
		for _, header := range headers {
			r.Header.Set(header.Name, header.Value)
		}
	}
	return w, r
}
