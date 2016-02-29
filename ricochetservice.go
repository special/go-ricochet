package goricochet

type RicochetService interface {
	OnConnect(serverHostname string)
	OnAuthenticationChallenge(channelID int32, serverHostname string, serverCookie [16]byte)
	OnAuthenticationResult(channelID int32, serverHostname string, result bool)

	OnOpenChannelRequest(channelID int32, serverHostname string)
	OnOpenChannelRequestAck(channelID int32, serverHostname string, result bool)
	OnChannelClose(channelID int32, serverHostname string)

	OnContactRequest(channelID string, serverHostname string, nick string, message string)

	OnChatMessage(channelID int32, serverHostname string, messageID int32, message string)
	OnChatMessageAck(channelID int32, serverHostname string, messageID int32)
}
