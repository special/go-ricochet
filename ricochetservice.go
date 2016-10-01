package goricochet

// RicochetService provides an interface for building automated ricochet applications.
type RicochetService interface {
	OnReady()
	OnConnect(oc *OpenConnection)
	OnDisconnect(oc *OpenConnection)

	// Authentication Management
	OnAuthenticationRequest(oc *OpenConnection, channelID int32, clientCookie [16]byte)
	OnAuthenticationChallenge(oc *OpenConnection, channelID int32, serverCookie [16]byte)
	OnAuthenticationProof(oc *OpenConnection, channelID int32, publicKey []byte, signature []byte, isKnownContact bool)
	OnAuthenticationResult(oc *OpenConnection, channelID int32, result bool, isKnownContact bool)

	// Contact Management
	IsKnownContact(hostname string) bool
	OnContactRequest(oc *OpenConnection, channelID int32, nick string, message string)
	OnContactRequestAck(oc *OpenConnection, channelID int32, status string)

	// Managing Channels
	OnOpenChannelRequest(oc *OpenConnection, channelID int32, channelType string)
	OnOpenChannelRequestSuccess(oc *OpenConnection, channelID int32)
	OnChannelClosed(oc *OpenConnection, channelID int32)

	// Chat Messages
	OnChatMessage(oc *OpenConnection, channelID int32, messageID int32, message string)
	OnChatMessageAck(oc *OpenConnection, channelID int32, messageID int32)

	// Handle Errors
	OnFailedChannelOpen(oc *OpenConnection, channelID int32, errorType string)
	OnGenericError(oc *OpenConnection, channelID int32)
	OnUnknownTypeError(oc *OpenConnection, channelID int32)
	OnUnauthorizedError(oc *OpenConnection, channelID int32)
	OnBadUsageError(oc *OpenConnection, channelID int32)
	OnFailedError(oc *OpenConnection, channelID int32)
}
