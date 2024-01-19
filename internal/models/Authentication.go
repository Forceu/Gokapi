package models

// AuthenticationConfig holds configuration on how to authenticate to Gokapi admin menu
type AuthenticationConfig struct {
	Method            int      `json:"Method"`
	SaltAdmin         string   `json:"SaltAdmin"`
	SaltFiles         string   `json:"SaltFiles"`
	Username          string   `json:"Username"`
	Password          string   `json:"Password"`
	HeaderKey         string   `json:"HeaderKey"`
	OAuthProvider     string   `json:"OauthProvider"`
	OAuthClientId     string   `json:"OAuthClientId"`
	OAuthClientSecret string   `json:"OAuthClientSecret"`
	OAuthUserScope    string   `json:"OauthUserScope"`
	OAuthGroupScope   string   `json:"OauthGroupScope"`
	HeaderUsers       []string `json:"HeaderUsers"`
	OAuthGroups       []string `json:"OAuthGroups"`
	OAuthUsers        []string `json:"OauthUsers"`
}
