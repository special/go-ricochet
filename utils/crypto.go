package utils

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
)

// LoadPrivateKeyFromFile loads a private key from a file...
func LoadPrivateKeyFromFile(filename string) (*rsa.PrivateKey, error) {
	pemData, err := ioutil.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("not a private key")
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
