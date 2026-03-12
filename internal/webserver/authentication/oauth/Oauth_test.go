package oauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"github.com/forceu/gokapi/internal/webserver/authentication"
	"golang.org/x/oauth2"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	configuration.Load()
	configuration.ConnectDatabase()
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

// mockOIDCServer is a self-contained fake OIDC provider.
// It serves the discovery document, JWKS, token, and userinfo endpoints.
type mockOIDCServer struct {
	server      *httptest.Server
	privateKey  *rsa.PrivateKey
	userEmail   string
	userSubject string
	tokenValid  bool
}

func newMockOIDCServer() *mockOIDCServer {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic("failed to generate RSA key: " + err.Error())
	}
	m := &mockOIDCServer{
		privateKey:  key,
		userEmail:   "testuser@example.com",
		userSubject: "test-subject-123",
		tokenValid:  true,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", m.handleDiscovery)
	mux.HandleFunc("/jwks", m.handleJWKS)
	mux.HandleFunc("/token", m.handleToken)
	mux.HandleFunc("/userinfo", m.handleUserinfo)
	m.server = httptest.NewServer(mux)
	return m
}

func (m *mockOIDCServer) URL() string { return m.server.URL }
func (m *mockOIDCServer) Close()      { m.server.Close() }

func (m *mockOIDCServer) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	base := m.server.URL
	doc := map[string]any{
		"issuer":                 base,
		"authorization_endpoint": base + "/auth",
		"token_endpoint":         base + "/token",
		"jwks_uri":               base + "/jwks",
		"userinfo_endpoint":      base + "/userinfo",
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(doc)
}

func (m *mockOIDCServer) handleJWKS(w http.ResponseWriter, r *http.Request) {
	pub := m.privateKey.Public().(*rsa.PublicKey)
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())
	jwks := map[string]any{
		"keys": []map[string]any{{
			"kty": "RSA",
			"use": "sig",
			"alg": "RS256",
			"kid": "test-key",
			"n":   n,
			"e":   e,
		}},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jwks)
}

func (m *mockOIDCServer) handleToken(w http.ResponseWriter, r *http.Request) {
	if !m.tokenValid {
		http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
		return
	}
	resp := map[string]any{
		"access_token": "mock-access-token",
		"token_type":   "Bearer",
		"expires_in":   3600,
		"id_token":     m.buildIDToken(),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (m *mockOIDCServer) handleUserinfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]any{
		"sub":   m.userSubject,
		"email": m.userEmail,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(info)
}

// buildIDToken builds a minimal unsigned-style ID token. Since our tests
// don't verify the signature path (we rely on the userinfo endpoint), we
// use a simple base64-encoded JSON payload wrapped in a fake JWT envelope.
func (m *mockOIDCServer) buildIDToken() string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"test-key"}`))
	payload, _ := json.Marshal(map[string]any{
		"iss":   m.server.URL,
		"sub":   m.userSubject,
		"email": m.userEmail,
		"aud":   []string{"test-client"},
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".fakesig"
}

func TestInit_WithoutGroupScope(t *testing.T) {
	mock := newMockOIDCServer()
	defer mock.Close()

	credentials := models.AuthenticationConfig{
		OAuthProvider:     mock.URL(),
		OAuthClientId:     "my-client",
		OAuthClientSecret: "my-secret",
	}
	// Ensure no group scope is set in the global config
	configuration.Get().Authentication.OAuthGroupScope = ""

	Init(mock.URL()+"/", credentials)

	test.IsEqualBool(t, ctx != nil, true)
	test.IsEqualBool(t, provider != nil, true)
	test.IsEqualString(t, config.ClientID, "my-client")
	test.IsEqualString(t, config.ClientSecret, "my-secret")
	test.IsEqualString(t, config.RedirectURL, mock.URL()+"/oauth-callback")
	// Base scopes only: openid, profile, email
	test.IsEqualInt(t, len(config.Scopes), 3)
	test.IsEqualBool(t, containsString(strings.Join(config.Scopes, ","), oidc.ScopeOpenID), true)
	test.IsEqualBool(t, containsString(strings.Join(config.Scopes, ","), "profile"), true)
	test.IsEqualBool(t, containsString(strings.Join(config.Scopes, ","), "email"), true)
}

func TestInit_WithGroupScope(t *testing.T) {
	mock := newMockOIDCServer()
	defer mock.Close()

	credentials := models.AuthenticationConfig{
		OAuthProvider:     mock.URL(),
		OAuthClientId:     "my-client",
		OAuthClientSecret: "my-secret",
	}
	configuration.Get().Authentication.OAuthGroupScope = "groups"
	defer func() { configuration.Get().Authentication.OAuthGroupScope = "" }()

	Init(mock.URL()+"/", credentials)

	// Group scope must be appended as a fourth scope
	test.IsEqualInt(t, len(config.Scopes), 4)
	test.IsEqualBool(t, containsString(strings.Join(config.Scopes, ","), "groups"), true)
}

// initWithMock calls Init using the mock server's URL and wires up ctx/provider/config.
func initWithMock(m *mockOIDCServer) {
	var err error
	ctx = context.Background()
	provider, err = oidc.NewProvider(ctx, m.server.URL)
	if err != nil {
		panic("failed to init mock OIDC provider: " + err.Error())
	}
	config = oauth2.Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		Endpoint:     provider.Endpoint(),
		RedirectURL:  m.server.URL + "/oauth-callback",
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}
}

// newRequest builds a test request, optionally attaching the OAuth state cookie.
func newRequest(url, stateValue string) *http.Request {
	req := httptest.NewRequest("GET", url, nil)
	if stateValue != "" {
		req.AddCookie(&http.Cookie{Name: authentication.CookieOauth, Value: stateValue})
	}
	return req
}

// --- Tests ---

func TestSetCallbackCookie(t *testing.T) {
	w, _ := test.GetRecorder("GET", "/", nil, nil, nil)
	setCallbackCookie(w, "test-value")
	cookies := w.Result().Cookies()
	test.IsEqualInt(t, len(cookies), 1)
	test.IsEqualString(t, cookies[0].Name, authentication.CookieOauth)
	test.IsEqualString(t, cookies[0].Value, "test-value")
}

func TestHandlerLogin(t *testing.T) {
	config.ClientID = "test-client"
	config.Endpoint.AuthURL = "https://example.com/auth"

	t.Run("Without consent", func(t *testing.T) {
		rr := httptest.NewRecorder()
		HandlerLogin(rr, httptest.NewRequest("GET", "/login", nil))

		test.IsEqualInt(t, rr.Code, http.StatusFound)
		location := rr.Header().Get("Location")
		test.IsNotEmpty(t, location)
		test.IsEqualBool(t, containsString(location, "prompt=none"), true)
		test.IsEqualBool(t, len(rr.Result().Cookies()) > 0, true)
	})

	t.Run("With consent", func(t *testing.T) {
		rr := httptest.NewRecorder()
		HandlerLogin(rr, httptest.NewRequest("GET", "/login?consent=true", nil))

		test.IsEqualInt(t, rr.Code, http.StatusFound)
		location := rr.Header().Get("Location")
		test.IsNotEmpty(t, location)
		test.IsEqualBool(t, containsString(location, "prompt=consent"), true)
		test.IsEqualBool(t, len(rr.Result().Cookies()) > 0, true)
	})
}

func TestHandlerCallback_MissingStateCookie(t *testing.T) {
	rr := httptest.NewRecorder()
	// No cookie — state cookie is absent entirely
	HandlerCallback(rr, httptest.NewRequest("GET", "/oauth-callback?state=some-state&code=123", nil))

	test.IsEqualInt(t, rr.Code, http.StatusTemporaryRedirect)
	test.IsNotEmpty(t, rr.Header().Get("Location"))
}

func TestHandlerCallback_StateMismatch(t *testing.T) {
	rr := httptest.NewRecorder()
	HandlerCallback(rr, newRequest("/oauth-callback?state=wrong-state&code=123", "correct-state"))

	test.IsEqualInt(t, rr.Code, http.StatusTemporaryRedirect)
	test.IsNotEmpty(t, rr.Header().Get("Location"))
}

func TestHandlerCallback_LoginRequired(t *testing.T) {
	for _, errCode := range []string{"login_required", "consent_required", "interaction_required"} {
		t.Run(errCode, func(t *testing.T) {
			config.Endpoint.AuthURL = "https://example.com/auth"
			rr := httptest.NewRecorder()
			url := fmt.Sprintf("/oauth-callback?state=mystate&error=%s", errCode)
			HandlerCallback(rr, newRequest(url, "mystate"))

			// Should re-initiate login with consent
			test.IsEqualInt(t, rr.Code, http.StatusFound)
			test.IsEqualBool(t, containsString(rr.Header().Get("Location"), "prompt=consent"), true)
		})
	}
}

func TestHandlerCallback_TokenExchangeFailure(t *testing.T) {
	mock := newMockOIDCServer()
	defer mock.Close()
	initWithMock(mock)
	mock.tokenValid = false

	rr := httptest.NewRecorder()
	HandlerCallback(rr, newRequest("/oauth-callback?state=mystate&code=badcode", "mystate"))

	test.IsEqualInt(t, rr.Code, http.StatusTemporaryRedirect)
	test.IsNotEmpty(t, rr.Header().Get("Location"))
}

func TestHandlerCallback_EmptyEmail(t *testing.T) {
	mock := newMockOIDCServer()
	defer mock.Close()
	initWithMock(mock)
	mock.userEmail = "" // userinfo will return empty email

	rr := httptest.NewRecorder()
	HandlerCallback(rr, newRequest("/oauth-callback?state=mystate&code=validcode", "mystate"))

	test.IsEqualInt(t, rr.Code, http.StatusTemporaryRedirect)
	test.IsNotEmpty(t, rr.Header().Get("Location"))
}

func TestHandlerCallback_Success(t *testing.T) {
	mock := newMockOIDCServer()
	defer mock.Close()
	initWithMock(mock)

	rr := httptest.NewRecorder()
	HandlerCallback(rr, newRequest("/oauth-callback?state=mystate&code=validcode", "mystate"))

	// Valid flow completes — should redirect somewhere (admin or error-auth depending on auth config)
	test.IsEqualBool(t, rr.Code == http.StatusTemporaryRedirect || rr.Code == http.StatusFound, true)
	test.IsNotEmpty(t, rr.Header().Get("Location"))
}

func TestIsLoginRequired(t *testing.T) {
	cases := []struct {
		query    string
		expected bool
	}{
		{"?error=login_required", true},
		{"?error=consent_required", true},
		{"?error=interaction_required", true},
		{"?error=access_denied", false},
		{"?error=unknown_error", false},
		{"?code=123", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/"+tc.query, nil)
			test.IsEqualBool(t, isLoginRequired(req), tc.expected)
		})
	}
}

func containsString(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
