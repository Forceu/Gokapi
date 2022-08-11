package models

// E2EInfoPlainText is stored locally and will be encrypted before storing on server
type E2EInfoPlainText struct {
	Files []E2EFile `json:"files"`
}

// E2EInfoEncrypted is the struct that is stored on the server and decrypted locally
type E2EInfoEncrypted struct {
	Version        int      `json:"version"`
	Nonce          []byte   `json:"nonce"`
	Content        []byte   `json:"content"`
	AvailableFiles []string `json:"availablefiles"`
}

// HasBeenSetUp returns true if E2E setup has been run
func (e *E2EInfoEncrypted) HasBeenSetUp() bool {
	return e.Version != 0 && len(e.Content) != 0
}

// E2EFile contains information about a stored e2e file
type E2EFile struct {
	Uuid     string `json:"uuid"`
	Id       string `json:"id"`
	Filename string `json:"filename"`
	Cipher   []byte `json:"cipher"`
}

// E2EHashContent contains the info that is added after the hash for an e2e link
type E2EHashContent struct {
	Filename string `json:"f"`
	Cipher   string `json:"c"`
}
