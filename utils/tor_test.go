package utils

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"testing"
)

const privateKeyData = `-----BEGIN RSA PRIVATE KEY-----
MIICXgIBAAKBgQC3xEJBH4oVFaotPJw6dezx67Gv4Xukw8CZRGqNFO8yF7Rejtcj
/0RTqqZwj6H6FjxY60dgYnN6IphW0juemNZhxOXeM/5Gb5xO+kWGi5Qt87aSDxnA
MDLgqw79ihuD3m1C1TBz0olmjXPU1VtadZuZcVBST7SLs2/k55GNNr7BoQIDAQAB
AoGBAK3ybVCdnSQWLM7DJ5LC23Wnx7sXceVlkiLCOyWuYjiFbatwBD/DupaD2yaD
HyzN7XOxyg93QZ2jr5XHTL30KEAn/3akNBsX3sjHZnjVfTwD5+oZKd7HYMMxekWf
87TIx2IHvGEo2NaFMLkEZ5TX3Gre8CYOofjFcpj4661ZfYp9AkEA9I0EmQX26ibs
CRGkwPuEj5q5N/PmIHgMWr1pepOlmzJjnxy6SI3NUwmzKrqM6YUM8loSywqfVMrJ
RVzA5jp76wJBAMBeu2hS8KcUTIu66j0pXMhI5wDA3yLiO53TEMwufCPXcaWUMH+e
5AIPL7aZ8ouf895OH0TZKxPNMnbrJ+5F0aMCQDoi/CDUxipMLnjJdP1bzdvF0Jp4
pRC6+VTpCpZVW11V0VEWJ0LwUwuWlr1ls/If60ACIc2bLN2fh9Gxhzo0VRkCQQCS
nKCAVhYLgLEGHaLAknGgQ8+rB1QIphuBoYc/1n3OYzi+VT7RRSvJVgGrTZFJUNLw
LuIt+sWWBeHcOETqmFO5AkEAwwfcxs8QZtX6hCj2MTPi8Q28LIoA/M6eAqYc2I0B
eXxf2J2Qco7sMmBLr1Jp3jZNd5W2fMtlhUZAomOj4piVOA==
-----END RSA PRIVATE KEY-----`

func TestGetTorHostname(t *testing.T) {

	block, _ := pem.Decode([]byte(privateKeyData))
	privateKey, _ := x509.ParsePKCS1PrivateKey(block.Bytes)

	// DER Encode the Public Key
	publicKeyBytes, _ := asn1.Marshal(rsa.PublicKey{
		N: privateKey.PublicKey.N,
		E: privateKey.PublicKey.E,
	})

	hostname := GetTorHostname(publicKeyBytes)
	t.Log(hostname)
	if hostname != "kwke2hntvyfqm7dr" {
		t.Errorf("Hostname %s does not equal %s", hostname, "kwke2hntvyfqm7dr")
	}
}
