package models

type E2EInfoPlainText struct {
	Files []E2EFile `json:"files"`
}
type E2EInfoEncrypted struct {
	Nonce   []byte `json:"nonce"`
	Content []byte `json:"content"`
}

type E2EFile struct {
	Filename string
	Cipher   []byte
}
