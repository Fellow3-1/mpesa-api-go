package mpesa

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"
)

func TestFormatTimestamp(t *testing.T) {
	utc := time.Date(2024, time.July, 1, 9, 30, 0, 0, time.UTC)
	if got := FormatTimestamp(utc); got != "20240701123000" {
		t.Errorf("FormatTimestamp(%v) = %q, want 20240701123000", utc, got)
	}

	eatTime := time.Date(2024, time.July, 1, 12, 30, 0, 0, EAT())
	if got := FormatTimestamp(eatTime); got != "20240701123000" {
		t.Errorf("FormatTimestamp(EAT) = %q", got)
	}
}

func TestGenerateSTKPushPassword(t *testing.T) {
	// Known answer from the Safaricom sandbox documentation.
	const want = "MTc0Mzc5YmZiMjc5ZjlhYTliZGJjZjE1OGU5N2RkNzFhNDY3Y2QyZTBjODkzMDU5YjEwZjc4ZTZiNzJhZGExZWQyYzkxOTIwMTcwODI0MTU1MDU1"
	got := GenerateSTKPushPassword("174379", testPassKey, "20170824155055")
	if got != want {
		t.Fatalf("GenerateSTKPushPassword() = %q, want %q", got, want)
	}

	decoded, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(decoded) != "174379"+testPassKey+"20170824155055" {
		t.Errorf("decoded password = %q", decoded)
	}
}

func TestNormalizePhoneNumber(t *testing.T) {
	valid := map[string]string{
		"0712345678":       "254712345678",
		"0112345678":       "254112345678",
		"712345678":        "254712345678",
		"112345678":        "254112345678",
		"254712345678":     "254712345678",
		"+254712345678":    "254712345678",
		"+254 712 345 678": "254712345678",
		"0712-345-678":     "254712345678",
		"(0712) 345 678":   "254712345678",
		"2540712345678":    "254712345678",
		"00254712345678":   "254712345678",
	}
	for input, want := range valid {
		got, err := NormalizePhoneNumber(input)
		if err != nil {
			t.Errorf("NormalizePhoneNumber(%q): %v", input, err)
			continue
		}
		if got != want {
			t.Errorf("NormalizePhoneNumber(%q) = %q, want %q", input, got, want)
		}
	}

	invalid := []string{"", "   ", "abc", "071234567", "07123456789", "0812345678", "254812345678", "2547123456789"}
	for _, input := range invalid {
		if got, err := NormalizePhoneNumber(input); err == nil {
			t.Errorf("NormalizePhoneNumber(%q) = %q, want an error", input, got)
		} else {
			var validationErr *ValidationError
			if !errors.As(err, &validationErr) {
				t.Errorf("NormalizePhoneNumber(%q) returned %T, want *ValidationError", input, err)
			}
		}
	}
}

func TestLoadCertificateRejectsBadInput(t *testing.T) {
	if _, err := LoadCertificate(nil); err == nil {
		t.Error("expected an error for empty input")
	}
	if _, err := LoadCertificate([]byte("not a pem block")); err == nil {
		t.Error("expected an error for non-PEM input")
	}

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("x")})
	if _, err := LoadCertificate(privateKeyPEM); err == nil {
		t.Error("expected an error for a non-certificate PEM block")
	}
}

func TestEncryptSecurityCredentialRoundTrip(t *testing.T) {
	_, key, pemBytes := testCertificate(t)

	credential, err := GenerateSecurityCredential(pemBytes, "Safaricom123!!")
	if err != nil {
		t.Fatalf("GenerateSecurityCredential: %v", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(credential)
	if err != nil {
		t.Fatalf("credential is not base64: %v", err)
	}
	plaintext, err := rsa.DecryptPKCS1v15(rand.Reader, key, ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(plaintext) != "Safaricom123!!" {
		t.Errorf("decrypted = %q", plaintext)
	}
}

func TestEncryptSecurityCredentialErrors(t *testing.T) {
	if _, err := EncryptSecurityCredential(nil, "secret"); err == nil {
		t.Error("expected an error for a nil certificate")
	}

	cert, _, _ := testCertificate(t)
	if _, err := EncryptSecurityCredential(cert, ""); err == nil {
		t.Error("expected an error for an empty password")
	}

	ecCert := testECDSACertificate(t)
	_, err := EncryptSecurityCredential(ecCert, "secret")
	if err == nil {
		t.Fatal("expected an error for a non-RSA certificate")
	}
	if !strings.Contains(err.Error(), "RSA") {
		t.Errorf("error = %v, want it to mention RSA", err)
	}
}

// testCertificate returns an RSA certificate, its private key and the PEM
// encoding of the certificate.
func testCertificate(t *testing.T) (*x509.Certificate, *rsa.PrivateKey, []byte) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "mpesa test certificate"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	return cert, key, pemBytes
}

func testECDSACertificate(t *testing.T) *x509.Certificate {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ECDSA key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "ec test certificate"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	return cert
}
