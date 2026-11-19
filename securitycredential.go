package mpesa

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
)

// Security credentials
//
// B2C, B2B, reversal, account balance and transaction status requests must be
// signed with a "SecurityCredential": the initiator password encrypted with
// Safaricom's RSA public certificate and base64 encoded.
//
// Download the certificate for your environment from
// https://developer.safaricom.co.ke (sandbox and production use different
// certificates), store it securely and load it with LoadCertificate.

// LoadCertificate parses a PEM encoded X.509 certificate such as the Safaricom
// initiator certificate.
func LoadCertificate(pemBytes []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("mpesa: no PEM block found in certificate data")
	}
	if block.Type != "CERTIFICATE" && block.Type != "TRUSTED CERTIFICATE" {
		return nil, fmt.Errorf("mpesa: unexpected PEM block %q, expected a CERTIFICATE", block.Type)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("mpesa: parse certificate: %w", err)
	}
	return cert, nil
}

// EncryptSecurityCredential encrypts initiatorPassword with the certificate's
// RSA public key (PKCS #1 v1.5) and returns the base64 encoded credential.
func EncryptSecurityCredential(cert *x509.Certificate, initiatorPassword string) (string, error) {
	if cert == nil {
		return "", errors.New("mpesa: certificate is required to generate a security credential")
	}
	if initiatorPassword == "" {
		return "", errors.New("mpesa: initiator password is required to generate a security credential")
	}
	pub, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("mpesa: certificate public key is %T, an RSA key is required", cert.PublicKey)
	}
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, pub, []byte(initiatorPassword))
	if err != nil {
		return "", fmt.Errorf("mpesa: encrypt security credential: %w", err)
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// GenerateSecurityCredential loads a PEM certificate and encrypts the initiator
// password with it.
func GenerateSecurityCredential(certPEM []byte, initiatorPassword string) (string, error) {
	cert, err := LoadCertificate(certPEM)
	if err != nil {
		return "", err
	}
	return EncryptSecurityCredential(cert, initiatorPassword)
}
