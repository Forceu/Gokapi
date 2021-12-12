//go:build test
// +build test

package models

import (
	"Gokapi/internal/test"
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
