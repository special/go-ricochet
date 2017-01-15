package goricochet

import "testing"
import "time"
import "log"

type TestUnknownContactService struct {
	StandardRicochetService
}

func (ts *TestUnknownContactService) OnNewConnection(oc *OpenConnection) {
	go oc.Process(&TestUnknownContactConnection{})
}

type TestUnknownContactConnection struct {
	StandardRicochetConnection
	FailedToOpen bool
}

func (tc *TestUnknownContactConnection) IsKnownContact(hostname string) bool {
	return false
}

func (tc *TestUnknownContactConnection) OnAuthenticationProof(channelID int32, publicKey, signature []byte) {
	result := tc.Conn.ValidateProof(channelID, publicKey, signature)
	tc.Conn.SendAuthenticationResult(channelID, result, false)
	tc.Conn.IsAuthed = result
	tc.Conn.CloseChannel(channelID)
}

func (tc *TestUnknownContactConnection) OnAuthenticationResult(channelID int32, result bool, isKnownContact bool) {
	log.Printf("Authentication Result")
	tc.StandardRicochetConnection.OnAuthenticationResult(channelID, result, isKnownContact)
	tc.Conn.OpenChatChannel(5)
}

func (tc *TestUnknownContactConnection) OnFailedChannelOpen(channelID int32, errorType string) {
	log.Printf("Failed Channel Open %v", errorType)
	tc.Conn.UnsetChannel(channelID)
	if errorType == "UnauthorizedError" {
		tc.FailedToOpen = true
	}
}

func TestUnknownContactServer(t *testing.T) {
	ricochetService := new(TestUnknownContactService)
	err := ricochetService.Init("./private_key")

	if err != nil {
		t.Errorf("Could not initate ricochet service: %v", err)
	}

	go ricochetService.Listen(ricochetService, 9882)

	time.Sleep(time.Second * 2)

	oc, err := ricochetService.Connect("127.0.0.1:9882|kwke2hntvyfqm7dr")
	if err != nil {
		t.Errorf("Could not connect to ricochet service:  %v", err)
	}
	connectionHandler := &TestUnknownContactConnection{
		StandardRicochetConnection: StandardRicochetConnection{
			PrivateKey: ricochetService.PrivateKey,
		},
	}
	go oc.Process(connectionHandler)

	time.Sleep(time.Second * 2)
	if !connectionHandler.FailedToOpen {
		t.Errorf("Test server did receive message should have failed")
	}

}
