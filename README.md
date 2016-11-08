# GoRicochet [![Build Status](https://travis-ci.org/s-rah/go-ricochet.svg?branch=master)](https://travis-ci.org/s-rah/go-ricochet) [![Go Report Card](https://goreportcard.com/badge/github.com/s-rah/go-ricochet)](https://goreportcard.com/report/github.com/s-rah/go-ricochet)

![GoRicochet](logo.png)

GoRicochet is an experimental implementation of the [Ricochet Protocol](https://ricochet.im)
in Go.

## Features

* A simple API that you can use to build Automated Ricochet Applications
* A suite of regression tests that test protocol compliance.

## Building an Automated Ricochet Application

Below is a simple echo bot, which responds to any chat message. You can also find this code under `examples/echobot`

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

Each automated ricochet service can extend of the `StandardRicochetService`. From there
certain functions can be extended to fully build out a complete application.

Currently GoRicochet does not establish a hidden service, so to make this service
available to the world you will have to [set up a hidden service](https://www.torproject.org/docs/tor-hidden-service.html.en)

## Security and Usage Note

This project is experimental and has not been independently reviewed. If you are
looking for a quick and easy way to use ricochet please check out [Ricochet Protocol](https://ricochet.im).
