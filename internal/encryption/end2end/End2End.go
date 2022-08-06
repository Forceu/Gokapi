package end2end

import (
	"bytes"
	"encoding/gob"
	"github.com/forceu/gokapi/internal/encryption"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
)

const e2eVersion = 1

// EncryptData encrypts the locally stored e2e data to save on the server
func EncryptData(files []models.E2EFile, key []byte) (models.E2EInfoEncrypted, error) {
	nonce, err := encryption.GetRandomNonce()
	if err != nil {
		return models.E2EInfoEncrypted{}, err
	}
	result := models.E2EInfoEncrypted{
		Nonce: nonce,
	}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err = enc.Encode(files)
	helper.Check(err)

	encryptedResult, err := encryption.EncryptDecryptBytes(buf.Bytes(), key, nonce, true)
	if err != nil {
		return models.E2EInfoEncrypted{}, err
	}
	result.Content = encryptedResult
	result.Version = e2eVersion
	return result, nil
}

// DecryptData decrypts the e2e data stored on the server
func DecryptData(encryptedContent models.E2EInfoEncrypted, key []byte) (models.E2EInfoPlainText, error) {
	result, err := encryption.EncryptDecryptBytes(encryptedContent.Content, key, encryptedContent.Nonce, false)
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
