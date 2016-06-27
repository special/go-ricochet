package goricochet

import "testing"
import "time"
import "log"

type TestUnknownContactService struct {
	StandardRicochetService
	FailedToOpen bool
}

func (ts *TestUnknownContactService) OnAuthenticationResult(oc *OpenConnection, channelID int32, result bool, isKnownContact bool) {
	log.Printf("Authentication Result")
	ts.StandardRicochetService.OnAuthenticationResult(oc, channelID, result, isKnownContact)
	oc.OpenChatChannel(5)
}

func (ts *TestUnknownContactService) OnFailedChannelOpen(oc *OpenConnection, channelID int32, errorType string) {
	log.Printf("Failed Channel Open %v", errorType)
	oc.UnsetChannel(channelID)
	if errorType == "UnauthorizedError" {
		ts.FailedToOpen = true
	}
}

func (ts *TestUnknownContactService) IsKnownContact(hostname string) bool {
	return false
}

func TestUnknownContactServer(t *testing.T) {
	ricochetService := new(StandardRicochetService)
	err := ricochetService.Init("./private_key")

	if err != nil {
		t.Errorf("Could not initate ricochet service: %v", err)
	}

	go ricochetService.Listen(ricochetService, 9882)

	time.Sleep(time.Second * 2)

	ricochetService2 := new(TestUnknownContactService)
	err = ricochetService2.Init("./private_key")

	if err != nil {
		t.Errorf("Could not initate ricochet service: %v", err)
	}

	go ricochetService2.Listen(ricochetService2, 9883)
	err = ricochetService2.Connect("127.0.0.1:9882|kwke2hntvyfqm7dr")
	if err != nil {
		t.Errorf("Could not connect to ricochet service:  %v", err)
	}

	time.Sleep(time.Second * 2)
	if !ricochetService2.FailedToOpen {
		t.Errorf("Test server did receive message should have failed")
	}

}
