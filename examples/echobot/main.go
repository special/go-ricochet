package main

import (
	"fmt"
	"github.com/s-rah/go-ricochet"
	"time"
)

func main() {
	ricochet := new(goricochet.Ricochet)

	// You will want to replace these values with your own test credentials
	ricochet.Init("./private_key", true)
	ricochet.Connect("kwke2hntvyfqm7dr", "127.0.0.1:55555|jlq67qzo6s4yp3sp")
	
	// Not needed past the initial run
	// TODO need to wait for contact response before sending OpenChannel
	// ricochet.SendContactRequest("EchoBot", "I'm an EchoBot")

	go ricochet.ListenAndWait()
	ricochet.OpenChatChannel(5)
	time.Sleep(time.Second * 1)
	ricochet.SendMessage(5, "Hi I'm an echo bot, I echo what you say!")

	for true {
		message,channel,_ := ricochet.Listen()
		fmt.Print(channel, message)
		if message != "" {
			ricochet.SendMessage(5, message)
		}
	}
}
