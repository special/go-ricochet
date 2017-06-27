package connection

import (
	"crypto/rsa"
	"github.com/s-rah/go-ricochet/channels"
	"github.com/s-rah/go-ricochet/utils"
	"log"
)

// AutoConnectionHandler implements the ConnectionHandler interface on behalf of
// the provided application type by automatically providing support for any
// built-in channel type whose high level interface is implemented by the
// application. For example, if the application's type implements the
// ChatChannelHandler interface, `im.ricochet.chat` will be available to the peer.
//
// The application handler can be any other type. To override or augment any of
// AutoConnectionHandler's behavior (such as adding new channel types, or reacting
// to connection close events), this type can be embedded in the type that it serves.
type AutoConnectionHandler struct {
	handlerMap        map[string]func() channels.Handler
	connection        *Connection
	authResultChannel chan channels.AuthChannelResult
	sach              func(hostname string, publicKey rsa.PublicKey) (allowed, known bool)
}

// Init ...
// TODO: Split this into client and server init
func (ach *AutoConnectionHandler) Init(privateKey *rsa.PrivateKey, serverHostname string) {

	ach.handlerMap = make(map[string]func() channels.Handler)
	ach.RegisterChannelHandler("im.ricochet.auth.hidden-service", func() channels.Handler {
		hsau := new(channels.HiddenServiceAuthChannel)
		hsau.PrivateKey = privateKey
		hsau.Handler = ach
		hsau.ServerHostname = serverHostname
		return hsau
	})
	ach.authResultChannel = make(chan channels.AuthChannelResult)
}

// SetServerAuthHandler ...
func (ach *AutoConnectionHandler) SetServerAuthHandler(sach func(hostname string, publicKey rsa.PublicKey) (allowed, known bool)) {
	ach.sach = sach
}

// OnReady ...
func (ach *AutoConnectionHandler) OnReady(oc *Connection) {
	ach.connection = oc
}

// OnClosed is called when the OpenConnection has closed for any reason.
func (ach *AutoConnectionHandler) OnClosed(err error) {
}

// WaitForAuthenticationEvent ...
func (ach *AutoConnectionHandler) WaitForAuthenticationEvent() channels.AuthChannelResult {
	return <-ach.authResultChannel
}

// ClientAuthResult ...
func (ach *AutoConnectionHandler) ClientAuthResult(accepted bool, isKnownContact bool) {
	log.Printf("Got auth result %v %v", accepted, isKnownContact)
	ach.authResultChannel <- channels.AuthChannelResult{Accepted: accepted, IsKnownContact: isKnownContact}
}

// ServerAuthValid ...
func (ach *AutoConnectionHandler) ServerAuthValid(hostname string, publicKey rsa.PublicKey) (allowed, known bool) {
	// Do something
	accepted, isKnownContact := ach.sach(hostname, publicKey)
	ach.authResultChannel <- channels.AuthChannelResult{Accepted: accepted, IsKnownContact: isKnownContact}
	return accepted, isKnownContact
}

// ServerAuthInvalid ...
func (ach *AutoConnectionHandler) ServerAuthInvalid(err error) {
	ach.authResultChannel <- channels.AuthChannelResult{Accepted: false, IsKnownContact: false}
}

// RegisterChannelHandler ...
func (ach *AutoConnectionHandler) RegisterChannelHandler(ctype string, handler func() channels.Handler) {
	_, exists := ach.handlerMap[ctype]
	if !exists {
		ach.handlerMap[ctype] = handler
	}
}

// OnOpenChannelRequest ...
func (ach *AutoConnectionHandler) OnOpenChannelRequest(ctype string) (channels.Handler, error) {
	handler, ok := ach.handlerMap[ctype]
	if ok {
		h := handler()
		log.Printf("Got Channel Handler")
		return h, nil
	}
	return nil, utils.UnknownChannelTypeError
}
