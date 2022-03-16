package ssl

import (
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	testconfiguration.WriteSslCertificates(true)
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestIsCertificatePresent(t *testing.T) {
	test.IsEqualBool(t, isCertificatePresent(), true)
	os.Remove("test/ssl.crt")
	test.IsEqualBool(t, isCertificatePresent(), false)
	os.Remove("test/ssl.key")
	test.IsEqualBool(t, isCertificatePresent(), false)
	testconfiguration.WriteSslCertificates(true)
	os.Remove("test/ssl.key")
	test.IsEqualBool(t, isCertificatePresent(), false)
	testconfiguration.WriteSslCertificates(true)
	test.IsEqualBool(t, isCertificatePresent(), true)
}

func TestGetCertificateLocations(t *testing.T) {
	cert, key := GetCertificateLocations()
	test.IsEqualString(t, cert, "test/ssl.crt")
	test.IsEqualString(t, key, "test/ssl.key")
}

func TestGetDomain(t *testing.T) {
	test.IsEqualString(t, getDomain("http://127.0.0.1"), "127.0.0.1")
	test.IsEqualString(t, getDomain("http://127.0.0.1:123"), "127.0.0.1")
	test.IsEqualString(t, getDomain("http://localhost/test"), "localhost")
	test.IsEqualString(t, getDomain("http://localhost:8080/test"), "localhost")
	test.IsEqualString(t, getDomain("https://github.com/forceu/gokapi"), "github.com")
}

func TestGetDaysRemaining(t *testing.T) {
	expiry := time.Unix(2147483645, 0)
	remainingDays := getDaysRemaining()
	result := time.Now().Add(time.Duration(remainingDays) * 24 * time.Hour).Sub(expiry)
	test.IsEqualBool(t, result.Hours() <= 12 && result.Hours() >= -12, true)
	os.Remove("test/ssl.key")
	test.IsEqualInt(t, getDaysRemaining(), -1)
	testconfiguration.WriteSslCertificates(false)
	test.IsEqualBool(t, getDaysRemaining() <= 0, true)
}

func TestGenerateIfInvalidCert(t *testing.T) {
	testconfiguration.WriteSslCertificates(true)
	GenerateIfInvalidCert("http://mydomain.com", false)
	test.IsEqualBool(t, getDaysRemaining() > 500, true)
	GenerateIfInvalidCert("http://mydomain.com", true)
	test.IsEqualInt(t, getDaysRemaining(), 365)
	testconfiguration.WriteSslCertificates(false)
	test.IsEqualBool(t, getDaysRemaining() <= 0, true)
	GenerateIfInvalidCert("https://127.0.0.1:8080/", false)
	test.IsEqualInt(t, getDaysRemaining(), 365)
	os.Remove("test/ssl.crt")
	test.IsEqualInt(t, getDaysRemaining(), -1)
	GenerateIfInvalidCert("http://127.0.0.1/", false)
	test.IsEqualInt(t, getDaysRemaining(), 365)
}
