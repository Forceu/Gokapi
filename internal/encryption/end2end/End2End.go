package end2end

import (
	"bytes"
	"encoding/gob"
	"github.com/forceu/gokapi/internal/encryption"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
)

func EncryptData(files []models.E2EFile, key, nonce []byte) (models.E2EInfoEncrypted, error) {
	result := models.E2EInfoEncrypted{
		Nonce: nonce,
	}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(files)
	helper.Check(err)

	encryptedResult, err := encryption.EncryptDecryptText(buf.Bytes(), key, nonce, true)
	if err != nil {
		return models.E2EInfoEncrypted{}, err
	}
	result.Content = encryptedResult
	return result, nil
}

func DecryptData(encryptedContent models.E2EInfoEncrypted, key []byte) (models.E2EInfoPlainText, error) {
	result, err := encryption.EncryptDecryptText(encryptedContent.Content, key, encryptedContent.Nonce, false)
	if err != nil {
		return models.E2EInfoPlainText{}, err
	}

	var fileData []models.E2EFile
	buf := bytes.NewBuffer(result)
	dec := gob.NewDecoder(buf)
	err = dec.Decode(&fileData)
	helper.Check(err)
	return models.E2EInfoPlainText{
		Files: fileData,
	}, nil
}
