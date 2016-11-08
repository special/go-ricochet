package main

import (
	"github.com/s-rah/go-ricochet"
	"log"
)

// EchoBotService is an example service which simply echoes back what a client
// sends it.
type EchoBotService struct {
	goricochet.StandardRicochetService
}

// IsKnownContact is configured to always accept Contact Requests
func (ebs *EchoBotService) IsKnownContact(hostname string) bool {
	return true
}

// OnContactRequest - we always accept new contact request.
func (ebs *EchoBotService) OnContactRequest(oc *goricochet.OpenConnection, channelID int32, nick string, message string) {
	ts.StandardRicochetService.OnContactRequest(oc, channelID, nick, message)
	oc.AckContactRequestOnResponse(channelID, "Accepted")
	oc.CloseChannel(channelID)
}

// OnChatMessage we acknowledge the message, grab the message content and send it back - opening
// a new channel if necessary.
func (ebs *EchoBotService) OnChatMessage(oc *goricochet.OpenConnection, channelID int32, messageID int32, message string) {
	log.Printf("Received Message from %s: %s", oc.OtherHostname, message)
	oc.AckChatMessage(channelID, messageID)
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
