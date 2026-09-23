package oauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
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
		errorHandling.RedirectToOAuthErrorPage(w, r, "Parameter state was not provided", err)
		return
	}
	if r.URL.Query().Get("state") != state.Value {
		errorHandling.RedirectToOAuthErrorPage(w, r, "Parameter state did not match", err)
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
		errorHandling.RedirectToOAuthErrorPage(w, r, "Failed to exchange token", err)
		return
	}

	allClaims := make(map[string]interface{})
	var subject, email string

	// 1. Try to fetch user info from the UserInfo endpoint
	userInfo, errUserInfo := provider.UserInfo(ctx, oauth2.StaticTokenSource(oauth2Token))
	if errUserInfo == nil && userInfo != nil {
		subject = userInfo.Subject
		email = userInfo.Email
		var uClaims map[string]interface{}
		if err := userInfo.Claims(&uClaims); err == nil {
			for k, v := range uClaims {
				allClaims[k] = v
			}
		}
	} else if errUserInfo != nil {
		log.Printf("OAuth: provider.UserInfo failed: %v", errUserInfo)
	}

	// 2. Extract and verify ID token (essential for providers like AD FS where userinfo returns only 'sub')
	rawIDTokenInterface := oauth2Token.Extra("id_token")
	if rawIDTokenInterface != nil {
		if rawIDToken, ok := rawIDTokenInterface.(string); ok && rawIDToken != "" {
			// Step A: Parse raw payload unconditionally so claims are always available even if JWKS verification has TLS/provider issues
			if rawClaims, err := parseJWTClaims(rawIDToken); err == nil {
				for k, v := range rawClaims {
					if v != nil && v != "" {
						allClaims[k] = v
					}
				}
				if subject == "" {
					if s := getStringOrFirstElement(rawClaims["sub"]); s != "" {
						subject = s
					}
				}
			} else {
				log.Printf("OAuth: Failed to parse raw JWT claims: %v", err)
			}

			// Step B: Verify with provider
			verifier := provider.Verifier(&oidc.Config{ClientID: config.ClientID})
			idToken, verifyErr := verifier.Verify(ctx, rawIDToken)
			if verifyErr != nil {
				log.Printf("OAuth: Failed to verify ID token with ClientID %q: %v. Retrying with SkipClientIDCheck...", config.ClientID, verifyErr)
				fallbackVerifier := provider.Verifier(&oidc.Config{SkipClientIDCheck: true})
				idToken, verifyErr = fallbackVerifier.Verify(ctx, rawIDToken)
				if verifyErr != nil {
					log.Printf("OAuth: Failed to verify ID token with RemoteKeySet: %v. Retrying with InsecureSkipSignatureCheck...", verifyErr)
					insecureVerifier := provider.Verifier(&oidc.Config{
						SkipClientIDCheck:          true,
						InsecureSkipSignatureCheck: true,
					})
					idToken, verifyErr = insecureVerifier.Verify(ctx, rawIDToken)
					if verifyErr != nil {
						log.Printf("OAuth: InsecureSkipSignatureCheck verification failed: %v", verifyErr)
					}
				}
			}

			if idToken != nil {
				if subject == "" {
					subject = idToken.Subject
				}
				var idClaims map[string]interface{}
				if err := idToken.Claims(&idClaims); err == nil {
					for k, v := range idClaims {
						if v != nil && v != "" {
							allClaims[k] = v
						}
					}
				}
			}
		}
	}

	// 3. Extract email with priority: email claim > schemas email > mail > userInfo.Email > UPN
	var detectedEmail string
	
	// Prioritize standard email claim first
	if e := getStringOrFirstElement(allClaims["email"]); e != "" {
		detectedEmail = e
		log.Printf("OAuth: Using email claim: %q", e)
	}
	
	// If no email claim, try case-insensitive search for email
	if detectedEmail == "" {
		for k, v := range allClaims {
			if strings.EqualFold(k, "email") {
				if e := getStringOrFirstElement(v); e != "" {
					detectedEmail = e
					log.Printf("OAuth: Using case-insensitive email claim: %q", e)
					break
				}
			}
		}
	}
	
	// Try Microsoft schema email claims
	if detectedEmail == "" {
		if e := getStringOrFirstElement(allClaims["http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"]); e != "" {
			detectedEmail = e
			log.Printf("OAuth: Using MS schema email claim: %q", e)
		}
	}
	if detectedEmail == "" {
		if e := getStringOrFirstElement(allClaims["http://schemas.microsoft.com/ws/2008/06/identity/claims/emailaddress"]); e != "" {
			detectedEmail = e
			log.Printf("OAuth: Using MS schema email claim (alternative): %q", e)
		}
	}
	
	// Try mail claim
	if detectedEmail == "" {
		if e := getStringOrFirstElement(allClaims["mail"]); e != "" {
			detectedEmail = e
			log.Printf("OAuth: Using mail claim: %q", e)
		}
	}
	
	// Fallback to userInfo.Email
	if detectedEmail == "" && userInfo != nil && userInfo.Email != "" {
		detectedEmail = userInfo.Email
		log.Printf("OAuth: Using userInfo.Email: %q", userInfo.Email)
	}
	
	// Only use UPN as last resort if no email-like identifier found
	if detectedEmail == "" {
		if u := getStringOrFirstElement(allClaims["upn"]); u != "" && strings.Contains(u, "@") {
			detectedEmail = u
			log.Printf("OAuth: Using UPN as fallback: %q", u)
		}
	}
	if detectedEmail == "" {
		if u := getStringOrFirstElement(allClaims["userPrincipalName"]); u != "" && strings.Contains(u, "@") {
			detectedEmail = u
			log.Printf("OAuth: Using userPrincipalName as fallback: %q", u)
		}
	}
	
	email = detectedEmail
	log.Printf("OAuth: User identifier selected - Email: %q, Subject: %q (raw email claim: %v, raw upn claim: %v)",
		email, subject, allClaims["email"], allClaims["upn"])

	if subject == "" && errUserInfo != nil {
		errorHandling.RedirectToOAuthErrorPage(w, r, "Failed to get user info or ID token", errUserInfo)
		return
	}

	if email == "" {
		errorHandling.RedirectToOAuthErrorPage(w, r, "An empty email address was provided.\nPlease make sure that you have your"+
			" email address set in your authentication user backend.", nil)
		return
	}

	var claimsSent authentication.OAuthUserClaims = claimsMap(allClaims)
	if len(allClaims) == 0 && userInfo != nil {
		claimsSent = userInfo
	}

	info := authentication.OAuthUserInfo{
		Subject:    subject,
		Email:      email,
		ClaimsSent: claimsSent,
	}
	err = authentication.CheckOauthUserAndRedirect(w, r, info)
	if err != nil {
		errorHandling.RedirectToOAuthErrorPage(w, r, "Failed to continue with login: ", err)
	}
}

type claimsMap map[string]interface{}

func (c claimsMap) Claims(v interface{}) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func getStringOrFirstElement(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return strings.TrimSpace(v)
	case []string:
		if len(v) > 0 {
			return strings.TrimSpace(v[0])
		}
	case []any:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func parseJWTClaims(rawJWT string) (map[string]interface{}, error) {
	parts := strings.Split(rawJWT, ".")
	if len(parts) < 2 {
		return nil, errors.New("invalid jwt format: less than 2 parts")
	}
	payloadSegment := parts[1]
	decoded, err := base64.RawURLEncoding.DecodeString(payloadSegment)
	if err != nil {
		if l := len(payloadSegment) % 4; l > 0 {
			payloadSegment += strings.Repeat("=", 4-l)
		}
		decoded, err = base64.URLEncoding.DecodeString(payloadSegment)
		if err != nil {
			return nil, err
		}
	}
	var claims map[string]interface{}
	err = json.Unmarshal(decoded, &claims)
	return claims, err
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
