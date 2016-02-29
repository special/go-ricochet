package main

import (
	"github.com/s-rah/go-ricochet"
)

type EchoBotService struct {
    goricochet.StandardRicochetService
}

func (ebs * EchoBotService) OnAuthenticationResult(channelID int32, serverHostname string, result bool) {
    if true {
        ebs.Ricochet().OpenChatChannel(5)
        ebs.Ricochet().SendMessage(5, "Hi I'm an echo bot, I echo what you say!")
    }
}

func (ebs * EchoBotService) OnChatMessage(channelID int32, serverHostname string, messageId int32, message string) {
   ebs.Ricochet().AckChatMessage(channelID, messageId)
   ebs.Ricochet().SendMessage(5, message)
}

func main() {
    ricochetService := new(EchoBotService)
    ricochetService.Init("./private_key", "kwke2hntvyfqm7dr") 
	err := ricochetService.Ricochet().Connect("kwke2hntvyfqm7dr", "127.0.0.1:55555|jlq67qzo6s4yp3sp")
	if err == nil { 
	    ricochetService.OnConnect("jlq67qzo6s4yp3sp")
        ricochetService.Ricochet().ListenAndWait("jlq67qzo6s4yp3sp", ricochetService)
	}
}
