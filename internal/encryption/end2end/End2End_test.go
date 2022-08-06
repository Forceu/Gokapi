package end2end

import (
	"github.com/forceu/gokapi/internal/encryption"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"reflect"
	"testing"
)

func TestEncrypting(t *testing.T) {
	cipherEncryption, err := encryption.GetRandomCipher()
	test.IsNil(t, err)
	cipherF1, err := encryption.GetRandomCipher()
	test.IsNil(t, err)
	cipherF2, err := encryption.GetRandomCipher()
	test.IsNil(t, err)

	files := make([]models.E2EFile, 2)
	files = append(files, models.E2EFile{
		Uuid:     "1234",
		Id:       "id123",
		Filename: "testfile",
		Cipher:   cipherF1,
	})
	files = append(files, models.E2EFile{
		Uuid:     "5678",
		Id:       "id5567",
		Filename: "testfile2",
		Cipher:   cipherF2,
	})

	encryptedFiles, err := EncryptData(files, cipherEncryption)
	test.IsNil(t, err)
	test.IsEqualBool(t, len(encryptedFiles.Content) > 0, true)
	test.IsEqualInt(t, encryptedFiles.Version, 1)
	test.IsEqualBool(t, len(encryptedFiles.Nonce) > 0, true)

	decryptedFiles, err := DecryptData(encryptedFiles, cipherEncryption)
	test.IsNil(t, err)
	test.IsEqualBool(t, reflect.DeepEqual(files, decryptedFiles.Files), true)

}
