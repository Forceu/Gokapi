package headers

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/forceu/gokapi/internal/models"
)

// Write sets headers to either display the file inline or to force download, the content type
// and if the file is encrypted, the creation timestamp to now
func Write(file models.File, w http.ResponseWriter, forceDownload, serveDecrypted bool) {
	encodedName := strings.NewReplacer("+", "%2B").Replace(url.PathEscape(file.Name))
	disposition := "attachment"
	if !forceDownload {
		disposition = "inline"
		w.Header().Set("Content-Security-Policy", "sandbox")
	}

	w.Header().Set("Content-Disposition", disposition+"; filename=\""+file.Name+"\"; filename*=UTF-8''"+encodedName)
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
