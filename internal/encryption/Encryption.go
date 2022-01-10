package encryption

import (
	"Gokapi/internal/configuration"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"github.com/awnumar/memguard"
	"github.com/secure-io/sio-go"
	"golang.org/x/crypto/scrypt"
	"io"
)

var key *memguard.Enclave

func Init(pw string) {
	if pw == "" {
		memguard.SafePanic(errors.New("empty password provided"))
	}

	settings := configuration.GetServerSettingsReadOnly()
	salt := settings.Authentication.SaltFiles
	configuration.ReleaseReadOnly()
	cipherKey, err := scrypt.Key([]byte(pw), []byte(salt), 1048576, 8, 1, 32)
	if err != nil {
		pw = ""
		cipherKey = []byte{}
		safePanic(err)
	}
	pw = ""

	buf := memguard.NewBufferFromBytes(cipherKey)
	cipherKey = []byte{}
	if buf.Size() == 0 {
		memguard.SafePanic(errors.New("invalid cipher created"))
	}
	key = buf.Seal()
}

func getStream() *sio.Stream {
	pw, err := key.Open()
	safePanic(err)
	defer pw.Seal()
	block, err := aes.NewCipher(pw.Bytes())
	safePanic(err)
	gcm, err := cipher.NewGCM(block)
	safePanic(err)
	stream := sio.NewStream(gcm, sio.BufSize)
	safePanic(err)
	return stream
}

func Encrypt(input io.Reader, output io.Writer) {
	stream := getStream()
	nonce := make([]byte, stream.NonceSize()) // Nonce is not used
	reader := stream.EncryptReader(input, nonce, nil)
	_, err := io.Copy(output, reader)
	safePanic(err)
}

func GetEncWriter(output io.Writer) *sio.EncWriter {
	stream := getStream()
	nonce := make([]byte, stream.NonceSize()) // Nonce is not used
	return stream.EncryptWriter(output, nonce, nil)
}

func Decrypt(input io.Reader, output io.Writer) {
	stream := getStream()
	nonce := make([]byte, stream.NonceSize()) // Nonce is not used
	reader := stream.DecryptReader(input, nonce, nil)
	_, err := io.Copy(output, reader)
	safePanic(err)
}

func safePanic(err error) {
	if err != nil {
		memguard.SafePanic(err)
	}
}
