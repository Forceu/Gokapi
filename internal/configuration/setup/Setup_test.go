package setup

import (
	"Gokapi/internal/models"
	"Gokapi/internal/test"
	"Gokapi/internal/webserver/authentication"
	"bytes"
	"testing"
)

var jsonForms []jsonFormObject

func TestInputToJson(t *testing.T) {
	buf := bytes.NewBufferString(testInputBasicAuth)
	var err error
	_, r := test.GetRecorder("POST", "/setupResult", nil, nil, buf)
	jsonForms, err = inputToJsonForm(r)
	test.IsNil(t, err)
	for _, item := range jsonForms {
		if item.Name == "auth_username" {
			test.IsEqualString(t, item.Value, "admin")
		}
	}

}

var config = models.Configuration{
	Authentication:   models.AuthenticationConfig{},
	Port:             "",
	ServerUrl:        "",
	DefaultDownloads: 0,
	DefaultExpiry:    0,
	DefaultPassword:  "",
	RedirectUrl:      "",
	Sessions:         nil,
	Files:            nil,
	Hotlinks:         nil,
	DownloadStatus:   nil,
	ApiKeys:          nil,
	ConfigVersion:    0,
	LengthId:         0,
	DataDir:          "",
	MaxMemory:        0,
	UseSsl:           false,
	MaxFileSizeMB:    0,
}

func TestToConfiguration(t *testing.T) {
	output,err := toConfiguration(&jsonForms)
	test.IsNil(t, err)
	test.IsEqualInt(t,output.Authentication.Method,authentication.Internal)
	test.IsEqualString(t,output.Authentication.Username,"admin")
	test.IsNotEqualString(t,output.Authentication.Password,"adminadmin")
	test.IsNotEqualString(t,output.Authentication.Password,"")
	test.IsEqualString(t,output.RedirectUrl,"https://github.com/Forceu/Gokapi/")

}

var testInputBasicAuth = "[{\"name\":\"authentication_sel\",\"value\":\"0\"},{\"name\":\"auth_username\",\"value\":\"admin\"},{\"name\":\"auth_pw\",\"value\":\"adminadmin\"},{\"name\":\"auth_pw2\",\"value\":\"adminadmin\"},{\"name\":\"oauth_provider\",\"value\":\"\"},{\"name\":\"oauth_id\",\"value\":\"\"},{\"name\":\"oauth_secret\",\"value\":\"\"},{\"name\":\"oauth_header_users\",\"value\":\"\"},{\"name\":\"auth_headerkey\",\"value\":\"\"},{\"name\":\"auth_header_users\",\"value\":\"\"},{\"name\":\"storage_sel\",\"value\":\"cloud\"},{\"name\":\"s3_bucket\",\"value\":\"testbucket\"},{\"name\":\"s3_region\",\"value\":\"testregion\"},{\"name\":\"s3_api\",\"value\":\"testapi\"},{\"name\":\"s3_secret\",\"value\":\"testsecret\"},{\"name\":\"s3_endpoint\",\"value\":\"testendpoint\"},{\"name\":\"localhost_sel\",\"value\":\"0\"},{\"name\":\"ssl_sel\",\"value\":\"0\"},{\"name\":\"port\",\"value\":\"53842\"},{\"name\":\"url\",\"value\":\"http://127.0.0.1:53842/\"},{\"name\":\"url_redirection\",\"value\":\"https://github.com/Forceu/Gokapi/\"}]\n"
