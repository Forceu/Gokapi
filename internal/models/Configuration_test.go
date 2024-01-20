package models

import (
	"github.com/forceu/gokapi/internal/test"
	"strings"
	"testing"
)

var testConfig = Configuration{
	Authentication: AuthenticationConfig{
		Method:            0,
		SaltAdmin:         "saltadmin",
		SaltFiles:         "saltfiles",
		Username:          "admin",
		Password:          "adminpwhashed",
		HeaderKey:         "",
		OAuthProvider:     "",
		OAuthClientId:     "",
		OAuthClientSecret: "",
		HeaderUsers:       nil,
		OAuthUsers:        nil,
	},
	Port:          ":12345",
	ServerUrl:     "https://testserver.com/",
	RedirectUrl:   "https://test.com",
	ConfigVersion: 14,
	LengthId:      5,
	DataDir:       "test",
	MaxMemory:     50,
	UseSsl:        true,
	MaxFileSizeMB: 20,
	PublicName:    "public-name",
	Encryption: Encryption{
		Level:        1,
		Cipher:       []byte{0x00},
		Salt:         "encsalt",
		Checksum:     "encsum",
		ChecksumSalt: "encsumsalt",
	},
	PicturesAlwaysLocal: true,
}

func TestConfiguration_ToJson(t *testing.T) {
	test.IsEqualBool(t, strings.Contains(string(testConfig.ToJson()), "\"SaltAdmin\": \"saltadmin\""), true)
}

func TestConfiguration_ToString(t *testing.T) {
	test.IsEqualString(t, testConfig.ToString(), exptectedUnidentedOutput)
}

const exptectedUnidentedOutput = `{"Authentication":{"Method":0,"SaltAdmin":"saltadmin","SaltFiles":"saltfiles","Username":"admin","Password":"adminpwhashed","HeaderKey":"","OauthProvider":"","OAuthClientId":"","OAuthClientSecret":"","OauthUserScope":"","OauthGroupScope":"","OAuthRecheckInterval":0,"HeaderUsers":null,"OAuthGroups":null,"OauthUsers":null},"Port":":12345","ServerUrl":"https://testserver.com/","RedirectUrl":"https://test.com","PublicName":"public-name","ConfigVersion":14,"LengthId":5,"DataDir":"test","MaxMemory":50,"UseSsl":true,"MaxFileSizeMB":20,"Encryption":{"Level":1,"Cipher":"AA==","Salt":"encsalt","Checksum":"encsum","ChecksumSalt":"encsumsalt"},"PicturesAlwaysLocal":true,"SaveIp":false}`
