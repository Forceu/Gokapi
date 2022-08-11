package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/secure-io/sio-go"
	"golang.org/x/crypto/scrypt"
	"io"
	"log"
	"os"
	"time"
)

// NoEncryption means all files are stored in plaintext
const NoEncryption = 0

// LocalEncryptionStored means remote files are stored in plaintext, cipher for local files is in plaintext
const LocalEncryptionStored = 1

// LocalEncryptionInput means remote files are stored in plaintext, password needs to be entered on startup
const LocalEncryptionInput = 2

// FullEncryptionStored means all files are encrypted, cipher for local files is in plaintext
const FullEncryptionStored = 3

// FullEncryptionInput means all files are encrypted, password needs to be entered on startup
const FullEncryptionInput = 4

// EndToEndEncryption means all files are encrypted and decrypted client-side
const EndToEndEncryption = 5

var encryptedKey, ramCipher []byte

const blockSize = 32
const nonceSize = 12

// Init needs to be called to load the master key into memory or ask the user for the password
func Init(config models.Configuration) {
	switch config.Encryption.Level {
	case NoEncryption:
		return
	case LocalEncryptionStored:
		fallthrough
	case FullEncryptionStored:
		initWithCipher(config.Encryption.Cipher)
	case LocalEncryptionInput:
		fallthrough
	case FullEncryptionInput:
		initWithPassword(config.Encryption.Salt, config.Encryption.Checksum, config.Encryption.ChecksumSalt)
	case EndToEndEncryption:
		return
	}
}

func initWithPassword(saltPw, expectedChecksum, saltChecksum string) {
	if saltPw == "" || saltChecksum == "" {
		log.Fatal("Empty salt provided. Please rerun setup with --reconfigure")
	}
	pw := readAndCheckPassword(expectedChecksum, saltChecksum)
	cipherKey, err := scrypt.Key([]byte(pw), []byte(saltPw), 1048576, 8, 1, blockSize)
	if err != nil {
		cipherKey = []byte{}
		log.Fatal(err)
	}

	storeMasterKey(cipherKey)
}

func readAndCheckPassword(expectedChecksum, saltChecksum string) string {
	fmt.Println("Please enter encryption password:")
	pw := helper.ReadPassword()
	if pw == "" {
		log.Fatal("Empty password provided")
	}
	fmt.Print("Checking password")

	checksumFinished := false
	go func() {
		for !checksumFinished {
			fmt.Print(".")
			time.Sleep(time.Second)
		}
	}()

	checkSum := PasswordChecksum(pw, saltChecksum)
	checksumFinished = true

	if checkSum != expectedChecksum {
		pw = ""
		fmt.Println("FAIL")
		log.Fatal("Incorrect password provided")
	}

	fmt.Println("OK")
	return pw
}

// PasswordChecksum creates a checksum which is used to check if the supplied password is correct
func PasswordChecksum(pw, salt string) string {
	cipherKey, err := scrypt.Key([]byte(pw), []byte(salt), 1048576, 8, 1, blockSize)
	if err != nil {
		cipherKey = []byte{}
		log.Fatal(err)
	}

	hasher := sha256.New()
	hasher.Write(cipherKey)
	return hex.EncodeToString(hasher.Sum(nil))
}

func initWithCipher(cipherKey []byte) {
	if len(cipherKey) != 32 {
		log.Fatal("Invalid cipher provided. Please rerun setup with --reconfigure")
	}
	storeMasterKey(cipherKey)
}

func storeMasterKey(cipherKey []byte) {
	var err error
	ramCipher, err = getRandomData(blockSize)
	if err != nil {
		log.Fatal(err)
	}
	encryptedKey, err = EncryptDecryptBytes(cipherKey, ramCipher, make([]byte, nonceSize), true)
	if err != nil {
		log.Fatal(err)
	}
}

func getMasterCipher() []byte {
	key, err := EncryptDecryptBytes(encryptedKey, ramCipher, make([]byte, nonceSize), false)
	if err != nil {
		key = []byte{}
		log.Fatal(err)
	}
	return key
}

// Encrypt encrypts a file
func Encrypt(encInfo *models.EncryptionInfo, input io.Reader, output io.Writer) error {
	key, err := generateNewFileKey(encInfo)
	if err != nil {
		return err
	}
	stream := getStream(key)
	nonce := make([]byte, stream.NonceSize()) // Nonce is not used
	reader := stream.EncryptReader(input, nonce, nil)
	_, err = io.Copy(output, reader)
	return err
}

func createDecryptReader(encInfo models.EncryptionInfo, input io.Reader) (*sio.DecReader, error) {
	key, err := GetCipherFromFile(encInfo)
	if err != nil {
		return nil, err
	}
	stream := getStream(key)
	nonce := make([]byte, stream.NonceSize()) // Nonce is not used
	return stream.DecryptReader(input, nonce, nil), nil
}

// DecryptReader modifies a reader so it can decrypt encrypted files
func DecryptReader(encInfo models.EncryptionInfo, input io.Reader, output io.Writer) error {
	reader, err := createDecryptReader(encInfo, input)
	if err != nil {
		return err
	}
	_, err = io.Copy(output, reader)
	return err
}

// IsCorrectKey checks if correct key is being used. This does not check for complete file authentication.
func IsCorrectKey(encInfo models.EncryptionInfo, input *os.File) bool {
	_, err := createDecryptReader(encInfo, input)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

// GetDecryptWriter returns a writer that can decrypt encrypted files
func GetDecryptWriter(cipherKey []byte, input io.Writer) (io.Writer, error) {
	stream := getStream(cipherKey)
	nonce := make([]byte, stream.NonceSize()) // Nonce is not used
	return stream.DecryptWriter(input, nonce, nil), nil
}

// GetDecryptReader returns a reader that can decrypt encrypted files
func GetDecryptReader(cipherKey []byte, input io.Reader) (io.Reader, error) {
	stream := getStream(cipherKey)
	nonce := make([]byte, stream.NonceSize()) // Nonce is not used
	return stream.DecryptReader(input, nonce, nil), nil
}

// GetEncryptReader returns a reader that can encrypt plain files
func GetEncryptReader(cipherKey []byte, input io.Reader) (io.Reader, error) {
	stream := getStream(cipherKey)
	nonce := make([]byte, stream.NonceSize()) // Nonce is not used
	return stream.EncryptReader(input, nonce, nil), nil
}

// GetEncryptWriter returns a writer that can encrypt plain files
func GetEncryptWriter(cipherKey []byte, input io.Writer) (*sio.EncWriter, error) {
	stream := getStream(cipherKey)
	nonce := make([]byte, stream.NonceSize()) // Nonce is not used
	return stream.EncryptWriter(input, nonce, nil), nil

}

func generateNewFileKey(encInfo *models.EncryptionInfo) ([]byte, error) {
	encryptionKey, err := getRandomData(blockSize)
	if err != nil {
		return []byte{}, err
	}
	nonce, err := getRandomData(nonceSize)
	if err != nil {
		return []byte{}, err
	}
	encInfo.Nonce = nonce
	encInfo.IsEncrypted = true
	encKey, err := fileCipherEncrypt(encryptionKey, nonce)
	if err != nil {
		return []byte{}, err
	}
	encInfo.DecryptionKey = encKey
	return encryptionKey, nil
}

// CalculateEncryptedFilesize returns the filesize of the encrypted file including the encryption overhead
func CalculateEncryptedFilesize(size int64) int64 {
	return size + getStream(make([]byte, blockSize)).Overhead(size)
}

// GetCipherFromFile loads the cipher from a file model
func GetCipherFromFile(encInfo models.EncryptionInfo) ([]byte, error) {
	cipherFile, err := fileCipherDecrypt(encInfo.DecryptionKey, encInfo.Nonce)
	if err != nil {
		return []byte{}, err
	}
	return cipherFile, nil
}

func getStream(cipherKey []byte) *sio.Stream {
	block, err := aes.NewCipher(cipherKey)
	if err != nil {
		log.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Fatal(err)
	}
	stream := sio.NewStream(gcm, sio.BufSize)
	if err != nil {
		log.Fatal(err)
	}
	return stream
}

func fileCipherEncrypt(input, nonce []byte) ([]byte, error) {
	return EncryptDecryptBytes(input, getMasterCipher(), nonce, true)
}
func fileCipherDecrypt(input, nonce []byte) ([]byte, error) {
	return EncryptDecryptBytes(input, getMasterCipher(), nonce, false)
}

// EncryptDecryptBytes encrypts or decrypts a byte array
func EncryptDecryptBytes(input, cipherBlock, nonce []byte, doEncrypt bool) ([]byte, error) {
	block, err := aes.NewCipher(cipherBlock)
	if err != nil {
		return []byte{}, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return []byte{}, err
	}
	if doEncrypt {
		return aesgcm.Seal(nil, nonce, input, nil), nil
	}
	return aesgcm.Open(nil, nonce, input, nil)
}

func getRandomData(size int) ([]byte, error) {
	data := make([]byte, size)
	read, err := rand.Read(data)
	if err != nil {
		return []byte{}, err
	}
	if read != size {
		return []byte{}, errors.New("incorrect size written")
	}
	return data, nil
}

// GetRandomCipher a 32 byte long array with random data
func GetRandomCipher() ([]byte, error) {
	return getRandomData(blockSize)
}

// GetRandomNonce a 12 byte long array with random data
func GetRandomNonce() ([]byte, error) {
	return getRandomData(nonceSize)
}
