package goricochet

import "testing"
import "bytes"

func TestAuthHandler(t *testing.T) {
	authHandler := new(AuthenticationHandler)
	authHandler.AddClientCookie([]byte("abcdefghijklmnop"))
	authHandler.AddServerCookie([]byte("qrstuvwxyz012345"))
	challenge := authHandler.GenChallenge("test.onion", "notareal.onion")
	expectedChallenge := []byte{0xf5, 0xdb, 0xfd, 0xf0, 0x3d, 0x94, 0x14, 0xf1, 0x4b, 0x37, 0x93, 0xe2, 0xa5, 0x11, 0x4a, 0x98, 0x31, 0x90, 0xea, 0xb8, 0x95, 0x7a, 0x2e, 0xaa, 0xd0, 0xd2, 0x0c, 0x74, 0x95, 0xba, 0xab, 0x73}
	t.Log(challenge, expectedChallenge)
	if bytes.Compare(challenge[:], expectedChallenge[:]) != 0 {
		t.Errorf("AuthenticationHandler Challenge Is Invalid, Got %x, Expected %x", challenge, expectedChallenge)
	}
}
