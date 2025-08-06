package cliapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/forceu/gokapi/cmd/cli-uploader/cliflags"
	"github.com/forceu/gokapi/internal/encryption"
	"github.com/forceu/gokapi/internal/encryption/end2end"
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
var e2eKey []byte

const megaByte = 1024 * 1024

type header struct {
	Key   string
	Value string
}

var EUnauthorised = errors.New("unauthorised")
var EFileTooBig = errors.New("file too big")
var EE2eKeyIncorrect = errors.New("e2e key incorrect")

func Init(url, key string, end2endKey []byte) {
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

func UploadFile(uploadParams cliflags.UploadConfig) (models.FileApiOutput, error) {
	file, err := os.OpenFile(uploadParams.File, os.O_RDONLY, 0664)
	if err != nil {
		fmt.Println("ERROR: Could not open file to upload")
		fmt.Println(err)
		os.Exit(4)
	}
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
		return models.FileApiOutput{}, EFileTooBig
	}
	uuid := helper.GenerateRandomString(30)

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
			err = uploadChunk(stream, uuid, i, int64(chunkSize)*megaByte, sizeBytes)
			if err != nil {
				return models.FileApiOutput{}, err
			}
		}
		metaData, err := completeChunk(uuid, "Encrypted File", sizeBytes, realSize, true, uploadParams)
		if err != nil {
			return models.FileApiOutput{}, err
		}

		e2eFile := models.E2EFile{
			Uuid:     uuid,
			Id:       metaData.Id,
			Filename: getFileName(file),
			Cipher:   cipher,
		}
		err = addE2EFileInfo(e2eFile)
		if err != nil {
			return models.FileApiOutput{}, err
		}
		hashContent, err := getHashContent(e2eFile)
		metaData.UrlDownload = metaData.UrlDownload + "#" + hashContent
		metaData.Name = getFileName(file)
		return metaData, err
	}

	for i := int64(0); i < sizeBytes; i = i + (int64(chunkSize) * megaByte) {
		err = uploadChunk(file, uuid, i, int64(chunkSize)*megaByte, sizeBytes)
		if err != nil {
			return models.FileApiOutput{}, err
		}
	}
	metaData, err := completeChunk(uuid, nameToBase64(file), sizeBytes, realSize, false, uploadParams)
	if err != nil {
		return models.FileApiOutput{}, err
	}
	return metaData, nil
}

func nameToBase64(f *os.File) string {
	return "base64:" + base64.StdEncoding.EncodeToString([]byte(getFileName(f)))
}

func getFileName(f *os.File) string {
	return filepath.Base(f.Name())
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

func completeChunk(uid, filename string, filesize, realsize int64, useE2e bool, uploadParams cliflags.UploadConfig) (models.FileApiOutput, error) {
	type expectedFormat struct {
		FileInfo models.FileApiOutput `json:"FileInfo"`
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
		{"contenttype", "application/octet-stream"}, // TODO
	})
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

func GetE2eInfo() (models.E2EInfoPlainText, error) {
	var result models.E2EInfoEncrypted
	var fileInfo models.E2EInfoPlainText
	resultJson, err := getUrl(gokapiUrl+"/e2e/get", []header{})
	if err != nil {
		return models.E2EInfoPlainText{}, err
	}
	err = json.Unmarshal([]byte(resultJson), &result)
	if err != nil {
		return models.E2EInfoPlainText{}, err
	}
	fileInfo, err = end2end.DecryptData(result, e2eKey)
	if err != nil {
		return models.E2EInfoPlainText{}, EE2eKeyIncorrect
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
