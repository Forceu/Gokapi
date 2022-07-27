//go:build js && wasm

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"github.com/forceu/gokapi/internal/encryption"
	"github.com/forceu/gokapi/internal/encryption/end2end"
	"github.com/forceu/gokapi/internal/models"
	"sync"
	"syscall/js"
)

var fileInfo models.E2EInfoPlainText
var key []byte
var fileMutex sync.Mutex

// Main routine that is called on startup
func main() {
	js.Global().Set("GokapiE2EInfoParse", js.FuncOf(InfoParse))
	js.Global().Set("GokapiE2EInfoEncrypt", js.FuncOf(InfoEncrypt))
	js.Global().Set("GokapiE2EGetById", js.FuncOf(GetById))
	js.Global().Set("GokapiE2EAddFile", js.FuncOf(AddFile))
	js.Global().Set("GokapiE2EGetNewCipher", js.FuncOf(GetNewCipher))
	js.Global().Set("GokapiE2ESetCipher", js.FuncOf(SetCipher))
	println("WASM end-to-end encryption module loaded")
	// Prevent the function from returning, which is required in a wasm module
	select {}
}

func InfoParse(this js.Value, args []js.Value) interface{} {
	var err error
	var e2EncModel models.E2EInfoEncrypted

	e2InfoJson := bytesFromJs(args[0])
	key, err = base64.StdEncoding.DecodeString(args[1].String())
	if err != nil {
		return jsError(err.Error())
	}
	if len(key) != 32 {
		return jsError("invalid cipher provided")
	}
	err = json.Unmarshal(e2InfoJson, &e2EncModel)
	if err != nil {
		return jsError(err.Error())
	}
	fileInfo, err = end2end.DecryptData(e2EncModel, key)
	if err != nil {
		return jsError(err.Error())
	}
	fileInfo.Files, err = removeExpiredFiles(args, fileInfo.Files)
	if err != nil {
		return jsError(err.Error())
	}
	return nil
}

func removeExpiredFiles(args []js.Value, files []models.E2EFile) ([]models.E2EFile, error) {
	availableFilesBase64, err := base64.StdEncoding.DecodeString(args[2].String())
	if err != nil {
		return nil, err
	}
	var fileIds models.E2EAvailableFiles
	buf := bytes.NewBuffer(availableFilesBase64)
	dec := gob.NewDecoder(buf)
	err = dec.Decode(&fileIds)
	if err != nil {
		return nil, err
	}
	cleanedFiles := make([]models.E2EFile, 0)
	for _, file := range files {
		for _, id := range fileIds.Ids {
			if file.Id == id {
				cleanedFiles = append(cleanedFiles, file)
				break
			}
		}
	}
	return cleanedFiles, err
}

func AddFile(this js.Value, args []js.Value) interface{} {
	fileMutex.Lock()
	files := fileInfo.Files
	id := args[0].String()
	fileName := args[1].String()
	cipherBase64 := args[2].String()
	cipher, err := base64.StdEncoding.DecodeString(cipherBase64)
	if err != nil {
		return jsError(err.Error())
	}

	files = append(files, models.E2EFile{
		Id:       id,
		Filename: fileName,
		Cipher:   cipher,
	})
	fileInfo.Files = files
	fileMutex.Unlock()
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
	rawKey, err := base64.StdEncoding.DecodeString(cipher)
	if err != nil {
		return jsError(err.Error())
	}
	if len(rawKey) != 32 {
		return jsError("Invalid cipher length")
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

func GetById(this js.Value, args []js.Value) interface{} {
	id := args[0].String()
	for _, file := range fileInfo.Files {
		if file.Id == id {
			return file
		}
	}
	return jsError("file not found")
}

// Wraps a message into a JavaScript object of type error
func jsError(message string) js.Value {
	errConstructor := js.Global().Get("Error")
	errVal := errConstructor.New(message)
	return errVal
}

// Returns a byte slice from a js.Value
func bytesFromJs(arg js.Value) []byte {
	out := make([]byte, arg.Length())
	js.CopyBytesToGo(out, arg)
	return out
}
