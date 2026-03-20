package configuration

import (
	"os"
	"strings"
	"testing"

	"github.com/forceu/gokapi/internal/configuration/cloudconfig"
	"github.com/forceu/gokapi/internal/configuration/configupgrade"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"golang.org/x/crypto/argon2"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestLoad(t *testing.T) {
	test.IsEqualBool(t, Exists(), true)
	Load()
	test.IsEqualString(t, parsedEnvironment.ConfigDir, "test")
	test.IsEqualString(t, serverSettings.Port, "127.0.0.1:53843")
	test.IsEqualString(t, serverSettings.Authentication.Username, "test")
	test.IsEqualString(t, serverSettings.ServerUrl, "http://127.0.0.1:53843/")
	test.IsEqualBool(t, serverSettings.UseSsl, false)

	_ = os.Setenv("GOKAPI_LENGTH_ID", "20")
	_ = os.Setenv("GOKAPI_LENGTH_HOTLINK_ID", "25")
	Load()
	test.IsEqualInt(t, parsedEnvironment.LengthId, 20)
	test.IsEqualInt(t, parsedEnvironment.LengthHotlinkId, 25)
	_ = os.Unsetenv("GOKAPI_LENGTH_ID")
	_ = os.Unsetenv("GOKAPI_LENGTH_HOTLINK_ID")
	test.IsEqualInt(t, serverSettings.ConfigVersion, configupgrade.CurrentConfigVersion)
	testconfiguration.Create(false)
	Load()
}

func TestLoadFromSetup(t *testing.T) {
	newConfig := models.Configuration{
		Authentication: models.AuthenticationConfig{},
		Port:           "localhost:123",
		ServerUrl:      "serverurl",
		RedirectUrl:    "redirect",
		ConfigVersion:  configupgrade.CurrentConfigVersion,
		DataDir:        "test",
		MaxMemory:      10,
		UseSsl:         true,
		MaxFileSizeMB:  199,
		DatabaseUrl:    "sqlite://./test/gokapi.sqlite",
	}
	newCloudConfig := cloudconfig.CloudConfig{Aws: models.AwsConfig{
		Bucket:    "bucket",
		Region:    "region",
		KeyId:     "keyid",
		KeySecret: "secret",
		Endpoint:  "",
	}}

	testconfiguration.WriteCloudConfigFile(true)
	LoadFromSetup(newConfig, nil, End2EndReconfigParameters{}, "")
	test.FileDoesNotExist(t, "test/cloudconfig.yml")
	test.IsEqualString(t, serverSettings.RedirectUrl, "redirect")

	LoadFromSetup(newConfig, &newCloudConfig, End2EndReconfigParameters{}, "")
	test.FileExists(t, "test/cloudconfig.yml")
	config, ok := cloudconfig.Load()
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, config.Aws.KeyId, "keyid")
	test.IsEqualString(t, serverSettings.ServerUrl, "serverurl")
}

func TestUsesHttps(t *testing.T) {
	usesHttps = false
	test.IsEqualBool(t, UsesHttps(), false)
	usesHttps = true
	test.IsEqualBool(t, UsesHttps(), true)
}

func BenchmarkArgon2id(b *testing.B) {
	salt := []byte(helper.GenerateRandomString(argonSaltLen))
	for i := 0; i < b.N; i++ {
		argon2.IDKey([]byte("password"), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	}
}

func TestHashSha1_KnownVector(t *testing.T) {
	// SHA1("password" + "salt") pre-computed externally for regression
	got := hashSha1("password", "salt")
	// echo -n "passwordsalt" | sha1sum
	want := "c88e9c67041a74e0357befdff93f87dde0904214"
	if got != want {
		t.Errorf("hashSha1 = %q, want %q", got, want)
	}
}

func TestHashSha1_DifferentSaltsDifferentHashes(t *testing.T) {
	h1 := hashSha1("password", "salt1")
	h2 := hashSha1("password", "salt2")
	if h1 == h2 {
		t.Error("different salts should produce different hashes")
	}
}

func TestHashSha1_DifferentPasswordsDifferentHashes(t *testing.T) {
	h1 := hashSha1("password1", "salt")
	h2 := hashSha1("password2", "salt")
	if h1 == h2 {
		t.Error("different passwords should produce different hashes")
	}
}

func TestHashSha1_OutputIs40Chars(t *testing.T) {
	got := hashSha1("password", "salt")
	if len(got) != 40 {
		t.Errorf("SHA1 hex output should be 40 chars, got %d", len(got))
	}
}

// --- HashPassword ---

func TestHashPassword_EmptyPasswordReturnsEmpty(t *testing.T) {
	got := HashPassword("", false, "")
	if got != "" {
		t.Errorf("empty password should return empty string, got %q", got)
	}
}

func TestHashPassword_LegacyPanicOnEmptySalt(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic with empty legacy salt, got none")
		}
	}()
	HashPassword("password", true, "")
}

func TestHashPassword_LegacyMatchesSha1(t *testing.T) {
	got := HashPassword("password", true, "mysalt")
	want := hashSha1("password", "mysalt")
	if got != want {
		t.Errorf("legacy hash = %q, want %q", got, want)
	}
}

func TestHashPassword_Argon2idFormat(t *testing.T) {
	got := HashPassword("password", false, "")
	parts := strings.Split(got, "$")
	if len(parts) != 3 {
		t.Fatalf("expected 3 $-separated parts, got %d: %q", len(parts), got)
	}
	if parts[0] != "argon2id" {
		t.Errorf("prefix = %q, want %q", parts[0], "argon2id")
	}
	if len(parts[1]) == 0 {
		t.Error("salt segment should not be empty")
	}
	if len(parts[2]) == 0 {
		t.Error("hash segment should not be empty")
	}
}

func TestHashPassword_Argon2idUniqueSaltsEachCall(t *testing.T) {
	h1 := HashPassword("password", false, "")
	h2 := HashPassword("password", false, "")
	if h1 == h2 {
		t.Error("two calls with same password should produce different hashes (random salt)")
	}
}

func TestHashPassword_Argon2idDifferentPasswordsDifferentHashes(t *testing.T) {
	// Same underlying salt is impossible to force, but hashes should differ
	h1 := HashPassword("password1", false, "")
	h2 := HashPassword("password2", false, "")
	if h1 == h2 {
		t.Error("different passwords should produce different hashes")
	}
}

// --- VerifyPassword ---

func TestVerifyPassword_LegacyCorrectPassword(t *testing.T) {
	stored := hashSha1("password", "mysalt")
	ok, needsRehash := VerifyPassword("password", stored, "mysalt")
	if !ok {
		t.Error("expected correct legacy password to verify")
	}
	if !needsRehash {
		t.Error("legacy hash should signal needsRehash=true")
	}
}

func TestVerifyPassword_LegacyWrongPassword(t *testing.T) {
	stored := hashSha1("password", "mysalt")
	ok, needsRehash := VerifyPassword("wrongpassword", stored, "mysalt")
	if ok {
		t.Error("wrong password should not verify")
	}
	if !needsRehash {
		t.Error("legacy hash should signal needsRehash=true even on failure")
	}
}

func TestVerifyPassword_Argon2idCorrectPassword(t *testing.T) {
	stored := HashPassword("password", false, "")
	ok, needsRehash := VerifyPassword("password", stored, "")
	if !ok {
		t.Error("correct argon2id password should verify")
	}
	if needsRehash {
		t.Error("argon2id hash should not signal needsRehash")
	}
}

func TestVerifyPassword_Argon2idWrongPassword(t *testing.T) {
	stored := HashPassword("password", false, "")
	ok, needsRehash := VerifyPassword("wrongpassword", stored, "")
	if ok {
		t.Error("wrong password should not verify against argon2id hash")
	}
	if needsRehash {
		t.Error("argon2id hash should not signal needsRehash")
	}
}

func TestVerifyPassword_MalformedHashReturnsFalse(t *testing.T) {
	cases := []string{
		"",
		"notahash",
		"argon2id$onlytwoparts",
		"wrongprefix$abc$def",
		"argon2id$notvalidhex!!!$abc123",
	}
	for _, stored := range cases {
		ok, needsRehash := VerifyPassword("password", stored, "")
		if ok {
			t.Errorf("malformed hash %q should not verify", stored)
		}
		if needsRehash {
			t.Errorf("malformed hash %q should not signal needsRehash", stored)
		}
	}
}

func TestVerifyPassword_EmptyPasswordNeverVerifies(t *testing.T) {
	stored := HashPassword("password", false, "")
	ok, _ := VerifyPassword("", stored, "")
	if ok {
		t.Error("empty password should not verify against a real hash")
	}
}

// --- Round-trip / migration ---

func TestRoundTrip_Argon2id(t *testing.T) {
	passwords := []string{"correct horse battery staple", "P@ssw0rd!", "unicode:日本語"}
	for _, pw := range passwords {
		stored := HashPassword(pw, false, "")
		ok, needsRehash := VerifyPassword(pw, stored, "")
		if !ok {
			t.Errorf("round-trip failed for password %q", pw)
		}
		if needsRehash {
			t.Errorf("needsRehash should be false for argon2id, password %q", pw)
		}
	}
}

func TestMigration_LegacyToArgon2id(t *testing.T) {
	// Simulate the on-login rehash migration path
	const password = "oldpassword"
	const salt = "legacysalt"

	legacyHash := HashPassword(password, true, salt)

	// User logs in: verify with legacy, then rehash
	ok, needsRehash := VerifyPassword(password, legacyHash, salt)
	if !ok || !needsRehash {
		t.Fatal("legacy verification failed or did not signal rehash")
	}

	newHash := HashPassword(password, false, "")

	// Subsequent logins use new argon2id hash
	ok, needsRehash = VerifyPassword(password, newHash, "")
	if !ok {
		t.Error("password should verify after migration")
	}
	if needsRehash {
		t.Error("should not need rehash after migration")
	}
}
