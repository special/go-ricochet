package connection

import (
	"errors"
	"github.com/golang/protobuf/proto"
	"github.com/s-rah/go-ricochet/channels"
	"github.com/s-rah/go-ricochet/utils"
	"github.com/s-rah/go-ricochet/wire/control"
	"io"
	"log"
	"time"
)

// Connection encapsulates the state required to maintain a connection to
// a ricochet service.
type Connection struct {
	utils.RicochetNetwork

	channelManager *ChannelManager

	// Ricochet Network Loop
	packetChannel chan utils.RicochetData
	errorChannel  chan error

	breakChannel       chan bool
	breakResultChannel chan bool

	unlockChannel         chan bool
	unlockResponseChannel chan bool

	messageBuilder utils.MessageBuilder

	Conn           io.ReadWriteCloser
	IsInbound      bool
	Authentication map[string]bool
	RemoteHostname string
}

func (rc *Connection) init() {

	rc.packetChannel = make(chan utils.RicochetData)
	rc.errorChannel = make(chan error)

	rc.breakChannel = make(chan bool)
	rc.breakResultChannel = make(chan bool)

	rc.unlockChannel = make(chan bool)
	rc.unlockResponseChannel = make(chan bool)

	rc.Authentication = make(map[string]bool)
	go rc.start()
}

// NewInboundConnection creates a new Connection struct
// modelling an Inbound Connection
func NewInboundConnection(conn io.ReadWriteCloser) *Connection {
	rc := new(Connection)
	rc.Conn = conn
	rc.IsInbound = true
	rc.init()
	rc.channelManager = NewServerChannelManager()
	return rc
}

// NewOutboundConnection creates a new Connection struct
// modelling an Inbound Connection
func NewOutboundConnection(conn io.ReadWriteCloser, remoteHostname string) *Connection {
	rc := new(Connection)
	rc.Conn = conn
	rc.IsInbound = false
	rc.init()
	rc.RemoteHostname = remoteHostname
	rc.channelManager = NewClientChannelManager()
	return rc
}

// start
func (rc *Connection) start() {
	for {
		packet, err := rc.RecvRicochetPacket(rc.Conn)
		if err != nil {
			rc.errorChannel <- err
			return
		}
		rc.packetChannel <- packet
	}
}

// Do allows any function utilizing Connection to be run safetly.
// All operations which require access to Connection managed resources should
// use Do()
func (rc *Connection) Do(do func() error) error {
	// Force process to soft-break so we can lock
	log.Printf("UnLocking Processloop")
	rc.unlockChannel <- true
	log.Printf("Unlocked Processloop")
	ret := do()
	log.Printf("Giving up lock Processloop")
	rc.unlockResponseChannel <- true
	return ret
}

// RequestOpenChannel sends an OpenChannel message to the remote client.
// and error is returned only if the requirements for opening this channel
// are not met on the local side (a nill error return does not mean the
// channel was opened successfully)
func (rc *Connection) RequestOpenChannel(ctype string, handler Handler) error {
	return rc.Do(func() error {
		chandler, err := handler.OnOpenChannelRequest(ctype)

		if err != nil {
			return err
		}

		// Check that we have the authentication already
		if chandler.RequiresAuthentication() != "none" {
			// Enforce Authentication Check.
			_, authed := rc.Authentication[chandler.RequiresAuthentication()]
			if !authed {
				return errors.New("connection is not auth'd")
			}
		}

		channel, err := rc.channelManager.OpenChannelRequest(chandler)

		if err != nil {
			return err
		}

		channel.SendMessage = func(message []byte) {
			rc.SendRicochetPacket(rc.Conn, channel.ID, message)
		}
		channel.DelegateAuthorization = func() {
			rc.Authentication[chandler.Type()] = true
		}
		channel.CloseChannel = func() {
			rc.SendRicochetPacket(rc.Conn, channel.ID, []byte{})
			rc.channelManager.RemoveChannel(channel.ID)
		}
		response, err := chandler.OpenOutbound(channel)
		if err == nil {
			rc.SendRicochetPacket(rc.Conn, 0, response)
		} else {
			rc.channelManager.RemoveChannel(channel.ID)
		}
		return nil
	})
}

// Process receives socket and protocol events for the connection. Methods
// of the application-provided `handler` will be called from this goroutine
// for all events.
//
// Process must be running in order to handle any events on the connection,
// including connection close.
//
// Process blocks until the connection is closed or until Break() is called.
// If the connection is closed, a non-nil error is returned.
func (rc *Connection) Process(handler Handler) error {
	log.Printf("Entering Processloop")
	handler.OnReady(rc)
	breaked := false
	for !breaked {

		var packet utils.RicochetData
		tick := time.Tick(30 * time.Second)
		select {
		case <-rc.unlockChannel:
			<-rc.unlockResponseChannel
			continue
		case <-rc.breakChannel:
			log.Printf("Process has Ended as Expected!!!")
			breaked = true
			continue
		case packet = <-rc.packetChannel:
			break
		case err := <-rc.errorChannel:
			rc.Conn.Close()
			handler.OnClosed(err)
			return err
		case <-tick:
			log.Printf("timeout")
			return errors.New("peer timed out")
		}

		log.Printf("Received Packet on Channel %d", packet.Channel)

		if packet.Channel == 0 {
			res := new(Protocol_Data_Control.Packet)
			err := proto.Unmarshal(packet.Data[:], res)
			if err == nil {
				rc.controlPacket(handler, res)
			}
		} else {
			// Let's check to see if we have defined this channel.
			channel, found := rc.channelManager.GetChannel(packet.Channel)
			if found {
				if len(packet.Data) == 0 {
					rc.channelManager.RemoveChannel(packet.Channel)
					(*channel.Handler).Closed(errors.New("channel closed by peer"))
				} else {
					// Send The Ricochet Packet to the Handler
					(*channel.Handler).Packet(packet.Data[:])
				}
			} else {
				// When a non-zero packet is received for an unknown
				// channel, the recipient responds by closing
				// that channel.
				if len(packet.Data) != 0 {
					rc.SendRicochetPacket(rc.Conn, packet.Channel, []byte{})
				}
			}
		}
	}

	rc.breakResultChannel <- true
	return nil

}

func (rc *Connection) controlPacket(handler Handler, res *Protocol_Data_Control.Packet) {

	if res.GetOpenChannel() != nil {

		opm := res.GetOpenChannel()
		chandler, err := handler.OnOpenChannelRequest(opm.GetChannelType())

		if err != nil {

			response := rc.messageBuilder.RejectOpenChannel(opm.GetChannelIdentifier(), "UnknownTypeError")
			rc.SendRicochetPacket(rc.Conn, 0, response)
			return
		}

		// Check that we have the authentication already
		if chandler.RequiresAuthentication() != "none" {
			// Enforce Authentication Check.
			_, authed := rc.Authentication[chandler.RequiresAuthentication()]
			if !authed {
				rc.SendRicochetPacket(rc.Conn, 0, []byte{})
				return
			}
		}

		channel, err := rc.channelManager.OpenChannelRequestFromPeer(opm.GetChannelIdentifier(), chandler)

		if err == nil {

			channel.SendMessage = func(message []byte) {
				rc.SendRicochetPacket(rc.Conn, channel.ID, message)
			}
			channel.DelegateAuthorization = func() {
				rc.Authentication[chandler.Type()] = true
			}
			channel.CloseChannel = func() {
				rc.SendRicochetPacket(rc.Conn, channel.ID, []byte{})
				rc.channelManager.RemoveChannel(channel.ID)
			}

			response, err := chandler.OpenInbound(channel, opm)
			if err == nil && channel.Pending == false {
				log.Printf("Opening Channel %v on %v", channel.Type, channel.ID)
				rc.SendRicochetPacket(rc.Conn, 0, response)
			} else {
				rc.channelManager.RemoveChannel(channel.ID)
				rc.SendRicochetPacket(rc.Conn, 0, []byte{})
			}
		} else {
			// Send Error Packet
			response := rc.messageBuilder.RejectOpenChannel(opm.GetChannelIdentifier(), "GenericError")
			rc.SendRicochetPacket(rc.Conn, 0, response)

		}
	} else if res.GetChannelResult() != nil {
		cr := res.GetChannelResult()
		id := cr.GetChannelIdentifier()

		channel, found := rc.channelManager.GetChannel(id)

		if !found {
			return
		}

		if cr.GetOpened() {
			(*channel.Handler).OpenOutboundResult(nil, cr)
		} else {
			(*channel.Handler).OpenOutboundResult(errors.New(""), cr)
		}

	} else if res.GetKeepAlive() != nil {
		// XXX Though not currently part of the protocol
		// We should likely put these calls behind
		// authentication.
		if res.GetKeepAlive().GetResponseRequested() {
			messageBuilder := new(utils.MessageBuilder)
			raw := messageBuilder.KeepAlive(true)
			rc.SendRicochetPacket(rc.Conn, 0, raw)
		}
	} else if res.GetEnableFeatures() != nil {
		// TODO Respond with an Empty List
		messageBuilder := new(utils.MessageBuilder)
		raw := messageBuilder.FeaturesEnabled([]string{})
		rc.SendRicochetPacket(rc.Conn, 0, raw)
	} else if res.GetFeaturesEnabled() != nil {
		// TODO We should never send out an enabled features
		// request.
	}
}

// Break causes Process() to return, but does not close the underlying connection
func (rc *Connection) Break() {
	log.Printf("breaking...")
	rc.breakChannel <- true
	<-rc.breakResultChannel // Wait for Process to End
}

// Channel is a convienciance method for returning a given channel to the caller
// of Process() - TODO - this is kind of ugly.
func (rc *Connection) Channel(ctype string, way channels.Direction) *channels.Channel {
	return rc.channelManager.Channel(ctype, way)
}
