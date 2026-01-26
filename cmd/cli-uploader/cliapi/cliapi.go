package cliapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/forceu/gokapi/cmd/cli-uploader/cliflags"
	"github.com/forceu/gokapi/internal/encryption"
	"github.com/forceu/gokapi/internal/encryption/end2end"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/schollz/progressbar/v3"
)

var gokapiUrl string
var apiKey string
var e2eKey []byte

const megaByte = 1024 * 1024

type header struct {
	Key   string
	Value string
}

// ErrInvalidRequest is returned when the API returns a 400
var ErrInvalidRequest = errors.New("400 Bad Request")

// ErrUnauthorised is returned when the API returns a 401
var ErrUnauthorised = errors.New("unauthorised")

// ErrNotFound is returned when the API returns a 404
var ErrNotFound = errors.New("404 Not Found")

// ErrFileTooBig is returned when the file size exceeds the allowed limit
var ErrFileTooBig = errors.New("file too big")

// ErrE2eKeyIncorrect is returned when the e2e key is incorrect
var ErrE2eKeyIncorrect = errors.New("e2e key incorrect")

// Init initialises the API client with the given url and key.
// The key is used for authentication.
// The end2endKey is used for end-to-end encryption.
func Init(url, key string, end2endKey []byte) {
	gokapiUrl = strings.TrimSuffix(url, "/") + "/api"
	apiKey = key
	e2eKey = end2endKey

}

// GetVersion returns the version of the Gokapi server
func GetVersion() (string, int, error) {
	result, err := getUrl(gokapiUrl+"/info/version", []header{}, false)
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
	if parsedResult.Version == "" {
		parsedResult.Version = "unknown"
	}
	return parsedResult.Version, parsedResult.VersionInt, nil
}

// GetConfig returns the upload configuration of the Gokapi server
func GetConfig() (int, int, bool, error) {
	result, err := getUrl(gokapiUrl+"/info/config", []header{}, false)
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

func getUrl(url string, headers []header, longTimeout bool) (string, error) {
	timeout := 30 * time.Second
	if longTimeout {
		timeout = 30 * time.Minute
	}
	client := &http.Client{
		Timeout: timeout,
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

	switch resp.StatusCode {
	case 400:
		return "", ErrInvalidRequest
	case 401:
		return "", ErrUnauthorised
	case 404:
		return "", ErrNotFound
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// UploadFile uploads a file to the Gokapi server
func UploadFile(uploadParams cliflags.FlagConfig) (models.FileApiOutput, error) {
	var progressBar *progressbar.ProgressBar
	file, err := os.OpenFile(uploadParams.File, os.O_RDONLY, 0664)
	if err != nil {
		fmt.Println("ERROR: Could not open file to upload")
		fmt.Println(err)
		os.Exit(4)
	}
	defer file.Close()
	maxSize, chunkSize, isE2e, err := GetConfig()
	if err != nil {
		return models.FileApiOutput{}, err
	}
	// TODO check for 401

	if len(e2eKey) == 0 || !isE2e || uploadParams.DisableE2e {
		isE2e = false
	}
	fileStat, err := file.Stat()
	if err != nil {
		return models.FileApiOutput{}, err
	}
	sizeBytes := fileStat.Size()
	realSize := fileStat.Size()
	if isE2e {
		sizeBytes = encryption.CalculateEncryptedFilesize(sizeBytes)
	}
	if sizeBytes > int64(maxSize)*megaByte {
		return models.FileApiOutput{}, ErrFileTooBig
	}
	uuid := helper.GenerateRandomString(30)

	if !uploadParams.JsonOutput {
		progressBar = progressbar.DefaultBytes(-1, "uploading")
	}

	if isE2e {
		cipher, err := encryption.GetRandomCipher()
		if err != nil {
			return models.FileApiOutput{}, err
		}
		stream, err := encryption.GetEncryptReader(cipher, file)
		if err != nil {
			return models.FileApiOutput{}, err
		}
		for i := int64(0); i < sizeBytes; i = i + (int64(chunkSize) * megaByte) {
			err = uploadChunk(stream, uuid, i, int64(chunkSize)*megaByte, sizeBytes, progressBar)
			if err != nil {
				return models.FileApiOutput{}, err
			}
		}
		metaData, err := completeChunk(uuid, "Encrypted File", sizeBytes, realSize, true, uploadParams, progressBar)
		if err != nil {
			return models.FileApiOutput{}, err
		}

		e2eFile := models.E2EFile{
			Uuid:     uuid,
			Id:       metaData.Id,
			Filename: getFileName(file, uploadParams),
			Cipher:   cipher,
		}
		err = addE2EFileInfo(e2eFile)
		if err != nil {
			return models.FileApiOutput{}, err
		}
		hashContent, err := getHashContent(e2eFile)
		metaData.UrlDownload = metaData.UrlDownload + "#" + hashContent
		metaData.Name = getFileName(file, uploadParams)
		return metaData, err
	}

	for i := int64(0); i < sizeBytes; i = i + (int64(chunkSize) * megaByte) {
		err = uploadChunk(file, uuid, i, int64(chunkSize)*megaByte, sizeBytes, progressBar)
		if err != nil {
			return models.FileApiOutput{}, err
		}
	}
	metaData, err := completeChunk(uuid, nameToBase64(file, uploadParams), sizeBytes, realSize, false, uploadParams, progressBar)
	if err != nil {
		return models.FileApiOutput{}, err
	}
	return metaData, nil
}

// DownloadFile downloads a file from the Gokapi server
func DownloadFile(downloadParams cliflags.FlagConfig) error {
	var progressBar *progressbar.ProgressBar

	info, err := getFileInfo(downloadParams.DownloadId)
	if err != nil {
		fmt.Println("ERROR: Could not get file info or file does not exist")
		return err
	}
	if downloadParams.OutputPath == "" {
		downloadParams.OutputPath = "."
	}
	if downloadParams.FileName == "" {
		downloadParams.FileName = info.Name
	}
	filename := downloadParams.OutputPath + "/" + downloadParams.FileName
	exists, err := helper.FileExists(filename)
	if err != nil {
		fmt.Println("ERROR: Could not check if file already exists")
		return err
	}
	if exists {
		fmt.Println("ERROR: File already exists, please specify a different filename")
		os.Exit(1)
	}
	if !helper.FolderExists(downloadParams.OutputPath) {
		err = os.Mkdir(downloadParams.OutputPath, 0770)
		if err != nil {
			fmt.Println("ERROR: Could not create output directory")
			return err
		}
	}
	helper.CreateDir(downloadParams.OutputPath)
	file, err := os.Create(downloadParams.OutputPath + "/" + downloadParams.FileName)
	defer file.Close()
	if err != nil {
		fmt.Println("ERROR: Could not create new file")
		return err
	}

	if !downloadParams.JsonOutput {
		progressBar = progressbar.DefaultBytes(info.SizeBytes, "Downloading")
	}

	req, err := http.NewRequest("GET", gokapiUrl+"/files/download/"+downloadParams.DownloadId, nil)
	if err != nil {
		return err
	}
	req.Header.Add("apikey", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("ERROR: Could not download file: Status code " + strconv.Itoa(resp.StatusCode))
		os.Exit(4)
	}

	if !downloadParams.JsonOutput {
		_, err = io.Copy(file, io.TeeReader(resp.Body, progressBar))
	} else {
		_, err = io.Copy(file, resp.Body)
	}

	if err != nil {
		fmt.Println("ERROR: Could not download file")
		return err
	}
	if downloadParams.RemoveRemote {
		err = deleteRemoteFile(downloadParams.DownloadId)
		if err != nil {
			return err
		}
	}
	if !downloadParams.JsonOutput {
		fmt.Println("File downloaded successfully")
	} else {
		fmt.Println("{\"result\":\"OK\"}")
	}
	return nil
}

func nameToBase64(f *os.File, uploadParams cliflags.FlagConfig) string {
	return "base64:" + base64.StdEncoding.EncodeToString([]byte(getFileName(f, uploadParams)))
}

func getFileName(f *os.File, uploadParams cliflags.FlagConfig) string {
	if uploadParams.FileName != "" {
		return uploadParams.FileName
	}
	return filepath.Base(f.Name())
}

func getFileInfo(id string) (models.FileApiOutput, error) {
	result, err := getUrl(gokapiUrl+"/files/list/"+id, []header{}, false)
	if err != nil {
		return models.FileApiOutput{}, err
	}
	var parsedResult models.FileApiOutput
	err = json.Unmarshal([]byte(result), &parsedResult)
	if err != nil {
		return models.FileApiOutput{}, err
	}
	return parsedResult, nil
}

func deleteRemoteFile(id string) error {
	_, err := getUrl(gokapiUrl+"/files/delete", []header{{"id", id}}, false)
	return err
}

func uploadChunk(f io.Reader, uuid string, offset, chunkSize, filesize int64, progressBar *progressbar.ProgressBar) error {
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

	var bodyReader io.Reader
	if progressBar != nil {
		bodyReader = io.TeeReader(body, progressBar)
	} else {
		bodyReader = body
	}

	r, err := http.NewRequest("POST", gokapiUrl+"/chunk/add", bodyReader)
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

func completeChunk(uid, filename string, filesize, realsize int64, useE2e bool, uploadParams cliflags.FlagConfig, progressBar *progressbar.ProgressBar) (models.FileApiOutput, error) {
	type expectedFormat struct {
		FileInfo models.FileApiOutput `json:"FileInfo"`
	}
	if progressBar != nil {
		_ = progressBar.Finish()
	}
	if !uploadParams.JsonOutput {
		fmt.Println("Finalising...")
	}
	result, err := getUrl(gokapiUrl+"/chunk/complete", []header{
		{"uuid", uid},
		{"filename", filename},
		{"filesize", strconv.FormatInt(filesize, 10)},
		{"realsize", strconv.FormatInt(realsize, 10)},
		{"isE2E", strconv.FormatBool(useE2e)},
		{"allowedDownloads", strconv.Itoa(uploadParams.ExpiryDownloads)},
		{"expiryDays", strconv.Itoa(uploadParams.ExpiryDays)},
		{"password", uploadParams.Password},
		{"contenttype", "application/octet-stream"},
	}, true)
	if err != nil {
		return models.FileApiOutput{}, err
	}
	var parsedResult expectedFormat
	err = json.Unmarshal([]byte(result), &parsedResult)
	if err != nil {
		return models.FileApiOutput{}, err
	}
	return parsedResult.FileInfo, nil
}

// GetE2eInfo returns the end-to-end encryption information of the Gokapi server for this user
func GetE2eInfo() (models.E2EInfoPlainText, error) {
	var result models.E2EInfoEncrypted
	var fileInfo models.E2EInfoPlainText
	resultJson, err := getUrl(gokapiUrl+"/e2e/get", []header{}, false)
	if err != nil {
		return models.E2EInfoPlainText{}, err
	}
	err = json.Unmarshal([]byte(resultJson), &result)
	if err != nil {
		return models.E2EInfoPlainText{}, err
	}
	fileInfo, err = end2end.DecryptData(result, e2eKey)
	if err != nil {
		return models.E2EInfoPlainText{}, ErrE2eKeyIncorrect
	}
	return fileInfo, nil
}

func addE2EFileInfo(file models.E2EFile) error {
	infoPlain, err := GetE2eInfo()
	if err != nil {
		return err
	}
	infoPlain.Files = append(infoPlain.Files, file)
	output, err := end2end.EncryptData(infoPlain.Files, e2eKey)
	if err != nil {
		return err
	}
	return setE2eInfo(output)
}

func setE2eInfo(input models.E2EInfoEncrypted) error {
	outputJson, err := json.Marshal(input)
	if err != nil {
		return err
	}
	content := base64.StdEncoding.EncodeToString(outputJson)

	apiURL := gokapiUrl + "/e2e/set"

	bodyData := map[string]string{
		"content": content,
	}
	bodyBytes, err := json.Marshal(bodyData)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	_ = resp.Body.Close()
	return nil
}

func getHashContent(input models.E2EFile) (string, error) {
	output, err := json.Marshal(models.E2EHashContent{
		Filename: input.Filename,
		Cipher:   base64.StdEncoding.EncodeToString(input.Cipher),
	})
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(output), nil
}
