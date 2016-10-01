package goricochet

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"github.com/s-rah/go-ricochet/utils"
	"io/ioutil"
	"log"
)

// StandardRicochetService implements all the necessary flows to implement a
// minimal, protocol compliant Ricochet Service. It can be built on by other
// applications to produce automated riochet applications.
type StandardRicochetService struct {
	ricochet       *Ricochet
	privateKey     *rsa.PrivateKey
	serverHostname string
}

// Init initializes a StandardRicochetService with the cryptographic key given
// by filename.
func (srs *StandardRicochetService) Init(filename string) error {
	srs.ricochet = new(Ricochet)
	srs.ricochet.Init()

	pemData, err := ioutil.ReadFile(filename)

	if err != nil {
		return errors.New("Could not setup ricochet service: could not read private key")
	}

	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return errors.New("Could not setup ricochet service: no valid PEM data found")
	}

	srs.privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return errors.New("Could not setup ricochet service: could not parse private key")
	}

	publicKeyBytes, _ := asn1.Marshal(rsa.PublicKey{
		N: srs.privateKey.PublicKey.N,
		E: srs.privateKey.PublicKey.E,
	})

	srs.serverHostname = utils.GetTorHostname(publicKeyBytes)
	log.Printf("Initialised ricochet service for %s", srs.serverHostname)

	return nil
}

// OnReady is called once a Server has been established (by calling Listen)
func (srs *StandardRicochetService) OnReady() {
}

// Listen starts the ricochet service. Listen must be called before any other method (apart from Init)
func (srs *StandardRicochetService) Listen(service RicochetService, port int) {
	srs.ricochet.Server(service, port)
}

// Connect can be called to initiate a new client connection to a server
func (srs *StandardRicochetService) Connect(hostname string) error {
	log.Printf("Connecting to...%s", hostname)
	oc, err := srs.ricochet.Connect(hostname)
	if err != nil {
		return errors.New("Could not connect to: " + hostname)
	}
	oc.MyHostname = srs.serverHostname
	return nil
}

// OnConnect is called when a client or server sucessfully passes Version Negotiation.
func (srs *StandardRicochetService) OnConnect(oc *OpenConnection) {
	if oc.Client {
		log.Printf("Sucessefully Connected to %s", oc.OtherHostname)
		oc.IsAuthed = true // Connections to Servers are Considered Authenticated by Default
		oc.Authenticate(1)
	} else {
		oc.MyHostname = srs.serverHostname
	}
}

// OnDisconnect is called when a connection is closed
func (srs *StandardRicochetService) OnDisconnect(oc *OpenConnection) {
}

// OnAuthenticationRequest is called when a client requests Authentication
func (srs *StandardRicochetService) OnAuthenticationRequest(oc *OpenConnection, channelID int32, clientCookie [16]byte) {
	oc.ConfirmAuthChannel(channelID, clientCookie)
}

// OnAuthenticationChallenge constructs a valid authentication challenge to the serverCookie
func (srs *StandardRicochetService) OnAuthenticationChallenge(oc *OpenConnection, channelID int32, serverCookie [16]byte) {
	// DER Encode the Public Key
	publickeyBytes, _ := asn1.Marshal(rsa.PublicKey{
		N: srs.privateKey.PublicKey.N,
		E: srs.privateKey.PublicKey.E,
	})
	oc.SendProof(1, serverCookie, publickeyBytes, srs.privateKey)
}

// OnAuthenticationProof is called when a client sends Proof for an existing authentication challenge
func (srs *StandardRicochetService) OnAuthenticationProof(oc *OpenConnection, channelID int32, publicKey []byte, signature []byte, isKnownContact bool) {
	result := oc.ValidateProof(channelID, publicKey, signature)
	oc.SendAuthenticationResult(channelID, result, isKnownContact)
	oc.IsAuthed = result
	oc.CloseChannel(channelID)
}

// OnAuthenticationResult is called once a server has returned the result of the Proof Verification
func (srs *StandardRicochetService) OnAuthenticationResult(oc *OpenConnection, channelID int32, result bool, isKnownContact bool) {
	oc.IsAuthed = result
}

// IsKnownContact allows a caller to determine if a hostname an authorized contact.
func (srs *StandardRicochetService) IsKnownContact(hostname string) bool {
	return false
}

// OnContactRequest is called when a client sends a new contact request
func (srs *StandardRicochetService) OnContactRequest(oc *OpenConnection, channelID int32, nick string, message string) {
}

// OnContactRequestAck is called when a server sends a reply to an existing contact request
func (srs *StandardRicochetService) OnContactRequestAck(oc *OpenConnection, channelID int32, status string) {
}

// OnOpenChannelRequest is called when a client or server requests to open a new channel
func (srs *StandardRicochetService) OnOpenChannelRequest(oc *OpenConnection, channelID int32, channelType string) {
	oc.AckOpenChannel(channelID, channelType)
}

// OnOpenChannelRequestSuccess is called when a client or server responds to an open channel request
func (srs *StandardRicochetService) OnOpenChannelRequestSuccess(oc *OpenConnection, channelID int32) {
}

// OnChannelClose is called when a client or server closes an existing channel
func (srs *StandardRicochetService) OnChannelClosed(oc *OpenConnection, channelID int32) {
}

// OnChatMessage is called when a new chat message is received.
func (srs *StandardRicochetService) OnChatMessage(oc *OpenConnection, channelID int32, messageID int32, message string) {
	oc.AckChatMessage(channelID, messageID)
}

// OnChatMessageAck is called when a new chat message is ascknowledged.
func (srs *StandardRicochetService) OnChatMessageAck(oc *OpenConnection, channelID int32, messageID int32) {
}

// OnFailedChannelOpen is called when a server fails to open a channel
func (srs *StandardRicochetService) OnFailedChannelOpen(oc *OpenConnection, channelID int32, errorType string) {
	oc.UnsetChannel(channelID)
}

// OnGenericError is called when a generalized error is returned from the peer
func (srs *StandardRicochetService) OnGenericError(oc *OpenConnection, channelID int32) {
	oc.RejectOpenChannel(channelID, "GenericError")
}

//OnUnknownTypeError is called when an unknown type error is returned from the peer
func (srs *StandardRicochetService) OnUnknownTypeError(oc *OpenConnection, channelID int32) {
	oc.RejectOpenChannel(channelID, "UnknownTypeError")
}

// OnUnauthorizedError is called when an unathorized error is returned from the peer
func (srs *StandardRicochetService) OnUnauthorizedError(oc *OpenConnection, channelID int32) {
	oc.RejectOpenChannel(channelID, "UnauthorizedError")
}

// OnBadUsageError is called when a bad usage error is returned from the peer
func (srs *StandardRicochetService) OnBadUsageError(oc *OpenConnection, channelID int32) {
	oc.RejectOpenChannel(channelID, "BadUsageError")
}

// OnFailedError is called when a failed error is returned from the peer
func (srs *StandardRicochetService) OnFailedError(oc *OpenConnection, channelID int32) {
	oc.RejectOpenChannel(channelID, "FailedError")
}
