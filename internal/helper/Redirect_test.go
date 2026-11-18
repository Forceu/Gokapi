package helper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forceu/gokapi/internal/test"
)

func TestRedirectRelative(t *testing.T) {
	w := httptest.NewRecorder()
	RedirectRelative(w, "../../error")
	test.IsEqualInt(t, w.Code, http.StatusTemporaryRedirect)
	test.IsEqualString(t, w.Header().Get("Location"), "../../error")
}
