package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"github.com/forceu/gokapi/internal/models"
	"github.com/secure-io/sio-go"
	"golang.org/x/crypto/scrypt"
	"io"
	"log"
)

var encryptedKey, ramCipher []byte

const blockSize = 32
const nonceSize = 12

func InitWithPassword(pw, salt string) {
	if pw == "" {
		panic(errors.New("empty password provided"))
	}

	cipherKey, err := scrypt.Key([]byte(pw), []byte(salt), 1048576, 8, 1, blockSize)
	pw = ""
	if err != nil {
		cipherKey = []byte{}
		log.Fatal(err)
	}
	storeMasterKey(cipherKey)
}

func InitWithCipher(cipherKey []byte) {
	storeMasterKey(cipherKey)
	cipherKey = []byte{}
}

func storeMasterKey(cipherKey []byte) {
	var err error
	ramCipher, err = getRandomData(blockSize)
	if err != nil {
		log.Fatal(err)
	}
	encryptedKey, err = encryptDecryptText(cipherKey, ramCipher, make([]byte, nonceSize), true)
	cipherKey = []byte{}
	if err != nil {
		log.Fatal(err)
	}
}

func getMasterCipher() []byte {
	key, err := encryptDecryptText(encryptedKey, ramCipher, make([]byte, nonceSize), false)
	if err != nil {
		key = []byte{}
		log.Fatal(err)
	}
	return key
}

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

func DecryptReader(encInfo models.EncryptionInfo, input io.Reader, output io.Writer) error {
	key, err := getCipherFromFile(encInfo)
	if err != nil {
		return err
	}
	stream := getStream(key)
	nonce := make([]byte, stream.NonceSize()) // Nonce is not used
	reader := stream.DecryptReader(input, nonce, nil)
	_, err = io.Copy(output, reader)
	return err
}
func GetDecryptWriter(encInfo models.EncryptionInfo, input io.Writer) (io.Writer, error) {
	key, err := getCipherFromFile(encInfo)
	if err != nil {
		return nil, err
	}
	stream := getStream(key)
	nonce := make([]byte, stream.NonceSize()) // Nonce is not used
	return stream.DecryptWriter(input, nonce, nil), nil
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

func getCipherFromFile(encInfo models.EncryptionInfo) ([]byte, error) {
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
	return encryptDecryptText(input, getMasterCipher(), nonce, true)
}
func fileCipherDecrypt(input, nonce []byte) ([]byte, error) {
	return encryptDecryptText(input, getMasterCipher(), nonce, false)
}

func encryptDecryptText(input, cipherBlock, nonce []byte, doEncrypt bool) ([]byte, error) {
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
