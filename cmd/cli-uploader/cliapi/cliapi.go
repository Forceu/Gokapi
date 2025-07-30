package cliapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

var gokapiUrl string
var apiKey string
var e2ekey string

var EUnauthorised = errors.New("unauthorised")

func Init(url, key, end2endkey string) {
	gokapiUrl = strings.TrimSuffix(url, "/") + "/api"
	apiKey = key
	e2ekey = end2endkey

}

func GetVersion() (string, int, error) {
	result, err := getUrl(gokapiUrl + "/info/version")
	if err != nil {
		return "", 0, err
	}
	type expectedFormat struct {
		Version    string
		VersionInt int
	}
	var parsedResult expectedFormat
	err = json.Unmarshal([]byte(result), &parsedResult)
	if err != nil {
		return "", 0, err
	}
	return parsedResult.Version, parsedResult.VersionInt, nil
}

func GetConfig() (int, int, bool, error) {
	result, err := getUrl(gokapiUrl + "/info/config")
	if err != nil {
		return 0, 0, false, err
	}
	type expectedFormat struct {
		MaxFilesize               int
		MaxChunksize              int
		EndToEndEncryptionEnabled bool
	}
	var parsedResult expectedFormat
	err = json.Unmarshal([]byte(result), &parsedResult)
	if err != nil {
		return 0, 0, false, err
	}
	return parsedResult.MaxFilesize, parsedResult.MaxChunksize, parsedResult.EndToEndEncryptionEnabled, nil
}

func getUrl(url string) (string, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("apikey", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return "", EUnauthorised
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
