package headers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/forceu/gokapi/internal/models"
)

// Write sets headers to either display the file inline or to force download, the content type
// and if the file is encrypted, the creation timestamp to now
func Write(file models.File, w http.ResponseWriter, forceDownload, serveDecrypted bool) {
	if forceDownload {
		w.Header().Set("Content-Disposition", "attachment; filename=\""+file.Name+"\"")
	} else {
		w.Header().Set("Content-Disposition", "inline; filename=\""+file.Name+"\"")
	}
	if !file.RequiresClientDecryption() || serveDecrypted {
		w.Header().Set("Content-Type", file.ContentType)
		w.Header().Set("Content-Length", strconv.FormatInt(file.SizeBytes, 10))
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	if file.Encryption.IsEncrypted {
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	}
}
