package main

import (
	"github.com/s-rah/go-ricochet"
	"log"
)

type EchoBotService struct {
	goricochet.StandardRicochetService
}

// Always Accept Contact Requests
func (ts *EchoBotService) IsKnownContact(hostname string) bool {
	return true
}

func (ts *EchoBotService) OnContactRequest(oc *goricochet.OpenConnection, channelID int32, nick string, message string) {
	ts.StandardRicochetService.OnContactRequest(oc, channelID, nick, message)
	oc.AckContactRequestOnResponse(channelID, "Accepted")
	oc.CloseChannel(channelID)
}

func (ebs *EchoBotService) OnChatMessage(oc *goricochet.OpenConnection, channelID int32, messageId int32, message string) {
	log.Printf("Received Message from %s: %s", oc.OtherHostname, message)
	oc.AckChatMessage(channelID, messageId)
	if oc.GetChannelType(6) == "none" {
		oc.OpenChatChannel(6)
	}
	oc.SendMessage(6, message)
}

func main() {
	ricochetService := new(EchoBotService)
	ricochetService.Init("./private_key")
	ricochetService.Listen(ricochetService, 12345)
}
