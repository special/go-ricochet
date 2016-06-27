package goricochet

import "testing"
import "time"
import "log"

type TestService struct {
	StandardRicochetService
	ReceivedMessage bool
	KnownContact    bool // Mocking contact request
}

func (ts *TestService) OnAuthenticationResult(oc *OpenConnection, channelID int32, result bool, isKnownContact bool) {
	ts.StandardRicochetService.OnAuthenticationResult(oc, channelID, result, isKnownContact)
	if !isKnownContact {
		log.Printf("Sending Contact Request")
		oc.SendContactRequest(3, "test", "test")
	}
}

func (ts *TestService) OnContactRequest(oc *OpenConnection, channelID int32, nick string, message string) {
	ts.StandardRicochetService.OnContactRequest(oc, channelID, nick, message)
	oc.AckContactRequestOnResponse(channelID, "Pending")
	oc.AckContactRequest(channelID, "Accepted")
	ts.KnownContact = true
	oc.CloseChannel(channelID)
}

func (ts *TestService) OnOpenChannelRequestSuccess(oc *OpenConnection, channelID int32) {
	ts.StandardRicochetService.OnOpenChannelRequestSuccess(oc, channelID)
	oc.SendMessage(channelID, "TEST MESSAGE")
}

func (ts *TestService) OnContactRequestAck(oc *OpenConnection, channelID int32, status string) {
	ts.StandardRicochetService.OnContactRequestAck(oc, channelID, status)
	if status == "Accepted" {
		log.Printf("Got accepted contact request")
		ts.KnownContact = true
		oc.OpenChatChannel(5)
	} else if status == "Pending" {
		log.Printf("Got pending contact request")
	}
}

func (ts *TestService) OnChatMessage(oc *OpenConnection, channelID int32, messageID int32, message string) {
	ts.StandardRicochetService.OnChatMessage(oc, channelID, messageID, message)
	if message == "TEST MESSAGE" {
		ts.ReceivedMessage = true
	}
}

func (ts *TestService) IsKnownContact(hostname string) bool {
	return ts.KnownContact
}

func TestServer(t *testing.T) {
	ricochetService := new(TestService)
	err := ricochetService.Init("./private_key")

	if err != nil {
		t.Errorf("Could not initate ricochet service: %v", err)
	}

	go ricochetService.Listen(ricochetService, 9878)

	time.Sleep(time.Second * 2)

	ricochetService2 := new(TestService)
	err = ricochetService2.Init("./private_key")

	if err != nil {
		t.Errorf("Could not initate ricochet service: %v", err)
	}

	go ricochetService2.Listen(ricochetService2, 9879)
	err = ricochetService2.Connect("127.0.0.1:9878|kwke2hntvyfqm7dr")
	if err != nil {
		t.Errorf("Could not connect to ricochet service:  %v", err)
	}

	time.Sleep(time.Second * 5) // Wait a bit longer
	if !ricochetService.ReceivedMessage {
		t.Errorf("Test server did not receive message")
	}

}

func TestServerInvalidKey(t *testing.T) {
	ricochetService := new(TestService)
	err := ricochetService.Init("./private_key.does.not.exist")

	if err == nil {
		t.Errorf("Should not have initate ricochet service, private key should not exist")
	}
}

func TestServerCouldNotConnect(t *testing.T) {
	ricochetService := new(TestService)
	err := ricochetService.Init("./private_key")
	if err != nil {
		t.Errorf("Could not initate ricochet service: %v", err)
	}
	err = ricochetService.Connect("127.0.0.1:65535|kwke2hntvyfqm7dr")
	if err == nil {
		t.Errorf("Should not have been been able to connect to 127.0.0.1:65535|kwke2hntvyfqm7dr")
	}
}
