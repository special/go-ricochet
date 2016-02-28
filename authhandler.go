package goricochet

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"io"
)

type AuthenticationHandler struct {
	clientCookie [16]byte
	serverCookie [16]byte
}

func (ah *AuthenticationHandler) AddClientCookie(cookie []byte) {
	copy(ah.clientCookie[:], cookie[:16])
}

func (ah *AuthenticationHandler) AddServerCookie(cookie []byte) {
	copy(ah.serverCookie[:], cookie[:16])
}

func (ah *AuthenticationHandler) GenRandom() [16]byte {
	var cookie [16]byte
	io.ReadFull(rand.Reader, cookie[:])
	return cookie
}

func (ah *AuthenticationHandler) GenClientCookie() [16]byte {
	ah.clientCookie = ah.GenRandom()
	return ah.clientCookie
}

func (ah *AuthenticationHandler) GenServerCookie() [16]byte {
	ah.serverCookie = ah.GenRandom()
	return ah.serverCookie
}

func (ah *AuthenticationHandler) GenChallenge(clientHostname string, serverHostname string) []byte {
	key := make([]byte, 32)
	copy(key[0:16], ah.clientCookie[:])
	copy(key[16:], ah.serverCookie[:])
	value := []byte(clientHostname + serverHostname)
	mac := hmac.New(sha256.New, key)
	mac.Write(value)
	hmac := mac.Sum(nil)
	return hmac
}
