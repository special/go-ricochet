package goricochet

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/s-rah/go-ricochet/auth"
	"io/ioutil"
	"log"
	"net"
	"os"
)

// Ricochet is a protocol to conducting anonymous IM.
type Ricochet struct {
	conn   net.Conn
	logger *log.Logger
}

// RicochetData is a structure containing the raw data and the channel it the
// message originated on.
type RicochetData struct {
	Channel int32
	Data    []byte
}

// Init sets up the Ricochet object.
func (r *Ricochet) Init(debugLog bool) {

	if debugLog {
		r.logger = log.New(os.Stdout, "[Ricochet]: ", log.Ltime|log.Lmicroseconds)
	} else {
		r.logger = log.New(ioutil.Discard, "[Ricochet]: ", log.Ltime|log.Lmicroseconds)
	}
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

	return r.negotiateVersion()
}

// Authenticate opens an Authentication Channel and send a client cookie
func (r *Ricochet) Authenticate(channelID int32, clientCookie [16]byte) error {
	messageBuilder := new(MessageBuilder)
	data, err := messageBuilder.OpenAuthenticationChannel(channelID, clientCookie)

	if err != nil {
		return errors.New("Cannot Marshal Open Channel Message")
	}
	r.logger.Printf("Sending Open Channel with Auth Request (channel:%d)", channelID)
	r.sendPacket(data, 0)
	return nil
}

// SendProof sends an authentication proof in response to a challenge.
func (r *Ricochet) SendProof(channelID int32, publickeyBytes []byte, signatureBytes []byte) error {
	// Construct a Proof Message
	proof := &Protocol_Data_AuthHiddenService.Proof{
		PublicKey: publickeyBytes,
		Signature: signatureBytes,
	}

	ahsPacket := &Protocol_Data_AuthHiddenService.Packet{
		Proof:  proof,
		Result: nil,
	}

	data, err := proto.Marshal(ahsPacket)

	if err != nil {
		return err
	}

	r.logger.Printf("Sending Proof Auth Request (channel:%d)", channelID)
	r.sendPacket(data, channelID)
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

// AckOpenChannel acknowledges a previously received open channel message
// Prerequisites:
//              * Must have Previously issued a successful Connect()
func (r *Ricochet) AckOpenChannel(channel int32, result bool) error {
	messageBuilder := new(MessageBuilder)
	data, err := messageBuilder.AckOpenChannel(channel, result)
	if err != nil {
		return errors.New("Failed to serialize open channel ack")
	}
	r.sendPacket(data, 0)
	return nil
}

// AckChatMessage acknowledges a previously received chat message.
// Prerequisites:
//              * Must have Previously issued a successful Connect()
func (r *Ricochet) AckChatMessage(channel int32, messageID int32) error {
	messageBuilder := new(MessageBuilder)
	data, err := messageBuilder.AckChatMessage(messageID)
	if err != nil {
		return errors.New("Failed to serialize chat message ack")
	}
	r.sendPacket(data, channel)
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

// ListenAndWait is intended to be a background thread listening for all messages
// a client will send, automaticall responding to some, and making the others available to
// Listen()
// Prerequisites:
//             * Must have previously issued a successful Connect()
func (r *Ricochet) ListenAndWait(serverHostname string, service RicochetService) error {
	for true {
		packets, err := r.getMessages()
		r.handleFatal(err, "Error attempted to get new messages")

		messageDecoder := new(MessageDecoder)

		for _, packet := range packets {

			if len(packet.Data) == 0 {
				r.logger.Printf("Closing Channel %d", packet.Channel)
				service.OnChannelClose(packet.Channel, serverHostname)
				break
			}

			if packet.Channel == 0 {

				message, err := messageDecoder.DecodeControlMessage(packet.Data)

				if err != nil {
					r.logger.Printf("Failed to decode data packet, discarding")
					break
				}

				if message.Type == "openchannel" && message.Ack == false {
					r.logger.Printf("new open channel request %d %s", message.ChannelID, serverHostname)
					service.OnOpenChannelRequest(message.ChannelID, serverHostname)
				} else if message.Type == "openchannel" && message.Ack == true {
					r.logger.Printf("new open channel request ack %d %s", message.ChannelID, serverHostname)
					service.OnOpenChannelRequestAck(message.ChannelID, serverHostname, message.Accepted)
				} else if message.Type == "openauthchannel" && message.Ack == true {
					r.logger.Printf("new authentication challenge %d %s", message.ChannelID, serverHostname)
					service.OnAuthenticationChallenge(message.ChannelID, serverHostname, message.ServerCookie)
				} else {
					r.logger.Printf("Received Unknown Control Message\n", message)
				}
			} else if packet.Channel == 1 {
				result, _ := messageDecoder.DecodeAuthMessage(packet.Data)
				r.logger.Printf("newreceived auth result %d", packet.Channel)
				service.OnAuthenticationResult(1, serverHostname, result)
			} else {

				// At this point the only other expected type of message is a Chat Message
				messageDecoder := new(MessageDecoder)
				message, err := messageDecoder.DecodeChatMessage(packet.Data)
				if err != nil {
					r.logger.Printf("Failed to decode data packet, discarding on channel %d", packet.Channel)
					break
				}

				if message.Ack == true {
					service.OnChatMessageAck(packet.Channel, serverHostname, message.MessageID)
				} else {
					service.OnChatMessage(packet.Channel, serverHostname, message.MessageID, message.Message)
				}
			}
		}
	}
	return nil
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
	r.logger.Printf("Got %d Packets", len(datas))
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
