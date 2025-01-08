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
		HeaderKey:         "",
		OAuthProvider:     "",
		OAuthClientId:     "",
		OAuthClientSecret: "",
	},
	Port:          ":12345",
	ServerUrl:     "https://testserver.com/",
	RedirectUrl:   "https://test.com",
	DatabaseUrl:   "sqlite://./test/gokapitest.sqlite",
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

const exptectedUnidentedOutput = `{"Authentication":{"Method":0,"SaltAdmin":"saltadmin","SaltFiles":"saltfiles","Username":"admin","HeaderKey":"","OauthProvider":"","OAuthClientId":"","OAuthClientSecret":"","OauthGroupScope":"","OAuthRecheckInterval":0,"OAuthGroups":null,"OnlyRegisteredUsers":false},"Port":":12345","ServerUrl":"https://testserver.com/","RedirectUrl":"https://test.com","PublicName":"public-name","DataDir":"test","DatabaseUrl":"sqlite://./test/gokapitest.sqlite","ConfigVersion":14,"LengthId":5,"MaxFileSizeMB":20,"MaxMemory":50,"ChunkSize":0,"MaxParallelUploads":0,"Encryption":{"Level":1,"Cipher":"AA==","Salt":"encsalt","Checksum":"encsum","ChecksumSalt":"encsumsalt"},"UseSsl":true,"PicturesAlwaysLocal":true,"SaveIp":false,"IncludeFilename":false}`
