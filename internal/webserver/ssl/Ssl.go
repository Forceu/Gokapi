package ssl

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/helper"
	"io"
	"math"
	"math/big"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
)

var configDir string

func isCertificatePresent() bool {
	certificate, key := GetCertificateLocations()
	return helper.FileExists(certificate) && helper.FileExists(key)
}

// GetCertificateLocations returns the filepath of the public certificate and private key
func GetCertificateLocations() (string, string) {
	if configDir == "" {
		env := environment.New()
		configDir = env.ConfigDir
	}
	return configDir + "/ssl.crt", configDir + "/ssl.key"
}

// GenerateIfInvalidCert checks validity of the SSL certificate and generates a new one if none is present or if it is expired
func GenerateIfInvalidCert(extUrl string, forceGeneration bool) {
	if !isCertificatePresent() || forceGeneration {
		generateCertificates(extUrl)
	} else {
		days := getDaysRemaining()
		if days < 8 {
			fmt.Println("Certificate is valid for less than 8 days.")
			generateCertificates(extUrl)
		} else {
			fmt.Printf("Certificate is valid for %d days. A new one will be generated 7 days before expiration.\n", days)
		}
	}
}

func getDaysRemaining() int {
	if !isCertificatePresent() {
		return -1
	}
	certificate, _ := GetCertificateLocations()
	file, err := os.Open(certificate)
	helper.Check(err)
	certContent, err := io.ReadAll(file)
	helper.Check(err)
	pemContent, _ := pem.Decode(certContent)
	pub, err := x509.ParseCertificate(pemContent.Bytes)
	helper.Check(err)
	hours := math.Round(pub.NotAfter.Sub(time.Now()).Hours() / 24)
	return int(hours)
}

func getDomain(extUrl string) string {
	u, err := url.Parse(extUrl)
	helper.Check(err)
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return u.Host
	}
	return host
}

func generateCertificates(externalUrl string) {
	fmt.Println("Generating new SSL certificate")
	certificate, key := GetCertificateLocations()
	fileCert, err := os.OpenFile(certificate, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0600)
	helper.Check(err)
	fileKey, err := os.OpenFile(key, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0600)
	helper.Check(err)
	defer fileCert.Close()
	defer fileKey.Close()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	helper.Check(err)
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Gokapi"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	host := getDomain(externalUrl)
	ip := net.ParseIP(host)
	if ip != nil {
		template.IPAddresses = append(template.IPAddresses, ip)
	} else {
		template.DNSNames = append(template.DNSNames, host)
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, priv.Public(), priv)
	helper.Check(err)
	err = pem.Encode(fileCert, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	helper.Check(err)
	pemBlock, err := x509.MarshalECPrivateKey(priv)
	helper.Check(err)
	err = pem.Encode(fileKey, &pem.Block{Type: "EC PRIVATE KEY", Bytes: pemBlock})
	helper.Check(err)
	fingerprint := sha256.New()
	fingerprint.Write(derBytes)
	fmt.Println("SSL certificate generation successful. It will be valid for 365 days.")
	fmt.Println()
	fmt.Println("If you are connecting directly to this server, please check that the certificate matches the following SHA-256 fingerprint:")
	fmt.Println(strings.ToUpper(hex.EncodeToString(fingerprint.Sum(nil))))
	fmt.Println()
}
