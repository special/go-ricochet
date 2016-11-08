package goricochet

import (
	"errors"
	"github.com/golang/protobuf/proto"
	"github.com/s-rah/go-ricochet/auth"
	"github.com/s-rah/go-ricochet/chat"
	"github.com/s-rah/go-ricochet/contact"
	"github.com/s-rah/go-ricochet/control"
	"github.com/s-rah/go-ricochet/utils"
	"io"
	"log"
	"net"
	"strconv"
)

// Ricochet is a protocol to conducting anonymous IM.
type Ricochet struct {
	newconns        chan *OpenConnection
	networkResolver utils.NetworkResolver
	rni             utils.RicochetNetworkInterface
}

// Init sets up the Ricochet object.
func (r *Ricochet) Init() {
	r.newconns = make(chan *OpenConnection)
	r.networkResolver = utils.NetworkResolver{}
	r.rni = new(utils.RicochetNetwork)
}

// Connect sets up a client ricochet connection to host e.g. qn6uo4cmsrfv4kzq.onion. If this
// function finished successfully then the connection can be assumed to
// be open and authenticated.
// To specify a local port using the format "127.0.0.1:[port]|ricochet-id".
func (r *Ricochet) Connect(host string) (*OpenConnection, error) {
	var err error
	conn, host, err := r.networkResolver.Resolve(host)

	if err != nil {
		return nil, err
	}

	return r.ConnectOpen(conn, host)
}

func (r *Ricochet) ConnectOpen(conn net.Conn, host string) (*OpenConnection, error) {
	oc, err := r.negotiateVersion(conn, true)
	if err != nil {
		return nil, err
	}
	oc.OtherHostname = host
	r.newconns <- oc
	return oc, nil
}

// Server launches a new server listening on port
func (r *Ricochet) Server(service RicochetService, port int) {
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		log.Printf("Cannot Listen on Port %v", port)
		return
	}

	r.ServeListener(service, ln)
}

func (r *Ricochet) ServeListener(service RicochetService, ln net.Listener) {
	go r.ProcessMessages(service)
	service.OnReady()
	for {
		// accept connection on port
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go r.processNewConnection(conn, service)
	}
}

// processNewConnection sets up a new connection
func (r *Ricochet) processNewConnection(conn net.Conn, service RicochetService) {
	oc, err := r.negotiateVersion(conn, false)
	if err == nil {
		r.newconns <- oc
		service.OnConnect(oc)
	}
}

// ProcessMessages is intended to be a background thread listening for all messages
// a client will send. The given RicochetService will be used to respond to messages.
// Prerequisites:
//             * Must have previously issued a successful Connect()
func (r *Ricochet) ProcessMessages(service RicochetService) {
	for {
		oc := <-r.newconns
		if oc == nil {
			return
		}
		go r.processConnection(oc, service)
	}
}

// Request that the ProcessMessages loop is stopped after handling all currently
// queued new connections.
func (r *Ricochet) RequestStopMessageLoop() {
	r.newconns <- nil
}

// ProcessConnection starts a blocking process loop which continually waits for
// new messages to arrive from the connection and uses the given RicochetService
// to process them.
func (r *Ricochet) processConnection(oc *OpenConnection, service RicochetService) {
	service.OnConnect(oc)
	defer service.OnDisconnect(oc)

	for {
		if oc.Closed {
			return
		}

		packet, err := r.rni.RecvRicochetPacket(oc.conn)
		if err != nil {
			oc.Close()
			return
		}

		if len(packet.Data) == 0 {
			service.OnChannelClosed(oc, packet.Channel)
			continue
		}

		if packet.Channel == 0 {

			res := new(Protocol_Data_Control.Packet)
			err := proto.Unmarshal(packet.Data[:], res)

			if err != nil {
				service.OnGenericError(oc, packet.Channel)
				continue
			}

			if res.GetOpenChannel() != nil {
				opm := res.GetOpenChannel()

				if oc.GetChannelType(opm.GetChannelIdentifier()) != "none" {
					// Channel is already in use.
					service.OnBadUsageError(oc, opm.GetChannelIdentifier())
					continue
				}

				// If I am a Client, the server can only open even numbered channels
				if oc.Client && opm.GetChannelIdentifier()%2 != 0 {
					service.OnBadUsageError(oc, opm.GetChannelIdentifier())
					continue
				}

				// If I am a Server, the client can only open odd numbered channels
				if !oc.Client && opm.GetChannelIdentifier()%2 != 1 {
					service.OnBadUsageError(oc, opm.GetChannelIdentifier())
					continue
				}

				switch opm.GetChannelType() {
				case "im.ricochet.auth.hidden-service":
					if oc.Client {
						// Servers are authed by default and can't auth with hidden-service
						service.OnBadUsageError(oc, opm.GetChannelIdentifier())
					} else if oc.IsAuthed {
						// Can't auth if already authed
						service.OnBadUsageError(oc, opm.GetChannelIdentifier())
					} else if oc.HasChannel("im.ricochet.auth.hidden-service") {
						// Can't open more than 1 auth channel
						service.OnBadUsageError(oc, opm.GetChannelIdentifier())
					} else {
						clientCookie, err := proto.GetExtension(opm, Protocol_Data_AuthHiddenService.E_ClientCookie)
						if err == nil {
							clientCookieB := [16]byte{}
							copy(clientCookieB[:], clientCookie.([]byte)[:])
							service.OnAuthenticationRequest(oc, opm.GetChannelIdentifier(), clientCookieB)
						} else {
							// Must include Client Cookie
							service.OnBadUsageError(oc, opm.GetChannelIdentifier())
						}
					}
				case "im.ricochet.chat":
					if !oc.IsAuthed {
						// Can't open chat channel if not authorized
						service.OnUnauthorizedError(oc, opm.GetChannelIdentifier())
					} else if !service.IsKnownContact(oc.OtherHostname) {
						// Can't open chat channel if not a known contact
						service.OnUnauthorizedError(oc, opm.GetChannelIdentifier())
					} else {
						service.OnOpenChannelRequest(oc, opm.GetChannelIdentifier(), "im.ricochet.chat")
					}
				case "im.ricochet.contact.request":
					if oc.Client {
						// Servers are not allowed to send contact requests
						service.OnBadUsageError(oc, opm.GetChannelIdentifier())
					} else if !oc.IsAuthed {
						// Can't open a contact channel if not authed
						service.OnUnauthorizedError(oc, opm.GetChannelIdentifier())
					} else if oc.HasChannel("im.ricochet.contact.request") {
						// Only 1 contact channel is allowed to be open at a time
						service.OnBadUsageError(oc, opm.GetChannelIdentifier())
					} else {
						contactRequestI, err := proto.GetExtension(opm, Protocol_Data_ContactRequest.E_ContactRequest)
						if err == nil {
							contactRequest, check := contactRequestI.(*Protocol_Data_ContactRequest.ContactRequest)
							if check {
								service.OnContactRequest(oc, opm.GetChannelIdentifier(), contactRequest.GetNickname(), contactRequest.GetMessageText())
								break
							}
						}
						service.OnBadUsageError(oc, opm.GetChannelIdentifier())
					}
				default:
					service.OnUnknownTypeError(oc, opm.GetChannelIdentifier())
				}
			} else if res.GetChannelResult() != nil {
				crm := res.GetChannelResult()
				if crm.GetOpened() {
					switch oc.GetChannelType(crm.GetChannelIdentifier()) {
					case "im.ricochet.auth.hidden-service":
						serverCookie, err := proto.GetExtension(crm, Protocol_Data_AuthHiddenService.E_ServerCookie)
						if err == nil {
							serverCookieB := [16]byte{}
							copy(serverCookieB[:], serverCookie.([]byte)[:])
							service.OnAuthenticationChallenge(oc, crm.GetChannelIdentifier(), serverCookieB)
						} else {
							service.OnBadUsageError(oc, crm.GetChannelIdentifier())
						}
					case "im.ricochet.chat":
						service.OnOpenChannelRequestSuccess(oc, crm.GetChannelIdentifier())
					case "im.ricochet.contact.request":
						responseI, err := proto.GetExtension(res.GetChannelResult(), Protocol_Data_ContactRequest.E_Response)
						if err == nil {
							response, check := responseI.(*Protocol_Data_ContactRequest.Response)
							if check {
								service.OnContactRequestAck(oc, crm.GetChannelIdentifier(), response.GetStatus().String())
								break
							}
						}
						service.OnBadUsageError(oc, crm.GetChannelIdentifier())
					default:
						service.OnBadUsageError(oc, crm.GetChannelIdentifier())
					}
				} else {
					if oc.GetChannelType(crm.GetChannelIdentifier()) != "none" {
						service.OnFailedChannelOpen(oc, crm.GetChannelIdentifier(), crm.GetCommonError().String())
					} else {
						oc.CloseChannel(crm.GetChannelIdentifier())
					}
				}
			} else {
				// Unknown Message
				oc.CloseChannel(packet.Channel)
			}
		} else if oc.GetChannelType(packet.Channel) == "im.ricochet.auth.hidden-service" {
			res := new(Protocol_Data_AuthHiddenService.Packet)
			err := proto.Unmarshal(packet.Data[:], res)

			if err != nil {
				oc.CloseChannel(packet.Channel)
				continue
			}

			if res.GetProof() != nil && !oc.Client { // Only Clients Send Proofs
				service.OnAuthenticationProof(oc, packet.Channel, res.GetProof().GetPublicKey(), res.GetProof().GetSignature(), service.IsKnownContact(oc.OtherHostname))
			} else if res.GetResult() != nil && oc.Client { // Only Servers Send Results
				service.OnAuthenticationResult(oc, packet.Channel, res.GetResult().GetAccepted(), res.GetResult().GetIsKnownContact())
			} else {
				// If neither of the above are satisfied we just close the connection
				oc.Close()
			}

		} else if oc.GetChannelType(packet.Channel) == "im.ricochet.chat" {

			// NOTE: These auth checks should be redundant, however they
			// are included here for defense-in-depth if for some reason
			// a previously authed connection becomes untrusted / not known and
			// the state is not cleaned up.
			if !oc.IsAuthed {
				// Can't send chat messages if not authorized
				service.OnUnauthorizedError(oc, packet.Channel)
			} else if !service.IsKnownContact(oc.OtherHostname) {
				// Can't send chat message if not a known contact
				service.OnUnauthorizedError(oc, packet.Channel)
			} else {
				res := new(Protocol_Data_Chat.Packet)
				err := proto.Unmarshal(packet.Data[:], res)

				if err != nil {
					oc.CloseChannel(packet.Channel)
					continue
				}

				if res.GetChatMessage() != nil {
					service.OnChatMessage(oc, packet.Channel, int32(res.GetChatMessage().GetMessageId()), res.GetChatMessage().GetMessageText())
				} else if res.GetChatAcknowledge() != nil {
					service.OnChatMessageAck(oc, packet.Channel, int32(res.GetChatMessage().GetMessageId()))
				} else {
					// If neither of the above are satisfied we just close the connection
					oc.Close()
				}
			}
		} else if oc.GetChannelType(packet.Channel) == "im.ricochet.contact.request" {

			// NOTE: These auth checks should be redundant, however they
			// are included here for defense-in-depth if for some reason
			// a previously authed connection becomes untrusted / not known and
			// the state is not cleaned up.
			if !oc.Client {
				// Clients are not allowed to send contact request responses
				service.OnBadUsageError(oc, packet.Channel)
			} else if !oc.IsAuthed {
				// Can't send a contact request if not authed
				service.OnBadUsageError(oc, packet.Channel)
			} else {
				res := new(Protocol_Data_ContactRequest.Response)
				err := proto.Unmarshal(packet.Data[:], res)
				log.Printf("%v", res)
				if err != nil {
					oc.CloseChannel(packet.Channel)
					continue
				}
				service.OnContactRequestAck(oc, packet.Channel, res.GetStatus().String())
			}
		} else if oc.GetChannelType(packet.Channel) == "none" {
			// Invalid Channel Assignment
			oc.CloseChannel(packet.Channel)
		} else {
			oc.Close()
		}
	}
}

// Perform version negotiation on the connection, and create an OpenConnection if successful
func (r *Ricochet) negotiateVersion(conn net.Conn, outbound bool) (*OpenConnection, error) {
	versions := []byte{0x49, 0x4D, 0x01, 0x01}

	// Outbound side of the connection sends a list of supported versions
	if outbound {
		if n, err := conn.Write(versions); err != nil || n < len(versions) {
			return nil, err
		}

		res := make([]byte, 1)
		if _, err := io.ReadAtLeast(conn, res, len(res)); err != nil {
			return nil, err
		}

		if res[0] != 0x01 {
			return nil, errors.New("unsupported protocol version")
		}
	} else {
		// Read version response header
		header := make([]byte, 3)
		if _, err := io.ReadAtLeast(conn, header, len(header)); err != nil {
			return nil, err
		}

		if header[0] != versions[0] || header[1] != versions[1] || header[2] < 1 {
			return nil, errors.New("invalid protocol response")
		}

		// Read list of supported versions (which is header[2] bytes long)
		versionList := make([]byte, header[2])
		if _, err := io.ReadAtLeast(conn, versionList, len(versionList)); err != nil {
			return nil, err
		}

		selectedVersion := byte(0xff)
		for _, v := range versionList {
			if v == 0x01 {
				selectedVersion = v
				break
			}
		}

		if n, err := conn.Write([]byte{selectedVersion}); err != nil || n < 1 {
			return nil, err
		}

		if selectedVersion == 0xff {
			return nil, errors.New("no supported protocol version")
		}
	}

	oc := new(OpenConnection)
	oc.Init(outbound, conn)
	return oc, nil
}
