package cliapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/forceu/gokapi/internal/encryption"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var gokapiUrl string
var apiKey string
var e2eKey string

const megaByte = 1024 * 1024

type header struct {
	Key   string
	Value string
}

var EUnauthorised = errors.New("unauthorised")
var EFileTooBig = errors.New("file too big")

func Init(url, key, end2endKey string) {
	gokapiUrl = strings.TrimSuffix(url, "/") + "/api"
	apiKey = key
	e2eKey = end2endKey

}

func GetVersion() (string, int, error) {
	result, err := getUrl(gokapiUrl+"/info/version", []header{})
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
	result, err := getUrl(gokapiUrl+"/info/config", []header{})
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

func getUrl(url string, headers []header) (string, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("apikey", apiKey)
	for _, addHeader := range headers {
		req.Header.Add(addHeader.Key, addHeader.Value)
	}

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

func UploadFile(f *os.File) (string, error) {
	maxSize, chunkSize, isE2e, err := GetConfig()
	if e2eKey == "" {
		isE2e = false
	}
	fi, err := f.Stat()
	if err != nil {
		return "", err
	}
	sizeBytes := fi.Size()
	if isE2e {
		sizeBytes = encryption.CalculateEncryptedFilesize(sizeBytes)
	}
	if sizeBytes > int64(maxSize*megaByte) {
		return "", EFileTooBig
	}
	uuid := helper.GenerateRandomString(30)

	for i := int64(0); i < sizeBytes; i = i + (int64(chunkSize) * megaByte) {
		err = uploadChunk(f, uuid, i, int64(chunkSize)*megaByte, sizeBytes)
		if err != nil {
			return "", err
		}
	}
	file, err := completeChunk(uuid, nameToBase64(f), sizeBytes)
	if err != nil {
		return "", err
	}
	return file.Id, nil
}

func nameToBase64(f *os.File) string {
	return "base64:" + base64.StdEncoding.EncodeToString([]byte(filepath.Base(f.Name())))
}

func uploadChunk(f io.Reader, uuid string, offset, chunkSize, filesize int64) error {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "uploadedfile")
	if err != nil {
		return err
	}
	buffer, err := io.ReadAll(io.LimitReader(f, chunkSize))
	if err != nil {
		return err
	}
	_, err = part.Write(buffer)
	if err != nil {
		return err
	}

	err = writer.WriteField("filesize", strconv.FormatInt(filesize, 10))
	if err != nil {
		return err
	}
	err = writer.WriteField("offset", strconv.FormatInt(offset, 10))
	if err != nil {
		return err
	}
	err = writer.WriteField("uuid", uuid)
	if err != nil {
		return err
	}
	err = writer.Close()
	if err != nil {
		return err
	}

	r, err := http.NewRequest("POST", gokapiUrl+"/chunk/add", body)
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", writer.FormDataContentType())
	r.Header.Set("apikey", apiKey)
	client := &http.Client{}
	resp, err := client.Do(r)

	if err != nil {
		return err
	}
	bodyContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	response := string(bodyContent)
	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to upload chunk: status code " + strconv.Itoa(resp.StatusCode) + ", response: " + response)
	}
	if response != "{\"result\":\"OK\"}" {
		return errors.New("failed to upload chunk: unexpected response: " + response)
	}
	return nil
}

func completeChunk(uid, filename string, filesize int64) (models.File, error) {
	type expectedFormat struct {
		FileInfo models.File `json:"FileInfo"`
	}
	result, err := getUrl(gokapiUrl+"/chunk/complete", []header{
		{"uuid", uid},
		{"filename", filename},
		{"filesize", strconv.FormatInt(filesize, 10)},
	})
	if err != nil {
		return models.File{}, err
	}
	var parsedResult expectedFormat
	err = json.Unmarshal([]byte(result), &parsedResult)
	if err != nil {
		return models.File{}, err
	}
	return parsedResult.FileInfo, nil
}
