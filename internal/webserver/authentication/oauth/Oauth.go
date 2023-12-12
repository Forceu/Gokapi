package oauth

import (
	"context"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/webserver/authentication"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"strings"
	"time"
)

var config oauth2.Config
var ctx context.Context
var provider *oidc.Provider

var prompt string

// Init starts the oauth connection
func Init(baseUrl string, credentials models.AuthenticationConfig) {
	var err error
	ctx = context.Background()
	provider, err = oidc.NewProvider(ctx, credentials.OauthProvider)
	if err != nil {
		log.Fatal(err)
	}

	if !strings.HasSuffix(baseUrl, "/") {
		baseUrl = baseUrl + "/"
	}
	config = oauth2.Config{
		ClientID:     credentials.OAuthClientId,
		ClientSecret: credentials.OAuthClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  baseUrl + "oauth-callback",
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
	prompt = credentials.OAuthPrompt
}

// HandlerLogin is a handler for showing the login screen
func HandlerLogin(w http.ResponseWriter, r *http.Request) {
	state := helper.GenerateRandomString(16)
	setCallbackCookie(w, state)
	var authOpts []oauth2.AuthCodeOption
	if prompt != "" {
		authOpts = append(authOpts, oauth2.SetAuthURLParam("prompt", prompt))
	}
	http.Redirect(w, r, config.AuthCodeURL(state, authOpts...), http.StatusFound)
}

// HandlerCallback is a handler for processing the oauth callback
func HandlerCallback(w http.ResponseWriter, r *http.Request) {
	state, err := r.Cookie(authentication.CookieOauth)
	if err != nil {
		http.Error(w, "state not found", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("state") != state.Value {
		http.Error(w, "state did not match", http.StatusBadRequest)
		return
	}

	oauth2Token, err := config.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	userInfo, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(oauth2Token))
	if err != nil {
		http.Error(w, "Failed to get userinfo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := struct {
		OAuth2Token *oauth2.Token
		UserInfo    *oidc.UserInfo
	}{oauth2Token, userInfo}

	authentication.CheckOauthUser(resp.UserInfo, w)
}

func setCallbackCookie(w http.ResponseWriter, value string) {
	c := &http.Cookie{
		Name:     authentication.CookieOauth,
		Value:    value,
		MaxAge:   int(time.Hour.Seconds()),
		HttpOnly: true,
	}
	http.SetCookie(w, c)
}
