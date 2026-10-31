//go:build test

package cliapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forceu/gokapi/cmd/cli-uploader/cliflags"
	"github.com/forceu/gokapi/internal/test"
)

func TestCompleteChunkExpiryHeader(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		_, _ = w.Write([]byte(`{"Result":"OK","FileInfo":{}}`))
	}))
	defer server.Close()
	Init(server.URL, "apikey", nil)

	testCases := []struct {
		name             string
		params           cliflags.FlagConfig
		maxExpirySeconds int64
		expectedHeader   []string
	}{
		{"no server limit, flag not set", cliflags.FlagConfig{}, 0, []string{"0"}},
		{"no server limit, flag set", cliflags.FlagConfig{ExpiryDays: 5, IsExpiryDaysSet: true}, 0, []string{"5"}},
		{"server limit, flag not set", cliflags.FlagConfig{}, 3600, nil},
		{"server limit, flag set", cliflags.FlagConfig{ExpiryDays: 0, IsExpiryDaysSet: true}, 3600, []string{"0"}},
	}
	for _, tc := range testCases {
		tc.params.JsonOutput = true
		_, err := completeChunk("uuid", "file", 1, 1, false, tc.params, tc.maxExpirySeconds, nil)
		test.IsNilWithMessage(t, err, tc.name)
		test.IsEqual(t, receivedHeaders.Values("expiryDays"), tc.expectedHeader)
	}
}
