package goricochet

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/s-rah/go-ricochet/auth"
	"github.com/s-rah/go-ricochet/chat"
	"github.com/s-rah/go-ricochet/control"
	"io/ioutil"
	"log"
	"net"
	"os"
)

// MessageType details the different kinds of messages used by Ricochet
type MessageType int

const (
	// CONTROL messages are those sent on channel 0
	CONTROL MessageType = iota
	// AUTH messages are those that deal with authentication
	AUTH = iota
	// DATA covers both chat and (later) file handling and other non-control messages.
	DATA = iota
)

// Ricochet is a protocol to conducting anonymous IM.
type Ricochet struct {
	conn         net.Conn
	privateKey   *rsa.PrivateKey
	logger       *log.Logger
	channelState map[int]int
	channel      chan RicochetMessage
	known        bool
}

// RicochetData is a structure containing the raw data and the channel it the
// message originated on.
type RicochetData struct {
	Channel int32
	Data    []byte
}

// RicochetMessage is a Wrapper Around Common Ricochet Protocol Strucutres
type RicochetMessage struct {
	Channel       int32
	ControlPacket *Protocol_Data_Control.Packet
	DataPacket    *Protocol_Data_Chat.Packet
	AuthPacket    *Protocol_Data_AuthHiddenService.Packet
}

func (r *Ricochet) IsKnownContact() bool {
	return r.known
}

// Init sets up the Ricochet object. It takes in a filename of a hidden service
// private_key file so it can successfully authenticate itself with other
// clients.
func (r *Ricochet) Init(filename string, debugLog bool) {

	if debugLog {
		r.logger = log.New(os.Stdout, "[Ricochet]: ", log.Ltime|log.Lmicroseconds)
	} else {
		r.logger = log.New(ioutil.Discard, "[Ricochet]: ", log.Ltime|log.Lmicroseconds)
	}

	pemData, err := ioutil.ReadFile(filename)

	if err != nil {
		r.logger.Print("Error Reading Private Key: ", err)
	}

	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		r.logger.Print("No valid PEM data found")
	}

	r.privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	r.handleFatal(err, "Private key can't be decoded")

	r.channelState = make(map[int]int)
	r.channel = make(chan RicochetMessage)
}

func (r *Ricochet) StartService(server RicochetService, port string) {
	// Listen
	ln, _ := net.Listen("tcp", port)
	conn, _ := ln.Accept()
	go r.runService(conn, server)
}

func (r *Ricochet) runService(conn net.Conn, server RicochetService) {
	// Negotiate Version

	// Loop For Messages
}

// Connect sets up a ricochet connection between from and to which are
// both ricochet formated hostnames e.g. qn6uo4cmsrfv4kzq.onion. If this
// function finished successfully then the connection can be assumed to
// be open and authenticated.
// To specify a local port using the format "127.0.0.1:[port]|ricochet-id".
func (r *Ricochet) Connect(from string, to string) error {

	var err error
	networkResolver := new(NetworkResolver)
	r.conn, to, err = networkResolver.Resolve(to)

	if err != nil {
		return err
	}

	r.negotiateVersion()

	authHandler := new(AuthenticationHandler)
	clientCookie := authHandler.GenClientCookie()

	messageBuilder := new(MessageBuilder)
	data, err := messageBuilder.OpenAuthenticationChannel(1, clientCookie)

	if err != nil {
		return errors.New("Cannot Marshal Open Channel Message")
	}

	r.sendPacket(data, 0)

	response, _ := r.getMessages()
	openChannelResponse, _ := r.decodePacket(response[0], CONTROL)
	r.logger.Print("Received Response: ", openChannelResponse)
	channelResult := openChannelResponse.ControlPacket.GetChannelResult()

	if channelResult.GetOpened() == true {
		r.logger.Print("Channel Opened Successfully: ", channelResult.GetChannelIdentifier())
	}

	sCookie, _ := proto.GetExtension(channelResult, Protocol_Data_AuthHiddenService.E_ServerCookie)
	authHandler.AddServerCookie(sCookie.([]byte))

	// DER Encode the Public Key
	publickeybytes, err := asn1.Marshal(rsa.PublicKey{
		N: r.privateKey.PublicKey.N,
		E: r.privateKey.PublicKey.E,
	})

	signature, _ := rsa.SignPKCS1v15(nil, r.privateKey, crypto.SHA256, authHandler.GenChallenge(from, to))

	signatureBytes := make([]byte, 128)
	copy(signatureBytes[:], signature[:])

	// Construct a Proof Message
	proof := &Protocol_Data_AuthHiddenService.Proof{
		PublicKey: publickeybytes,
		Signature: signatureBytes,
	}

	ahsPacket := &Protocol_Data_AuthHiddenService.Packet{
		Proof:  proof,
		Result: nil,
	}

	data, err = proto.Marshal(ahsPacket)
	r.sendPacket(data, 1)

	response, err = r.getMessages()

	if err != nil {
		return err
	}

	resultResponse, _ := r.decodePacket(response[0], AUTH)
	r.logger.Print("Received Result: ", resultResponse)

	if resultResponse.AuthPacket.GetResult().GetAccepted() != true {
		return errors.New("authorization failed")
	}

	r.known = resultResponse.AuthPacket.GetResult().GetIsKnownContact()
	return nil
}

// OpenChannel opens a new chat channel with the given id
// Prerequisites:
//              * Must have Previously issued a successful Connect()
//              * If acting as the client, id must be odd, else even
func (r *Ricochet) OpenChatChannel(id int32) error {
	messageBuilder := new(MessageBuilder)
	data, err := messageBuilder.OpenChatChannel(id)

	if err != nil {
		return errors.New("error constructing control channel message to open channel")
	}

	r.logger.Printf("Opening Chat Channel: %d", id)
	r.sendPacket(data, 0)
	return nil
}

// SendContactRequest initiates a contact request to the server.
// Prerequisites:
//              * Must have Previously issued a successful Connect()
func (r *Ricochet) SendContactRequest(channel int32, nick string, message string) error {
	messageBuilder := new(MessageBuilder)
	data, err := messageBuilder.OpenContactRequestChannel(channel, nick, message)

	if err != nil {
		return errors.New("error constructing control channel message to send contact request")
	}

	r.sendPacket(data, 0)
	return nil
}

// SendMessage sends a Chat Message (message) to a give Channel (channel).
// Prerequisites:
//             * Must have previously issued a successful Connect()
//             * Must have previously opened channel with OpenChanel
func (r *Ricochet) SendMessage(channel int32, message string) error {
	messageBuilder := new(MessageBuilder)
	data, err := messageBuilder.ChatMessage(message)

	if err != nil {
		return errors.New("error constructing control channel message to send chat message")
	}

	r.logger.Printf("Sending Message on Channel: %d", channel)
	r.sendPacket(data, channel)
	return nil
}

// negotiateVersion Perform version negotiation with the connected host.
func (r *Ricochet) negotiateVersion() error {
	version := make([]byte, 4)
	version[0] = 0x49
	version[1] = 0x4D
	version[2] = 0x01
	version[3] = 0x01

	fmt.Fprintf(r.conn, "%s", version)
	r.logger.Print("Negotiating Version ", version)
	res, err := r.recv()

	if len(res) != 1 || err != nil {
		return errors.New("Failed Version Negotiating")
	}

	if res[0] != 1 {
		return errors.New("Failed Version Negotiating - Invalid Version ")
	}

	r.logger.Print("Successfully Negotiated Version ", res[0])
	return nil
}

// sendPacket places the data into a structure needed for the client to
// decode the packet and writes the packet to the network.
func (r *Ricochet) sendPacket(data []byte, channel int32) {
	header := make([]byte, 4+len(data))
	header[0] = byte(len(header) >> 8)
	header[1] = byte(len(header) & 0x00FF)
	header[2] = 0x00
	header[3] = byte(channel)
	copy(header[4:], data[:])

	fmt.Fprintf(r.conn, "%s", header)
}

// Listen blocks and waits for a new message to arrive from the connected user
// once a message has arrived, it returns the message and the channel it occured
// on, else it returns an error.
// Prerequisites:
//             * Must have previously issued a successful Connect()
//             * Must have previously ran "go ricochet.ListenAndWait()"
func (r *Ricochet) Listen() (string, int32, error) {
	var message RicochetMessage
	message = <-r.channel
	r.logger.Printf("Received Chat Message on Channel %d", message.Channel)
	if message.DataPacket.GetChatMessage() == nil {
		return "", 0, errors.New("Did not receive a chat message")
	}

	messageID := message.DataPacket.GetChatMessage().GetMessageId()
	cr := &Protocol_Data_Chat.ChatAcknowledge{
		MessageId: proto.Uint32(messageID),
		Accepted:  proto.Bool(true),
	}

	pc := &Protocol_Data_Chat.Packet{
		ChatAcknowledge: cr,
	}

	data, err := proto.Marshal(pc)
	if err != nil {
		return "", 0, errors.New("Failed to serialize chat message")
	}

	r.sendPacket(data, message.Channel)
	return message.DataPacket.GetChatMessage().GetMessageText(), message.Channel, nil
}

// ListenAndWait is intended to be a background thread listening for all messages
// a client will send, automaticall responding to some, and making the others available to
// Listen()
// Prerequisites:
//             * Must have previously issued a successful Connect()
func (r *Ricochet) ListenAndWait() error {
	for true {
		packets, err := r.getMessages()
		if err != nil {
			return errors.New("Error attempted to get new messages")
		}

		for _, packet := range packets {
			if packet.Channel == 0 {
				// This is a Control Channel Message
				message, err := r.decodePacket(packet, CONTROL)

				if err != nil {
					r.logger.Printf("Failed to decode control packet, discarding")
					break
				}

				// Automatically accept new channels
				if message.ControlPacket.GetOpenChannel() != nil {
					// TODO Reject if already in use.
					cr := &Protocol_Data_Control.ChannelResult{
						ChannelIdentifier: proto.Int32(message.ControlPacket.GetOpenChannel().GetChannelIdentifier()),
						Opened:            proto.Bool(true),
					}

					pc := &Protocol_Data_Control.Packet{
						ChannelResult: cr,
					}

					data, err := proto.Marshal(pc)
					// TODO we should set up some kind of error channel.
					r.handleFatal(err, "error marshalling control protocol")

					r.logger.Printf("Client Opening Channel: %d\n", message.ControlPacket.GetOpenChannel().GetChannelIdentifier())
					r.sendPacket(data, 0)
					r.channelState[int(message.ControlPacket.GetOpenChannel().GetChannelIdentifier())] = 1
					break
				}

				if message.ControlPacket.GetChannelResult() != nil {
					channelResult := message.ControlPacket.GetChannelResult()
					if channelResult.GetOpened() == true {
						r.logger.Print("Channel Opened Successfully: ", channelResult.GetChannelIdentifier())
						r.channelState[int(message.ControlPacket.GetChannelResult().GetChannelIdentifier())] = 1
					}
					break
				}

				r.logger.Printf("Received Unknown Control Message\n")

			} else if packet.Channel == 3 {
				// Contact Request
				r.logger.Printf("Received Unknown Message on Channel 3\n")
			} else {
				// At this point the only other expected type of message
				// is a Chat Message
				message, err := r.decodePacket(packet, DATA)
				if err != nil {
					r.logger.Printf("Failed to decode data packet, discarding")
					break
				}
				r.channel <- message
			}
		}
	}
	return nil
}

// decodePacket take a raw RicochetData message and decodes it based on a given MessageType
func (r *Ricochet) decodePacket(packet RicochetData, t MessageType) (rm RicochetMessage, err error) {

	rm.Channel = packet.Channel

	if t == CONTROL {
		res := new(Protocol_Data_Control.Packet)
		err = proto.Unmarshal(packet.Data[:], res)
		rm.ControlPacket = res
	} else if t == AUTH {
		res := new(Protocol_Data_AuthHiddenService.Packet)
		err = proto.Unmarshal(packet.Data[:], res)
		rm.AuthPacket = res
	} else if t == DATA {
		res := new(Protocol_Data_Chat.Packet)
		err = proto.Unmarshal(packet.Data[:], res)
		rm.DataPacket = res
	}

	if err != nil {
		return rm, errors.New("Error Unmarshalling Response")
	}
	return rm, err
}

// getMessages returns an array of new messages received from the ricochet client
func (r *Ricochet) getMessages() ([]RicochetData, error) {
	buf, err := r.recv()
	if err != nil {
		return nil, errors.New("Failed to retrieve new messages from the client")
	}

	pos := 0
	finished := false
	datas := []RicochetData{}

	for !finished {
		size := int(binary.BigEndian.Uint16(buf[pos+0 : pos+2]))
		channel := int(binary.BigEndian.Uint16(buf[pos+2 : pos+4]))

		if pos+size > len(buf) {
			return datas, errors.New("Partial data packet received")
		}

		data := RicochetData{
			Channel: int32(channel),
			Data:    buf[pos+4 : pos+size],
		}

		datas = append(datas, data)
		pos += size
		if pos >= len(buf) {
			finished = true
		}
	}
	return datas, nil
}

// recv reads data from the client, and returns the raw byte array, else error.
func (r *Ricochet) recv() ([]byte, error) {
	buf := make([]byte, 4096)
	n, err := r.conn.Read(buf)
	if err != nil {
		return nil, err
	}
	ret := make([]byte, n)
	copy(ret[:], buf[:])
	return ret, nil
}

func (r *Ricochet) handleFatal(err error, message string) {
	if err != nil {
		r.logger.Fatal(message)
	}
}
