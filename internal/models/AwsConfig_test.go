package models

import (
	"github.com/forceu/gokapi/internal/test"
	"testing"
)

func TestIsAwsProvided(t *testing.T) {
	config := AwsConfig{}
	test.IsEqualBool(t, config.IsAllProvided(), false)
	config = AwsConfig{
		Bucket:    "test",
		Region:    "test",
		Endpoint:  "",
		KeyId:     "test",
		KeySecret: "test",
	}
	test.IsEqualBool(t, config.IsAllProvided(), true)
}

func TestAwsCredentialModes(t *testing.T) {
	for _, tc := range []struct {
		key, secret string
		valid       bool
	}{
		{"", "", true}, {"key", "secret", true}, {"key", "", false}, {"", "secret", false},
	} {
		config := AwsConfig{Bucket: "bucket", Region: "region", KeyId: tc.key, KeySecret: tc.secret}
		test.IsEqualBool(t, config.IsAllProvided(), tc.valid)
		config.Region = ""
		test.IsEqualBool(t, config.IsAllProvided(), false)
	}
}
