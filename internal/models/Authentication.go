package models

// AuthenticationConfig holds configuration on how to authenticate to Gokapi admin menu
type AuthenticationConfig struct {
	Method               int      `json:"Method"`
	SaltAdmin            string   `json:"SaltAdmin"`
	SaltFiles            string   `json:"SaltFiles"`
	Username             string   `json:"Username"`
	Password             string   `json:"Password"`
	HeaderKey            string   `json:"HeaderKey"`
	HeaderAdminUser      string   `json:"HeaderAdminUser"`
	OAuthProvider        string   `json:"OauthProvider"`
	OAuthClientId        string   `json:"OAuthClientId"`
	OAuthClientSecret    string   `json:"OAuthClientSecret"`
	OAuthUserScope       string   `json:"OauthUserScope"`
	OAuthGroupScope      string   `json:"OauthGroupScope"`
	OAuthAdminUser       string   `json:"OAuthAdminUser"`
	OAuthRecheckInterval int      `json:"OAuthRecheckInterval"`
	HeaderUsers          []string `json:"HeaderUsers"`
	OAuthGroups          []string `json:"OAuthGroups"`
	OAuthUsers           []string `json:"OauthUsers"`
}

const (
	// AuthenticationInternal authentication method uses a user / password combination handled by Gokapi
	AuthenticationInternal = iota

	// AuthenticationOAuth2 authentication retrieves the users email with Open Connect ID
	AuthenticationOAuth2

	// AuthenticationHeader authentication relies on a header from a reverse proxy to parse the username
	AuthenticationHeader

	// AuthenticationDisabled authentication ignores all internal authentication procedures. A reverse proxy needs to restrict access
	AuthenticationDisabled
)
