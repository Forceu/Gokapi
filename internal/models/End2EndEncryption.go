package models

type E2EInfoPlainText struct {
	Files []E2EFile `json:"files"`
}
type E2EInfoEncrypted struct {
	Version int    `json:"version"`
	Nonce   []byte `json:"nonce"`
	Content []byte `json:"content"`
}

type E2EFile struct {
	Id       string `json:"id"`
	Filename string `json:"filename"`
	Cipher   []byte `json:"cipher"`
}

type E2EAvailableFiles struct {
	Ids []string `json:"ids"`
}
