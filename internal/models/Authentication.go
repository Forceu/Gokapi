package models

type AuthenticationConfig struct {
	Method            int      `json:"Method"`
	SaltAdmin         string   `json:"SaltAdmin"`
	SaltFiles         string   `json:"SaltFiles"`
	Username          string   `json:"Username"`
	Password          string   `json:"Password"`
	HeaderKey         string   `json:"HeaderKey"`
	OauthProvider     string   `json:"OauthProvider"`
	OAuthClientId     string   `json:"OAuthClientId"`
	OAuthClientSecret string   `json:"OAuthClientSecret"`
	HeaderUsers       []string `json:"HeaderUsers"`
	OauthUsers        []string `json:"OauthUsers"`
}
