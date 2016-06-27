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

func (ts *TestBadUsageService) OnConnect(oc *OpenConnection) {
	if oc.Client {
		oc.OpenChannel(17, "im.ricochet.auth.hidden-service") // Fail because no Extension
	}
	ts.StandardRicochetService.OnConnect(oc)
	if oc.Client {
		oc.Authenticate(103) // Should Fail because cannot open more than one auth-hidden-service channel at once
	}
}

func (ts *TestBadUsageService) OnAuthenticationProof(oc *OpenConnection, channelID int32, publicKey []byte, signature []byte, isKnownContact bool) {
	oc.Authenticate(2)                       // Try to authenticate again...will fail servers don't auth
	oc.SendContactRequest(4, "test", "test") // Only clients can send contact requests
	ts.StandardRicochetService.OnAuthenticationProof(oc, channelID, publicKey, signature, isKnownContact)
	oc.OpenChatChannel(5) // Fail because server can only open even numbered channels
	oc.OpenChatChannel(3) // Fail because already in use...
}

// OnContactRequest is called when a client sends a new contact request
func (ts *TestBadUsageService) OnContactRequest(oc *OpenConnection, channelID int32, nick string, message string) {
	oc.AckContactRequestOnResponse(channelID, "Pending") // Done to keep the contact request channel open
}

func (ts *TestBadUsageService) OnAuthenticationResult(oc *OpenConnection, channelID int32, result bool, isKnownContact bool) {
	ts.StandardRicochetService.OnAuthenticationResult(oc, channelID, result, isKnownContact)

	oc.OpenChatChannel(3) // Succeed
	oc.OpenChatChannel(3) // Should fail as duplicate (channel already in use)

	oc.OpenChatChannel(6) // Should fail because clients are not allowed to open even numbered channels

	oc.SendMessage(101, "test") // Should fail as 101 doesn't exist

	oc.Authenticate(1) // Try to authenticate again...will fail because we have already authenticated

	oc.OpenChannel(19, "im.ricochet.contact.request") // Will Fail
	oc.SendContactRequest(11, "test", "test")         // Succeed
	oc.SendContactRequest(13, "test", "test")         // Trigger singleton contact request check

	oc.OpenChannel(15, "im.ricochet.not-a-real-type") // Fail UnknownType
}

// OnChannelClose is called when a client or server closes an existing channel
func (ts *TestBadUsageService) OnChannelClosed(oc *OpenConnection, channelID int32) {
	if channelID == 101 {
		log.Printf("Received Channel Closed: %v", channelID)
		ts.ChannelClosed++
	}
}

func (ts *TestBadUsageService) OnFailedChannelOpen(oc *OpenConnection, channelID int32, errorType string) {
	log.Printf("Failed Channel Open %v %v", channelID, errorType)
	ts.StandardRicochetService.OnFailedChannelOpen(oc, channelID, errorType)
	if errorType == "BadUsageError" {
		ts.BadUsageErrorCount++
	} else if errorType == "UnknownTypeError" {
		ts.UnknownTypeErrorCount++
	}
}

func (ts *TestBadUsageService) IsKnownContact(hostname string) bool {
	return true
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
	err = ricochetService2.Connect("127.0.0.1:9884|kwke2hntvyfqm7dr")
	if err != nil {
		t.Errorf("Could not connect to ricochet service:  %v", err)
	}

	time.Sleep(time.Second * 3)
	if ricochetService2.ChannelClosed != 1 || ricochetService2.BadUsageErrorCount != 7 || ricochetService.BadUsageErrorCount != 4 || ricochetService2.UnknownTypeErrorCount != 1 {
		t.Errorf("Invalid number of errors seen Closed:%v, Client Bad Usage:%v UnknownTypeErrorCount: %v, Server Bad Usage: %v ", ricochetService2.ChannelClosed, ricochetService2.BadUsageErrorCount, ricochetService2.UnknownTypeErrorCount, ricochetService.BadUsageErrorCount)
	}

}
