package headers

import "net/http"

const (
	contentSecurityPolicy            = "frame-ancestors 'none'; object-src 'none'; base-uri 'self'"
	streamSaverContentSecurityPolicy = "frame-ancestors 'self'; object-src 'none'; base-uri 'self'"
	streamSaverFramePath             = "/serviceworker/index.html"
)

// ContentSecurityPolicy adds the baseline policy to every response.
func ContentSecurityPolicy(next http.Handler) http.Handler {
	return contentSecurityPolicyHandler(next, false)
}

// ContentSecurityPolicyWithStreamSaver adds the baseline policy while allowing
// the StreamSaver document to be framed by Gokapi's same-origin download page.
func ContentSecurityPolicyWithStreamSaver(next http.Handler) http.Handler {
	return contentSecurityPolicyHandler(next, true)
}

func contentSecurityPolicyHandler(next http.Handler, allowStreamSaverFrame bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		policy := contentSecurityPolicy
		if allowStreamSaverFrame && r.URL.Path == streamSaverFramePath {
			policy = streamSaverContentSecurityPolicy
		}
		w.Header().Set("Content-Security-Policy", policy)
		next.ServeHTTP(w, r)
	})
}
