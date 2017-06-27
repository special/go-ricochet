package connection

import (
	"crypto/rsa"
	"github.com/s-rah/go-ricochet/utils"
	"net"
	"testing"
	"time"
)

// Server
func ServerAuthValid(hostname string, publicKey rsa.PublicKey) (allowed, known bool) {
	return true, true
}

func TestProcessAuthAsServer(t *testing.T) {

	ln, _ := net.Listen("tcp", "127.0.0.1:0")

	go func() {
		cconn, _ := net.Dial("tcp", ln.Addr().String())

		orc := NewOutboundConnection(cconn, "kwke2hntvyfqm7dr")
		orc.TraceLog(true)
		privateKey, _ := utils.LoadPrivateKeyFromFile("../testing/private_key")

		known, err := HandleOutboundConnection(orc).ProcessAuthAsClient(privateKey)
		if err != nil {
			t.Errorf("Error while testing ProcessAuthAsClient (in ProcessAuthAsServer) %v", err)
			return
		} else if !known {
			t.Errorf("Client should have been known to the server, instead known was: %v", known)
			return
		}
	}()

	conn, _ := ln.Accept()
	privateKey, _ := utils.LoadPrivateKeyFromFile("../testing/private_key")

	rc := NewInboundConnection(conn)
	err := HandleInboundConnection(rc).ProcessAuthAsServer(privateKey, ServerAuthValid)
	if err != nil {
		t.Errorf("Error while testing ProcessAuthAsServer: %v", err)
	}
}

func TestProcessServerAuthFail(t *testing.T) {

	ln, _ := net.Listen("tcp", "127.0.0.1:0")

	go func() {
		cconn, _ := net.Dial("tcp", ln.Addr().String())

		orc := NewOutboundConnection(cconn, "kwke2hntvyfqm7dr")
		privateKey, _ := utils.LoadPrivateKeyFromFile("../testing/private_key")

		HandleOutboundConnection(orc).ProcessAuthAsClient(privateKey)

	}()

	conn, _ := ln.Accept()
	privateKey, _ := utils.LoadPrivateKeyFromFile("../testing/private_key_auth_fail_test")

	rc := NewInboundConnection(conn)
	err := HandleInboundConnection(rc).ProcessAuthAsServer(privateKey, ServerAuthValid)
	if err == nil {
		t.Errorf("Error while testing ProcessAuthAsServer - should have failed %v", err)
	}
}

func TestProcessAuthTimeout(t *testing.T) {

	ln, _ := net.Listen("tcp", "127.0.0.1:0")

	go func() {
		net.Dial("tcp", ln.Addr().String())
		time.Sleep(16 * time.Second)

	}()

	conn, _ := ln.Accept()
	privateKey, _ := utils.LoadPrivateKeyFromFile("../testing/private_key")

	rc := NewInboundConnection(conn)
	err := HandleInboundConnection(rc).ProcessAuthAsServer(privateKey, ServerAuthValid)
	if err != utils.ActionTimedOutError {
		t.Errorf("Error while testing TestProcessAuthTimeout - Should have timed out after 15 seconds")
	}
}
