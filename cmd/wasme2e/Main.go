//go:build js && wasm

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/forceu/gokapi/internal/encryption"
	"github.com/forceu/gokapi/internal/encryption/end2end"
	"github.com/forceu/gokapi/internal/models"
	"github.com/secure-io/sio-go"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"sync"
	"syscall/js"
)

var fileInfo models.E2EInfoPlainText
var key []byte
var fileMutex sync.Mutex

var uploads map[string]uploadData

type uploadData struct {
	totalFilesizeEncrypted int64
	totalFilesizePlain     int64
	bytesSent              int64
	id                     string
	writerInput            *bytes.Buffer
	encrypter              *sio.EncWriter
	cipher                 []byte
	filename               string
}

// Main routine that is called on startup
func main() {
	uploads = make(map[string]uploadData)
	js.Global().Set("GokapiE2EInfoParse", js.FuncOf(InfoParse))
	js.Global().Set("GokapiE2EInfoEncrypt", js.FuncOf(InfoEncrypt))
	js.Global().Set("GokapiE2EAddFile", js.FuncOf(AddFile))
	js.Global().Set("GokapiE2EGetNewCipher", js.FuncOf(GetNewCipher))
	js.Global().Set("GokapiE2ESetCipher", js.FuncOf(SetCipher))
	js.Global().Set("GokapiE2EEncryptNew", js.FuncOf(EncryptNew))
	js.Global().Set("GokapiE2EUploadChunk", js.FuncOf(UploadChunk))
	js.Global().Set("GokapiE2EDecryptMenu", js.FuncOf(DecryptMenu))
	println("WASM end-to-end encryption module loaded")
	// Prevent the function from returning, which is required in a wasm module
	select {}
}

func EncryptNew(this js.Value, args []js.Value) interface{} {
	id := args[0].String()
	fileSize := int64(args[1].Float())
	filename := args[2].String()
	fileSizeEncrypted := encryption.CalculateEncryptedFilesize(fileSize)
	cipher, err := encryption.GetRandomCipher()
	if err != nil {
		return jsError(err.Error())
	}
	input := bytes.NewBuffer(nil)
	stream, err := encryption.GetEncryptWriter(cipher, input)
	if err != nil {
		return jsError(err.Error())
	}
	result := uploadData{
		totalFilesizeEncrypted: fileSizeEncrypted,
		totalFilesizePlain:     fileSize,
		bytesSent:              0,
		id:                     id,
		encrypter:              stream,
		writerInput:            input,
		cipher:                 cipher,
		filename:               filename,
	}
	uploads[id] = result
	return fileSizeEncrypted
}

func UploadChunk(this js.Value, args []js.Value) interface{} {
	id := args[0].String()
	if uploads[id].id != id {
		return jsError("upload id not found")
	}
	size := int64(args[1].Float())
	isLastChunk := args[2].Bool()
	chunkContent := make([]byte, size)
	js.CopyBytesToGo(chunkContent, args[3])

	// Handler for the Promise
	// We need to return a Promise because HTTP requests are blocking in Go
	handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resolve := args[0]
		reject := args[1]

		// Run this code asynchronously
		go func() {
			uploadInfo := uploads[id]

			_, err := io.Copy(uploadInfo.encrypter, bytes.NewReader(chunkContent))
			if err != nil {
				reject.Invoke(jsError(err.Error()))
				return
			}
			if isLastChunk {
				err = uploads[id].encrypter.Close()
				if err != nil {
					reject.Invoke(jsError(err.Error()))
					return
				}
			}
			encryptedContent := uploads[id].writerInput.Bytes()

			uploadInfo.bytesSent = uploadInfo.bytesSent + int64(len(encryptedContent))
			uploadInfo.writerInput.Reset()
			uploads[id] = uploadInfo
			chunkContent = nil

			jsResult := js.Global().Get("Uint8Array").New(len(encryptedContent))
			js.CopyBytesToJS(jsResult, encryptedContent)
			resolve.Invoke(jsResult)
		}()
		return nil
	})
	// Create and return the Promise object
	// The Promise will resolve with a Response object
	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

func postChunk(data *[]byte, uuid string, fileSize, offset int64, jsFile js.Value) error {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "encrypted.file")
	if err != nil {
		return err
	}
	_, err = part.Write(*data)
	if err != nil {
		return err
	}

	err = writer.WriteField("dztotalfilesize", strconv.FormatInt(fileSize, 10))
	if err != nil {
		return err
	}
	err = writer.WriteField("dzchunkbyteoffset", strconv.FormatInt(offset, 10))
	if err != nil {
		return err
	}
	err = writer.WriteField("dzuuid", uuid)
	if err != nil {
		return err
	}
	err = writer.Close()
	if err != nil {
		return err
	}

	r, err := http.NewRequest("POST", "./uploadChunk", body)
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", writer.FormDataContentType())
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

func InfoParse(this js.Value, args []js.Value) interface{} {
	var err error
	var e2EncModel models.E2EInfoEncrypted

	e2InfoJson := args[0].String()
	err = json.Unmarshal([]byte(e2InfoJson), &e2EncModel)
	if err != nil {
		return jsError(err.Error())
	}
	fileInfo, err = end2end.DecryptData(e2EncModel, key)
	if err != nil {
		return jsError(err.Error())
	}
	fileInfo.Files = removeExpiredFiles(e2EncModel)
	return nil
}

func DecryptMenu(this js.Value, args []js.Value) interface{} {
	for _, file := range fileInfo.Files {
		cipher := base64.StdEncoding.EncodeToString(file.Cipher)
		hashContent, err := json.Marshal(models.E2EHashContent{
			Filename: file.Filename,
			Cipher:   cipher,
		})
		if err != nil {
			return jsError(err.Error())
		}
		hashBase64 := base64.StdEncoding.EncodeToString(hashContent)
		js.Global().Call("decryptFileEntry", file.Id, file.Filename, hashBase64)
	}
	return nil
}

func removeExpiredFiles(encInfo models.E2EInfoEncrypted) []models.E2EFile {
	cleanedFiles := make([]models.E2EFile, 0)
	for _, id := range encInfo.AvailableFiles {
		for _, file := range fileInfo.Files {
			if file.Id == id {
				cleanedFiles = append(cleanedFiles, file)
				break
			}
		}
	}
	return cleanedFiles
}

func AddFile(this js.Value, args []js.Value) interface{} {
	fileMutex.Lock()
	files := fileInfo.Files
	uuid := args[0].String()
	if uploads[uuid].id != uuid {
		return jsError("upload id not found")
	}
	id := args[1].String()
	fileName := args[2].String()

	files = append(files, models.E2EFile{
		Uuid:     uuid,
		Id:       id,
		Filename: fileName,
		Cipher:   uploads[uuid].cipher,
	})
	fileInfo.Files = files
	fileMutex.Unlock()
	delete(uploads, uuid)
	return nil
}

func GetNewCipher(this js.Value, args []js.Value) interface{} {
	cipher, err := encryption.GetRandomCipher()
	if err != nil {
		return jsError(err.Error())
	}
	setAsMaster := args[0].Bool()
	if setAsMaster {
		key = cipher
	}
	return base64.StdEncoding.EncodeToString(cipher)
}

func SetCipher(this js.Value, args []js.Value) interface{} {
	cipher := args[0].String()
	return setCipher(cipher)
}

func setCipher(keyBase64 string) interface{} {
	rawKey, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return jsError(err.Error())
	}
	if len(rawKey) != 32 {
		return jsError("invalid cipher length")
	}
	key = rawKey
	return nil
}

func InfoEncrypt(this js.Value, args []js.Value) interface{} {
	output, err := end2end.EncryptData(fileInfo.Files, key)
	if err != nil {
		return jsError(err.Error())
	}
	outputJson, err := json.Marshal(output)
	if err != nil {
		return jsError(err.Error())
	}
	return base64.StdEncoding.EncodeToString(outputJson)
}

// Wraps a message into a JavaScript object of type error
func jsError(message string) js.Value {
	errConstructor := js.Global().Get("Error")
	errVal := errConstructor.New(message)
	return errVal
}
