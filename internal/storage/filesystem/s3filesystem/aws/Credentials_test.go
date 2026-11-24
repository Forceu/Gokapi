//go:build !noaws && !awsmock

package aws

import (
	"github.com/forceu/gokapi/internal/models"
	"testing"
)

func TestSessionCredentials(t *testing.T) {
	previous := awsConfig
	t.Cleanup(func() { awsConfig = previous })
	t.Setenv("AWS_ACCESS_KEY_ID", "environment-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "environment-secret")
	t.Setenv("AWS_SESSION_TOKEN", "temporary-token")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	for _, static := range []bool{false, true} {
		awsConfig = models.AwsConfig{Bucket: "bucket", Region: "eu-central-1"}
		if static {
			awsConfig.KeyId = "static-key"
			awsConfig.KeySecret = "static-secret"
		}
		sess := createSession()
		value, err := sess.Config.Credentials.Get()
		if err != nil {
			t.Fatal(err)
		}
		if static {
			if value.AccessKeyID != "static-key" || value.SecretAccessKey != "static-secret" || value.SessionToken != "" {
				t.Fatalf("static credentials not preserved: %s", value.ProviderName)
			}
		} else if value.AccessKeyID != "environment-key" || value.SecretAccessKey != "environment-secret" || value.SessionToken != "temporary-token" {
			t.Fatalf("default credential chain not used: %s", value.ProviderName)
		}
	}
}
