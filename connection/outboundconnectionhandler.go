package connection

import (
	"crypto/rsa"
	"github.com/s-rah/go-ricochet/channels"
	"github.com/s-rah/go-ricochet/policies"
	"github.com/s-rah/go-ricochet/utils"
)

// OutboundConnectionHandler is a convieniance wrapper for handling outbound
// connections
type OutboundConnectionHandler struct {
	connection *Connection
}

// HandleOutboundConnection returns an OutboundConnectionHandler given a connection
func HandleOutboundConnection(c *Connection) *OutboundConnectionHandler {
	och := new(OutboundConnectionHandler)
	och.connection = c
	return och
}

// ProcessAuthAsClient blocks until authentication has succeeded or failed with the
// provided privateKey, or the connection is closed. A non-nil error is returned in all
// cases other than successful authentication.
//
// ProcessAuthAsClient cannot be called at the same time as any other call to a Porcess
// function. Another Process function must be called after this function successfully
// returns to continue handling connection events.
//
// For successful authentication, the `known` return value indicates whether the peer
// accepts us as a known contact. Unknown contacts will generally need to send a contact
// request before any other activity.
func (och *OutboundConnectionHandler) ProcessAuthAsClient(privateKey *rsa.PrivateKey) (bool, error) {

	if privateKey == nil {
		return false, utils.PrivateKeyNotSetError
	}

	ach := new(AutoConnectionHandler)
	ach.Init(privateKey, och.connection.RemoteHostname)

	var result channels.AuthChannelResult
	go func() {
		err := och.connection.RequestOpenChannel("im.ricochet.auth.hidden-service", ach)
		if err != nil {
			return
		}
		result = ach.WaitForAuthenticationEvent()
		och.connection.Break()
	}()

	policy := policies.UnknownPurposeTimeout
	err := policy.ExecuteAction(func() error {
		return och.connection.Process(ach)
	})

	if err == nil {
		if result.Accepted == true {
			return result.IsKnownContact, nil
		}
	}
	return false, utils.ServerRejectedClientConnectionError
}
