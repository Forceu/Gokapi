package helper

import "net/http"

// RedirectRelative sends a temporary redirect with url as-is in the Location header.
// Unlike http.Redirect, a relative url is not resolved against the request path on the server,
// but by the browser against the URL it requested. This keeps redirects working behind a
// reverse proxy that serves Gokapi under a path prefix and strips it (e.g. /gokapi/admin -> /admin)
func RedirectRelative(w http.ResponseWriter, url string) {
	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
