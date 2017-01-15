package goricochet

import "testing"
import "time"
import "log"

type TestBadUsageService struct {
	StandardRicochetService
	BadUsageErrorCount    int
	UnknownTypeErrorCount int
	ChannelClosed         int
}

type TestBadUsageConnection struct {
	StandardRicochetConnection
	Service *TestBadUsageService
}

func (ts *TestBadUsageService) OnNewConnection(oc *OpenConnection) {
	ts.StandardRicochetService.OnNewConnection(oc)
	go oc.Process(&TestBadUsageConnection{Service: ts})
}

func (tc *TestBadUsageConnection) OnReady(oc *OpenConnection) {
	if oc.Client {
		oc.OpenChannel(17, "im.ricochet.auth.hidden-service") // Fail because no Extension
	}
	tc.StandardRicochetConnection.OnReady(oc)
	if oc.Client {
		oc.Authenticate(103) // Should Fail because cannot open more than one auth-hidden-service channel at once
	}
}

func (tc *TestBadUsageConnection) OnAuthenticationProof(channelID int32, publicKey []byte, signature []byte) {
	tc.Conn.Authenticate(2)                       // Try to authenticate again...will fail servers don't auth
	tc.Conn.SendContactRequest(4, "test", "test") // Only clients can send contact requests
	tc.StandardRicochetConnection.OnAuthenticationProof(channelID, publicKey, signature)
	tc.Conn.OpenChatChannel(5) // Fail because server can only open even numbered channels
	tc.Conn.OpenChatChannel(3) // Fail because already in use...
}

// OnContactRequest is called when a client sends a new contact request
func (tc *TestBadUsageConnection) OnContactRequest(channelID int32, nick string, message string) {
	tc.Conn.AckContactRequestOnResponse(channelID, "Pending") // Done to keep the contact request channel open
}

func (tc *TestBadUsageConnection) OnAuthenticationResult(channelID int32, result bool, isKnownContact bool) {
	tc.StandardRicochetConnection.OnAuthenticationResult(channelID, result, isKnownContact)

	tc.Conn.OpenChatChannel(3) // Succeed
	tc.Conn.OpenChatChannel(3) // Should fail as duplicate (channel already in use)

	tc.Conn.OpenChatChannel(6) // Should fail because clients are not allowed to open even numbered channels

	tc.Conn.SendMessage(101, "test") // Should fail as 101 doesn't exist

	tc.Conn.Authenticate(1) // Try to authenticate again...will fail because we have already authenticated

	tc.Conn.OpenChannel(19, "im.ricochet.contact.request") // Will Fail
	tc.Conn.SendContactRequest(11, "test", "test")         // Succeed
	tc.Conn.SendContactRequest(13, "test", "test")         // Trigger singleton contact request check

	tc.Conn.OpenChannel(15, "im.ricochet.not-a-real-type") // Fail UnknownType
}

// OnChannelClose is called when a client or server closes an existing channel
func (tc *TestBadUsageConnection) OnChannelClosed(channelID int32) {
	if channelID == 101 {
		log.Printf("Received Channel Closed: %v", channelID)
		tc.Service.ChannelClosed++
	}
}

func (tc *TestBadUsageConnection) OnFailedChannelOpen(channelID int32, errorType string) {
	log.Printf("Failed Channel Open %v %v", channelID, errorType)
	tc.StandardRicochetConnection.OnFailedChannelOpen(channelID, errorType)
	if errorType == "BadUsageError" {
		tc.Service.BadUsageErrorCount++
	} else if errorType == "UnknownTypeError" {
		tc.Service.UnknownTypeErrorCount++
	}
}

func TestBadUsageServer(t *testing.T) {
	ricochetService := new(TestBadUsageService)
	err := ricochetService.Init("./private_key")

	if err != nil {
		t.Errorf("Could not initate ricochet service: %v", err)
	}

	go ricochetService.Listen(ricochetService, 9884)

	time.Sleep(time.Second * 2)

	ricochetService2 := new(TestBadUsageService)
	err = ricochetService2.Init("./private_key")

	if err != nil {
		t.Errorf("Could not initate ricochet service: %v", err)
	}

	go ricochetService2.Listen(ricochetService2, 9885)
	oc, err := ricochetService2.Connect("127.0.0.1:9884|kwke2hntvyfqm7dr")
	if err != nil {
		t.Errorf("Could not connect to ricochet service:  %v", err)
	}
	go oc.Process(&TestBadUsageConnection{
		Service: ricochetService2,
		StandardRicochetConnection: StandardRicochetConnection{
			PrivateKey: ricochetService2.PrivateKey,
		},
	})

	time.Sleep(time.Second * 3)
	if ricochetService2.ChannelClosed != 1 || ricochetService2.BadUsageErrorCount != 7 || ricochetService.BadUsageErrorCount != 4 || ricochetService2.UnknownTypeErrorCount != 1 {
		t.Errorf("Invalid number of errors seen Closed:%v, Client Bad Usage:%v UnknownTypeErrorCount: %v, Server Bad Usage: %v ", ricochetService2.ChannelClosed, ricochetService2.BadUsageErrorCount, ricochetService2.UnknownTypeErrorCount, ricochetService.BadUsageErrorCount)
	}

}
