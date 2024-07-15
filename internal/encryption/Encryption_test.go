package encryption

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"io"
	"os"
	"testing"

	"golang.org/x/crypto/scrypt"
)

// Note: most of these tests are written by AI

func TestGetRandomCipher(t *testing.T) {
	cipher1, err := GetRandomCipher()
	test.IsNil(t, err)
	test.IsEqualInt(t, len(cipher1), 32)
	cipher2, err := GetRandomCipher()
	test.IsNil(t, err)
	isEqual := bytes.Compare(cipher1, cipher2)
	test.IsEqualBool(t, isEqual != 0, true)
}

func TestInit(t *testing.T) {
	config := models.Configuration{
		Encryption: models.Encryption{
			Level:  NoEncryption,
			Cipher: []byte("01234567890123456789012345678901"),
		},
	}
	Init(config)
	// Testing for no encryption, nothing should change

	config.Encryption.Level = LocalEncryptionStored
	Init(config)
	test.IsNotNil(t, ramCipher)
	test.IsNotNil(t, encryptedKey)

	config.Encryption.Level = FullEncryptionStored
	Init(config)
	test.IsNotNil(t, ramCipher)
	test.IsNotNil(t, encryptedKey)
}

func TestPasswordChecksum(t *testing.T) {
	password := "securepassword"
	salt := "somesalt"
	checksum := PasswordChecksum(password, salt)
	expectedChecksum, err := scrypt.Key([]byte(password), []byte(salt), 1048576, 8, 1, blockSize)
	test.IsNil(t, err)
	hasher := sha256.New()
	hasher.Write(expectedChecksum)
	test.IsEqualString(t, hex.EncodeToString(hasher.Sum(nil)), checksum)
	checksum = PasswordChecksum("testpw", "testsalt")
	test.IsEqualString(t, checksum, "30161cdf03347d6d3f99743532b8523e03e79d4d91ddd3a623be414519ee9ca9")
	checksum = PasswordChecksum("testpw", "test")
	test.IsEqualString(t, checksum, "41d1781205837071affbf2268588b3f2e755f0365cfe16aff6136155c1013029")
	checksum = PasswordChecksum("test", "test")
	test.IsEqualString(t, checksum, "a3325e881a99e897aab8ba1de274803cddd4f035409c98e976fec9b8005694e6")
	checksum = PasswordChecksum("test", "testsalt")
	test.IsEqualString(t, checksum, "2dbcdfd0989dd2e1be0eea54f176c102e891fd4cb8182544fa4c9dba45307846")
}

func TestEncryptDecryptBytes(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	test.IsNil(t, err)

	nonce := make([]byte, 12)
	_, err = rand.Read(nonce)
	test.IsNil(t, err)

	plaintext := []byte("this is some plaintext")

	ciphertext, err := EncryptDecryptBytes(plaintext, key, nonce, true)
	test.IsNil(t, err)

	decrypted, err := EncryptDecryptBytes(ciphertext, key, nonce, false)
	test.IsNil(t, err)
	test.IsEqualByteSlice(t, plaintext, decrypted)
}

func TestGenerateNewFileKey(t *testing.T) {
	encInfo := &models.EncryptionInfo{}
	key, err := generateNewFileKey(encInfo)
	test.IsNil(t, err)
	test.IsEqualInt(t, 32, len(key))
	test.IsEqualInt(t, 12, len(encInfo.Nonce))
	test.IsEqualBool(t, encInfo.IsEncrypted, true)
	test.IsEqualInt(t, 48, len(encInfo.DecryptionKey))
}

func TestEncryptDecrypt(t *testing.T) {
	plaintext := []byte("this is some plaintext")
	input := bytes.NewReader(plaintext)
	var encrypted bytes.Buffer
	encInfo := &models.EncryptionInfo{}

	err := Encrypt(encInfo, input, &encrypted)
	test.IsNil(t, err)

	var decrypted bytes.Buffer
	err = DecryptReader(*encInfo, &encrypted, &decrypted)
	test.IsNil(t, err)
	test.IsEqualByteSlice(t, plaintext, decrypted.Bytes())
}

func TestGetRandomData(t *testing.T) {
	data, err := getRandomData(32)
	test.IsNil(t, err)
	test.IsEqualInt(t, 32, len(data))
}

func TestCalculateEncryptedFilesize(t *testing.T) {
	size := int64(1024)
	encryptedSize := CalculateEncryptedFilesize(size)
	test.IsEqualBool(t, encryptedSize > size, true)
}

func TestGetStream(t *testing.T) {
	key, err := GetRandomCipher()
	test.IsNil(t, err)
	stream := getStream(key)
	test.IsNotNil(t, stream)
}

func TestStoreMasterKey(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	test.IsNil(t, err)

	storeMasterKey(key)
	test.IsNotNil(t, ramCipher)
	test.IsNotNil(t, encryptedKey)
}

func TestFileCipherEncryptDecrypt(t *testing.T) {
	input := []byte("testdata")
	nonce, err := GetRandomNonce()
	test.IsNil(t, err)

	encrypted, err := fileCipherEncrypt(input, nonce)
	test.IsNil(t, err)

	decrypted, err := fileCipherDecrypt(encrypted, nonce)
	test.IsNil(t, err)
	test.IsEqualByteSlice(t, input, decrypted)
}

func TestGetCipherFromFile(t *testing.T) {
	// Initialize the encryption key and nonce
	encInfo := &models.EncryptionInfo{
		DecryptionKey: make([]byte, 32),
		Nonce:         make([]byte, 12),
	}
	_, err := rand.Read(encInfo.DecryptionKey)
	test.IsNil(t, err)
	_, err = rand.Read(encInfo.Nonce)
	test.IsNil(t, err)

	// Set the master key and ram cipher
	key := make([]byte, 32)
	_, err = rand.Read(key)
	test.IsNil(t, err)
	storeMasterKey(key)

	// Encrypt a sample key to store in encInfo.DecryptionKey
	encKey, err := fileCipherEncrypt(key, encInfo.Nonce)
	test.IsNil(t, err)
	encInfo.DecryptionKey = encKey

	// Retrieve the cipher from the file info
	retrievedKey, err := GetCipherFromFile(*encInfo)
	test.IsNil(t, err)
	test.IsEqualInt(t, 32, len(retrievedKey))
	test.IsEqualByteSlice(t, key, retrievedKey)
}

func TestIsCorrectKey(t *testing.T) {
	// Create a temporary file for testing
	file, err := os.CreateTemp("", "testfile")
	test.IsNil(t, err)
	defer os.Remove(file.Name())

	// Write some encrypted data to the file
	encInfo := &models.EncryptionInfo{
		DecryptionKey: make([]byte, 32),
		Nonce:         make([]byte, 12),
	}
	_, err = rand.Read(encInfo.DecryptionKey)
	test.IsNil(t, err)
	_, err = rand.Read(encInfo.Nonce)
	test.IsNil(t, err)

	plaintext := []byte("this is some plaintext")
	input := bytes.NewReader(plaintext)
	err = Encrypt(encInfo, input, file)
	test.IsNil(t, err)

	// Re-open the file for reading
	_, err = file.Seek(0, io.SeekStart)
	test.IsNil(t, err)

	// Test if the key is correct
	isCorrect := IsCorrectKey(*encInfo, file)
	test.IsEqualBool(t, isCorrect, true)
}
