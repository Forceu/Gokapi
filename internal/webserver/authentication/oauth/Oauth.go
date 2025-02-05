package oauth

import (
	"context"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/webserver/authentication"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"time"
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

// HandlerLogin is a handler for showing the login screen
func HandlerLogin(w http.ResponseWriter, r *http.Request) { // If user clicked logout, force consent
	initLogin(w, r, r.URL.Query().Has("consent"))
}

func initLogin(w http.ResponseWriter, r *http.Request, showConsentScreen bool) {
	state := helper.GenerateRandomString(16)
	setCallbackCookie(w, state)
	prompt := "none"
	if showConsentScreen {
		prompt = "consent"
	}
	http.Redirect(w, r, config.AuthCodeURL(state)+"&prompt="+prompt, http.StatusFound)
}

func isLoginRequired(r *http.Request) bool {
	errorsRequiringLogin := []string{"login_required", "consent_required", "interaction_required"}
	errorCode := r.URL.Query().Get("error")
	for _, possibleError := range errorsRequiringLogin {
		if errorCode == possibleError {
			return true
		}
	}
	return false
}

// HandlerCallback is a handler for processing the oauth callback
func HandlerCallback(w http.ResponseWriter, r *http.Request) {
	state, err := r.Cookie(authentication.CookieOauth)
	if err != nil {
		showOauthErrorPage(w, r, "Parameter state was not provided")
		return
	}
	if r.URL.Query().Get("state") != state.Value {
		showOauthErrorPage(w, r, "Parameter state did not match")
		return
	}

	oauth2Token, err := config.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		if isLoginRequired(r) {
			initLogin(w, r, true)
			return
		}
		showOauthErrorPage(w, r, "Failed to exchange token: "+err.Error())
		return
	}

	userInfo, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(oauth2Token))
	if err != nil {
		showOauthErrorPage(w, r, "Failed to get userinfo: "+err.Error())
		return
	}
	if userInfo.Email == "" {
		showOauthErrorPage(w, r, "An empty email address was provided.\nPlease make sure that you have your email address set in your authentication user backend.")
		return
	}
	info := authentication.OAuthUserInfo{
		Subject:    userInfo.Subject,
		Email:      userInfo.Email,
		ClaimsSent: userInfo,
	}
	err = authentication.CheckOauthUserAndRedirect(info, w)
	if err != nil {
		showOauthErrorPage(w, r, "Failed to extract scope value: "+err.Error())
	}
}

func showOauthErrorPage(w http.ResponseWriter, r *http.Request, errorMessage string) {
	// Extract the query parameters from the original URL
	queryParams := r.URL.Query()
	queryParams.Add("error_generic", errorMessage)
	redirectURL := "./error-oauth?" + queryParams.Encode()
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
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
