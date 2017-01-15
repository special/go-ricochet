package goricochet

import "testing"
import "time"
import "log"

// The purpose of this test is to exercise the Unauthorized Error flows that occur
// when a client attempts to open a Chat Channel or Send a Contact Reuqest before Authentication
// itself with the Service.

type TestUnauthorizedService struct {
	StandardRicochetService
}

func (ts *TestUnauthorizedService) OnNewConnection(oc *OpenConnection) {
	go oc.Process(&StandardRicochetConnection{})
}

type TestUnauthorizedConnection struct {
	StandardRicochetConnection
	FailedToOpen int
}

func (tc *TestUnauthorizedConnection) OnReady(oc *OpenConnection) {
	tc.StandardRicochetConnection.OnReady(oc)
	if oc.Client {
		log.Printf("Attempting Authentication Not Authorized")
		oc.IsAuthed = true // Connections to Servers are Considered Authenticated by Default
		// REMOVED Authenticate
		oc.OpenChatChannel(5)
		oc.SendContactRequest(3, "test", "test")
	}
}

func (tc *TestUnauthorizedConnection) OnFailedChannelOpen(channelID int32, errorType string) {
	tc.Conn.UnsetChannel(channelID)
	if errorType == "UnauthorizedError" {
		tc.FailedToOpen++
	}
}

func TestUnauthorizedClientReject(t *testing.T) {
	ricochetService := new(TestService)
	err := ricochetService.Init("./private_key")

	if err != nil {
		t.Errorf("Could not initate ricochet service: %v", err)
	}

	go ricochetService.Listen(ricochetService, 9880)

	time.Sleep(time.Second * 2)

	ricochetService2 := new(TestUnauthorizedService)
	err = ricochetService2.Init("./private_key")

	if err != nil {
		t.Errorf("Could not initate ricochet service: %v", err)
	}

	go ricochetService2.Listen(ricochetService2, 9881)
	oc, err := ricochetService2.Connect("127.0.0.1:9880|kwke2hntvyfqm7dr")
	if err != nil {
		t.Errorf("Could not connect to ricochet service:  %v", err)
	}
	connectionHandler := &TestUnauthorizedConnection{
		StandardRicochetConnection: StandardRicochetConnection{
			PrivateKey: ricochetService2.PrivateKey,
		},
	}
	go oc.Process(connectionHandler)

	time.Sleep(time.Second * 2)
	if connectionHandler.FailedToOpen != 2 {
		t.Errorf("Test server did not reject open channels with unauthorized error")
	}

}
