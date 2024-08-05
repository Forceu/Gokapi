package headers

import (
	"github.com/forceu/gokapi/internal/models"
	"net/http"
	"time"
)

// Write sets headers to either display the file inline or to force download, the content type
// and if the file is encrypted, the creation timestamp to now
func Write(file models.File, w http.ResponseWriter, forceDownload bool) {
	if forceDownload {
		w.Header().Set("Content-Disposition", "attachment; filename=\""+file.Name+"\"")
	} else {
		w.Header().Set("Content-Disposition", "inline; filename=\""+file.Name+"\"")
	}
	w.Header().Set("Content-Type", file.ContentType)

	if file.Encryption.IsEncrypted {
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	}
}
