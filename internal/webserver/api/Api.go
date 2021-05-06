package api

import (
	"Gokapi/internal/configuration"
	"Gokapi/internal/helper"
	"Gokapi/internal/storage"
	"encoding/json"
	"net/http"
	"strings"
)

// Process parses the request and executes the API call or returns an error message to the sender
func Process(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("cache-control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	request := parseRequest(r)
	if !isAuthorised(w, request) {
		return
	}
	switch request.requestUrl {
	case "/files":
		list(w)
	case "/files/add":
		upload(w, request)
	case "/files/delete":
		deleteFile(w, request)
	default:
		sendError(w, http.StatusBadRequest, "Invalid request")
	}
}

func deleteFile(w http.ResponseWriter, request apiRequest) {
	ok := storage.DeleteFile(request.headerId)
	if ok {
		sendOk(w)
	} else {
		sendError(w, http.StatusBadRequest, "Invalid id provided.")
	}
}

func list(w http.ResponseWriter) {
	sendOk(w)
	settings := configuration.GetServerSettings()
	result, err := json.Marshal(settings.Files)
	configuration.Release()
	helper.Check(err)
	_, _ = w.Write(result)
}

func upload(w http.ResponseWriter, request apiRequest) {
	sendOk(w)
	// TODO
}

func isValidApiKey(key string) bool {
	if key == "" {
		return false
	}
	settings := configuration.GetServerSettings()
	savedKey := settings.ApiKeys[key]
	configuration.Release()
	return savedKey.Id != ""
}

func isAuthorised(w http.ResponseWriter, request apiRequest) bool {
	if true {
		return true
	}
	if isValidApiKey(request.apiKey) {
		return true
	}
	sendError(w, http.StatusUnauthorized, "Unauthorized")
	return false
}

func sendError(w http.ResponseWriter, errorInt int, errorMessage string) {
	w.WriteHeader(errorInt)
	_, _ = w.Write([]byte("{\"Result\":\"error\",\"ErrorMessage\":\"" + errorMessage + "\"}"))
}

func sendOk(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
}

type apiRequest struct {
	apiKey     string
	requestUrl string
	headerId   string
}

func parseRequest(r *http.Request) apiRequest {
	return apiRequest{
		apiKey:     r.Header.Get("apikey"),
		headerId:   r.Header.Get("id"),
		requestUrl: strings.Replace(r.URL.String(), "/api", "", 1),
	}
}
