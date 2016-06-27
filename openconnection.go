package goricochet

import (
	"crypto"
	"crypto/rsa"
	"encoding/asn1"
	"github.com/s-rah/go-ricochet/utils"
	"net"
)

// OpenConnection encapsulates the state required to maintain a connection to
// a ricochet service.
// Notably OpenConnection does not enforce limits on the channelIDs, channel Assignments
// or the direction of messages. These are considered to be service enforced rules.
// (and services are considered to be the best to define them).
type OpenConnection struct {
	conn        net.Conn
	authHandler map[int32]*AuthenticationHandler
	channels    map[int32]string
	rni         utils.RicochetNetworkInterface

	Client        bool
	IsAuthed      bool
	MyHostname    string
	OtherHostname string
	Closed        bool
}

// Init intializes a OpenConnection object to a default state.
func (oc *OpenConnection) Init(outbound bool, conn net.Conn) {
	oc.conn = conn
	oc.authHandler = make(map[int32]*AuthenticationHandler)
	oc.channels = make(map[int32]string)
	oc.rni = new(utils.RicochetNetwork)

	oc.Client = outbound
	oc.IsAuthed = false
	oc.MyHostname = ""
	oc.OtherHostname = ""
}

// UnsetChannel removes a type association from the channel.
func (oc *OpenConnection) UnsetChannel(channel int32) {
	oc.channels[channel] = "none"
}

// GetChannelType returns the type of the channel on this connection
func (oc *OpenConnection) GetChannelType(channel int32) string {
	if val, ok := oc.channels[channel]; ok {
		return val
	}
	return "none"
}

func (oc *OpenConnection) setChannel(channel int32, channelType string) {
	oc.channels[channel] = channelType
}

// HasChannel returns true if the connection has a channel of an associated type, false otherwise
func (oc *OpenConnection) HasChannel(channelType string) bool {
	for _, val := range oc.channels {
		if val == channelType {
			return true
		}
	}
	return false
}

// CloseChannel closes a given channel
// Prerequisites:
//              * Must have previously connected to a service
func (oc *OpenConnection) CloseChannel(channel int32) {
	oc.UnsetChannel(channel)
	oc.rni.SendRicochetPacket(oc.conn, channel, []byte{})
}

// Close closes the entire connection
func (oc *OpenConnection) Close() {
	oc.conn.Close()
	oc.Closed = true
}

// Authenticate opens an Authentication Channel and send a client cookie
// Prerequisites:
//              * Must have previously connected to a service
func (oc *OpenConnection) Authenticate(channel int32) {
	defer utils.RecoverFromError()

	oc.authHandler[channel] = new(AuthenticationHandler)
	messageBuilder := new(MessageBuilder)
	data, err := messageBuilder.OpenAuthenticationChannel(channel, oc.authHandler[channel].GenClientCookie())
	utils.CheckError(err)

	oc.setChannel(channel, "im.ricochet.auth.hidden-service")
	oc.rni.SendRicochetPacket(oc.conn, 0, data)
}

// ConfirmAuthChannel responds to a new authentication request.
// Prerequisites:
//              * Must have previously connected to a service
func (oc *OpenConnection) ConfirmAuthChannel(channel int32, clientCookie [16]byte) {
	defer utils.RecoverFromError()

	oc.authHandler[channel] = new(AuthenticationHandler)
	oc.authHandler[channel].AddClientCookie(clientCookie[:])
	messageBuilder := new(MessageBuilder)
	data, err := messageBuilder.ConfirmAuthChannel(channel, oc.authHandler[channel].GenServerCookie())
	utils.CheckError(err)

	oc.setChannel(channel, "im.ricochet.auth.hidden-service")
	oc.rni.SendRicochetPacket(oc.conn, 0, data)
}

// SendProof sends an authentication proof in response to a challenge.
// Prerequisites:
//              * Must have previously connected to a service
//              * channel must be of type auth
func (oc *OpenConnection) SendProof(channel int32, serverCookie [16]byte, publicKeyBytes []byte, privateKey *rsa.PrivateKey) {

	if oc.authHandler[channel] == nil {
		return // NoOp
	}

	oc.authHandler[channel].AddServerCookie(serverCookie[:])

	challenge := oc.authHandler[channel].GenChallenge(oc.MyHostname, oc.OtherHostname)
	signature, _ := rsa.SignPKCS1v15(nil, privateKey, crypto.SHA256, challenge)

	defer utils.RecoverFromError()
	messageBuilder := new(MessageBuilder)
	data, err := messageBuilder.Proof(publicKeyBytes, signature)
	utils.CheckError(err)

	oc.rni.SendRicochetPacket(oc.conn, channel, data)
}

// ValidateProof determines if the given public key and signature align with the
// already established challenge vector for this communication
// Prerequisites:
//              * Must have previously connected to a service
//              * Client and Server must have already sent their respective cookies (Authenticate and ConfirmAuthChannel)
func (oc *OpenConnection) ValidateProof(channel int32, publicKeyBytes []byte, signature []byte) bool {

	if oc.authHandler[channel] == nil {
		return false
	}

	provisionalHostname := utils.GetTorHostname(publicKeyBytes)
	publicKey := new(rsa.PublicKey)
	_, err := asn1.Unmarshal(publicKeyBytes, publicKey)
	if err != nil {
		return false
	}
	challenge := oc.authHandler[channel].GenChallenge(provisionalHostname, oc.MyHostname)
	err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, challenge[:], signature)
	if err == nil {
		return true

	}
	return false

}

// SendAuthenticationResult responds to an existed authentication Proof
// Prerequisites:
//              * Must have previously connected to a service
//              * channel must be of type auth
func (oc *OpenConnection) SendAuthenticationResult(channel int32, accepted bool, isKnownContact bool) {
	defer utils.RecoverFromError()
	messageBuilder := new(MessageBuilder)
	data, err := messageBuilder.AuthResult(accepted, isKnownContact)
	utils.CheckError(err)
	oc.rni.SendRicochetPacket(oc.conn, channel, data)
}

// OpenChatChannel opens a new chat channel with the given id
// Prerequisites:
//              * Must have previously connected to a service
//              * If acting as the client, id must be odd, else even
func (oc *OpenConnection) OpenChatChannel(channel int32) {
	defer utils.RecoverFromError()
	messageBuilder := new(MessageBuilder)
	data, err := messageBuilder.OpenChannel(channel, "im.ricochet.chat")
	utils.CheckError(err)

	oc.setChannel(channel, "im.ricochet.chat")
	oc.rni.SendRicochetPacket(oc.conn, 0, data)
}

// OpenChannel opens a new chat channel with the given id
// Prerequisites:
//              * Must have previously connected to a service
//              * If acting as the client, id must be odd, else even
func (oc *OpenConnection) OpenChannel(channel int32, channelType string) {
	defer utils.RecoverFromError()
	messageBuilder := new(MessageBuilder)
	data, err := messageBuilder.OpenChannel(channel, channelType)
	utils.CheckError(err)

	oc.setChannel(channel, channelType)
	oc.rni.SendRicochetPacket(oc.conn, 0, data)
}

// AckOpenChannel acknowledges a previously received open channel message
// Prerequisites:
//             * Must have previously connected and authenticated to a service
func (oc *OpenConnection) AckOpenChannel(channel int32, channeltype string) {
	defer utils.RecoverFromError()
	messageBuilder := new(MessageBuilder)

	data, err := messageBuilder.AckOpenChannel(channel)
	utils.CheckError(err)

	oc.setChannel(channel, channeltype)
	oc.rni.SendRicochetPacket(oc.conn, 0, data)
}

// RejectOpenChannel acknowledges a rejects a previously received open channel message
// Prerequisites:
//             * Must have previously connected
func (oc *OpenConnection) RejectOpenChannel(channel int32, errortype string) {
	defer utils.RecoverFromError()
	messageBuilder := new(MessageBuilder)
	data, err := messageBuilder.RejectOpenChannel(channel, errortype)
	utils.CheckError(err)

	oc.rni.SendRicochetPacket(oc.conn, 0, data)
}

// SendContactRequest initiates a contact request to the server.
// Prerequisites:
//             * Must have previously connected and authenticated to a service
func (oc *OpenConnection) SendContactRequest(channel int32, nick string, message string) {
	defer utils.RecoverFromError()

	messageBuilder := new(MessageBuilder)
	data, err := messageBuilder.OpenContactRequestChannel(channel, nick, message)
	utils.CheckError(err)

	oc.setChannel(channel, "im.ricochet.contact.request")
	oc.rni.SendRicochetPacket(oc.conn, 0, data)
}

// AckContactRequestOnResponse responds a contact request from a client
// Prerequisites:
//             * Must have previously connected and authenticated to a service
//             * Must have previously received a Contact Request
func (oc *OpenConnection) AckContactRequestOnResponse(channel int32, status string) {
	defer utils.RecoverFromError()

	messageBuilder := new(MessageBuilder)
	data, err := messageBuilder.ReplyToContactRequestOnResponse(channel, status)
	utils.CheckError(err)

	oc.setChannel(channel, "im.ricochet.contact.request")
	oc.rni.SendRicochetPacket(oc.conn, 0, data)
}

// AckContactRequest responds to contact request from a client
// Prerequisites:
//             * Must have previously connected and authenticated to a service
//             * Must have previously received a Contact Request
func (oc *OpenConnection) AckContactRequest(channel int32, status string) {
	defer utils.RecoverFromError()

	messageBuilder := new(MessageBuilder)
	data, err := messageBuilder.ReplyToContactRequest(channel, status)
	utils.CheckError(err)

	oc.setChannel(channel, "im.ricochet.contact.request")
	oc.rni.SendRicochetPacket(oc.conn, channel, data)
}

// AckChatMessage acknowledges a previously received chat message.
// Prerequisites:
//             * Must have previously connected and authenticated to a service
//             * Must have established a known contact status with the other service
//             * Must have received a Chat message on an open im.ricochet.chat channel with the messageID
func (oc *OpenConnection) AckChatMessage(channel int32, messageID int32) {
	defer utils.RecoverFromError()

	messageBuilder := new(MessageBuilder)
	data, err := messageBuilder.AckChatMessage(messageID)
	utils.CheckError(err)

	oc.rni.SendRicochetPacket(oc.conn, channel, data)
}

// SendMessage sends a Chat Message (message) to a give Channel (channel).
// Prerequisites:
//             * Must have previously connected and authenticated to a service
//             * Must have established a known contact status with the other service
//             * Must have previously opened channel with OpenChanel of type im.ricochet.chat
func (oc *OpenConnection) SendMessage(channel int32, message string) {
	defer utils.RecoverFromError()
	messageBuilder := new(MessageBuilder)
	data, err := messageBuilder.ChatMessage(message, 0)
	utils.CheckError(err)
	oc.rni.SendRicochetPacket(oc.conn, channel, data)
}
