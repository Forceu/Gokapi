package headers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forceu/gokapi/internal/models"
)

func TestContentSecurityPolicy(t *testing.T) {
	testCases := []struct {
		name           string
		middleware     func(http.Handler) http.Handler
		requestPath    string
		expectedPolicy string
	}{
		{
			name:           "baseline",
			middleware:     ContentSecurityPolicy,
			requestPath:    "/setup/start",
			expectedPolicy: contentSecurityPolicy,
		},
		{
			name:           "setup does not allow StreamSaver framing",
			middleware:     ContentSecurityPolicy,
			requestPath:    streamSaverFramePath,
			expectedPolicy: contentSecurityPolicy,
		},
		{
			name:           "application allows exact StreamSaver path",
			middleware:     ContentSecurityPolicyWithStreamSaver,
			requestPath:    streamSaverFramePath + "?stream=1",
			expectedPolicy: streamSaverContentSecurityPolicy,
		},
		{
			name:           "application denies StreamSaver sibling",
			middleware:     ContentSecurityPolicyWithStreamSaver,
			requestPath:    "/serviceworker/sw.js",
			expectedPolicy: contentSecurityPolicy,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, testCase.requestPath, nil)
			testCase.middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})).ServeHTTP(recorder, request)

			actualPolicy := recorder.Result().Header.Get("Content-Security-Policy")
			if actualPolicy != testCase.expectedPolicy {
				t.Errorf("Content-Security-Policy = %q, want %q", actualPolicy, testCase.expectedPolicy)
			}
		})
	}
}

func TestContentSecurityPolicyPreservesInlineFileSandbox(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/downloadFile", nil)
	file := models.File{Name: "example.html", ContentType: "text/html", SizeBytes: 42}

	ContentSecurityPolicy(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		Write(file, w, false, false)
	})).ServeHTTP(recorder, request)

	policies := recorder.Result().Header.Values("Content-Security-Policy")
	if len(policies) != 2 {
		t.Fatalf("Content-Security-Policy values = %q, want baseline and sandbox policies", policies)
	}
	if policies[0] != contentSecurityPolicy {
		t.Errorf("baseline policy = %q, want %q", policies[0], contentSecurityPolicy)
	}
	if policies[1] != "sandbox" {
		t.Errorf("inline file policy = %q, want %q", policies[1], "sandbox")
	}
}
