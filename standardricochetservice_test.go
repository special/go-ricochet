package goricochet

import "testing"
import "time"
import "log"

type TestService struct {
	StandardRicochetService
}

func (ts *TestService) OnNewConnection(oc *OpenConnection) {
	ts.StandardRicochetService.OnNewConnection(oc)
	go oc.Process(&TestConnection{})
}

type TestConnection struct {
	StandardRicochetConnection
	KnownContact bool // Mocking contact request
}

func (tc *TestConnection) IsKnownContact(hostname string) bool {
	return tc.KnownContact
}

func (tc *TestConnection) OnAuthenticationProof(channelID int32, publicKey, signature []byte) {
	result := tc.Conn.ValidateProof(channelID, publicKey, signature)
	tc.Conn.SendAuthenticationResult(channelID, result, tc.KnownContact)
	tc.Conn.IsAuthed = result
	tc.Conn.CloseChannel(channelID)
}

func (tc *TestConnection) OnAuthenticationResult(channelID int32, result bool, isKnownContact bool) {
	tc.StandardRicochetConnection.OnAuthenticationResult(channelID, result, isKnownContact)
	if !isKnownContact {
		log.Printf("Sending Contact Request")
		tc.Conn.SendContactRequest(3, "test", "test")
	}
}

func (tc *TestConnection) OnContactRequest(channelID int32, nick string, message string) {
	tc.StandardRicochetConnection.OnContactRequest(channelID, nick, message)
	tc.Conn.AckContactRequestOnResponse(channelID, "Pending")
	tc.Conn.AckContactRequest(channelID, "Accepted")
	tc.KnownContact = true
	tc.Conn.CloseChannel(channelID)
}

func (tc *TestConnection) OnOpenChannelRequestSuccess(channelID int32) {
	tc.StandardRicochetConnection.OnOpenChannelRequestSuccess(channelID)
	tc.Conn.SendMessage(channelID, "TEST MESSAGE")
}

func (tc *TestConnection) OnContactRequestAck(channelID int32, status string) {
	tc.StandardRicochetConnection.OnContactRequestAck(channelID, status)
	if status == "Accepted" {
		log.Printf("Got accepted contact request")
		tc.KnownContact = true
		tc.Conn.OpenChatChannel(5)
	} else if status == "Pending" {
		log.Printf("Got pending contact request")
	}
}

func (tc *TestConnection) OnChatMessage(channelID int32, messageID int32, message string) {
	tc.StandardRicochetConnection.OnChatMessage(channelID, messageID, message)
	if message == "TEST MESSAGE" {
		receivedMessage = true
	}
}

var receivedMessage bool

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
	oc, err := ricochetService2.Connect("127.0.0.1:9878|kwke2hntvyfqm7dr")
	if err != nil {
		t.Errorf("Could not connect to ricochet service:  %v", err)
	}
	testClient := &TestConnection{
		StandardRicochetConnection: StandardRicochetConnection{
			PrivateKey: ricochetService2.PrivateKey,
		},
	}
	go oc.Process(testClient)

	time.Sleep(time.Second * 5) // Wait a bit longer
	if !receivedMessage {
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
	_, err = ricochetService.Connect("127.0.0.1:65535|kwke2hntvyfqm7dr")
	if err == nil {
		t.Errorf("Should not have been been able to connect to 127.0.0.1:65535|kwke2hntvyfqm7dr")
	}
}
