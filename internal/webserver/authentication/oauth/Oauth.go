package oauth

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/webserver/authentication"
	"github.com/forceu/gokapi/internal/webserver/errorHandling"
	"golang.org/x/oauth2"
)

var config oauth2.Config
var ctx context.Context
var provider *oidc.Provider

// Init starts the oauth connection
func Init(baseUrl string, credentials models.AuthenticationConfig) {
	var err error
	ctx = context.Background()
	provider, err = oidc.NewProvider(ctx, credentials.OAuthProvider)
	if err != nil {
		log.Fatal(err)
	}

	systemConfig := configuration.Get()
	scopes := []string{oidc.ScopeOpenID, "profile", "email"}
	if systemConfig.Authentication.OAuthGroupScope != "" {
		scopes = append(scopes, systemConfig.Authentication.OAuthGroupScope)
	}

	config = oauth2.Config{
		ClientID:     credentials.OAuthClientId,
		ClientSecret: credentials.OAuthClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  baseUrl + "oauth-callback",
		Scopes:       scopes,
	}
}

const (
	promptSilent = iota
	promptSelectAccount
	promptConsent
)

// HandlerLogin is a handler for showing the login screen
func HandlerLogin(w http.ResponseWriter, r *http.Request) {
	// If user clicked logout, force account selection
	if r.URL.Query().Has("consent") {
		initLogin(w, r, promptConsent)
		return
	}
	initLogin(w, r, promptSilent)
}

func initLogin(w http.ResponseWriter, r *http.Request, screenType int) {
	state := helper.GenerateRandomString(32)
	setCallbackCookie(w, state)
	var prompt string
	switch screenType {
	case promptSilent:
		prompt = "none"
	case promptSelectAccount:
		prompt = "select_account"
	case promptConsent:
		prompt = "consent"
	default:
		panic("invalid screen type")
	}
	http.Redirect(w, r, config.AuthCodeURL(state)+"&prompt="+prompt, http.StatusFound)
}

func isConsentRequired(r *http.Request) bool {
	return r.URL.Query().Get("error") == "consent_required"
}

func isLoginRequired(r *http.Request) bool {
	errorCode := r.URL.Query().Get("error")
	return errorCode == "login_required" || errorCode == "interaction_required"
}

// HandlerCallback is a handler for processing the oauth callback
func HandlerCallback(w http.ResponseWriter, r *http.Request) {
	state, err := r.Cookie(authentication.CookieOauth)
	if err != nil {
		errorHandling.RedirectToOAuthErrorPage(w, r, "errorpage_oauth_no_state", err)
		return
	}
	if r.URL.Query().Get("state") != state.Value {
		errorHandling.RedirectToOAuthErrorPage(w, r, "errorpage_oauth_state_mismatch", err)
		return
	}

	if isConsentRequired(r) {
		initLogin(w, r, promptConsent)
		return
	}
	if isLoginRequired(r) {
		initLogin(w, r, promptSelectAccount)
		return
	}

	oauth2Token, err := config.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		errorHandling.RedirectToOAuthErrorPage(w, r, "errorpage_oauth_exchange_failed", err)
		return
	}

	userInfo, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(oauth2Token))
	if err != nil {
		errorHandling.RedirectToOAuthErrorPage(w, r, "errorpage_oauth_userinfo_failed", err)
		return
	}
	if userInfo.Email == "" {
		errorHandling.RedirectToOAuthErrorPage(w, r, "errorpage_oauth_no_email", nil)
		return
	}
	info := authentication.OAuthUserInfo{
		Subject:    userInfo.Subject,
		Email:      userInfo.Email,
		ClaimsSent: userInfo,
	}
	err = authentication.CheckOauthUserAndRedirect(w, r, info)
	if err != nil {
		errorHandling.RedirectToOAuthErrorPage(w, r, "errorpage_oauth_login_failed", err)
	}
}

func setCallbackCookie(w http.ResponseWriter, value string) {
	c := &http.Cookie{
		Name:     authentication.CookieOauth,
		Value:    value,
		MaxAge:   int(time.Hour.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, c)
}
