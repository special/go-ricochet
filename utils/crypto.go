package utils

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
)

const (
	InvalidPrivateKeyFileError = Error("InvalidPrivateKeyFileError")
)

// LoadPrivateKeyFromFile loads a private key from a file...
func LoadPrivateKeyFromFile(filename string) (*rsa.PrivateKey, error) {
	pemData, err := ioutil.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, InvalidPrivateKeyFileError
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
