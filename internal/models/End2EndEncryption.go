package models

type E2EInfoPlainText struct {
	Files []E2EFile `json:"files"`
}
type E2EInfoEncrypted struct {
	Version        int      `json:"version"`
	Nonce          []byte   `json:"nonce"`
	Content        []byte   `json:"content"`
	AvailableFiles []string `json:"availablefiles"`
}

func (e *E2EInfoEncrypted) HasBeenSetUp() bool {
	return e.Version != 0 && len(e.Content) != 0
}

type E2EFile struct {
	Id       string `json:"id"`
	Filename string `json:"filename"`
	Cipher   []byte `json:"cipher"`
}
