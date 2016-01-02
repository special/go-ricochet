package goricochet

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/s-rah/go-ricochet/auth"
	"github.com/s-rah/go-ricochet/chat"
	"github.com/s-rah/go-ricochet/contact"
	"github.com/s-rah/go-ricochet/control"
	"h12.me/socks"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
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
	privateKey   *pem.Block
	logger       *log.Logger
	channelState map[int]int
	channel      chan RicochetMessage
}

// RicochetData is a structure containing the raw data and the channel it the
// message originated on.
type RicochetData struct {
	Channel int
	Data    []byte
}

// RicochetMessage is a Wrapper Around Common Ricochet Protocol Strucutres
type RicochetMessage struct {
	Channel       int
	ControlPacket *Protocol_Data_Control.Packet
	DataPacket    *Protocol_Data_Chat.Packet
	AuthPacket    *Protocol_Data_AuthHiddenService.Packet
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

	r.privateKey = block
	r.channelState = make(map[int]int)
	r.channel = make(chan RicochetMessage)
}

// Connect sets up a ricochet connection between from and to which are
// both ricochet formated hostnames e.g. qn6uo4cmsrfv4kzq.onion. If this
// function finished successfully then the connection can be assumed to
// be open and authenticated.
// To specify a local port using the format "127.0.0.1:[port]|ricochet-id" for
// to
func (r *Ricochet) Connect(from string, to string) error {

	if strings.HasPrefix(to, "127.0.0.1") {
		toAddr := strings.Split(to, "|")
		tcpAddr, err := net.ResolveTCPAddr("tcp", toAddr[0])
		if err != nil {
			r.logger.Fatal("Cannot Resolve TCP Address ", err)
			return errors.New("Cannot Resolve Local TCP Address")
		}
		r.conn, err = net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			r.logger.Fatal("Cannot Dial TCP Address ", err)
			return errors.New("Cannot Dial Local TCP Address")
		}
		r.logger.Print("Connected to " + to + " as " + toAddr[1])
		to = toAddr[1]
	} else {
		dialSocksProxy := socks.DialSocksProxy(socks.SOCKS5, "127.0.0.1:9050")
		r.logger.Print("Connecting to ", to+".onion:9878")
		conn, err := dialSocksProxy("", to+".onion:9878")
		if err != nil {
			r.logger.Fatal("Cannot Dial Remove Address ", err)
			return errors.New("Cannot Dial Remote Ricochet Address")
		}
		r.conn = conn
		r.logger.Print("Connected to ", to+".onion:9878")
	}

	r.negotiateVersion()

	// Construct an Open Channel Message
	oc := &Protocol_Data_Control.OpenChannel{
		ChannelIdentifier: proto.Int32(1),
		ChannelType:       proto.String("im.ricochet.auth.hidden-service"),
	}

	var cookie [16]byte
	io.ReadFull(rand.Reader, cookie[:])

	err := proto.SetExtension(oc, Protocol_Data_AuthHiddenService.E_ClientCookie, cookie[:])
	pc := &Protocol_Data_Control.Packet{
		OpenChannel: oc,
	}
	data, err := proto.Marshal(pc)

	if err != nil {
		r.logger.Fatal("Cannot Marshal Open Channel Message: ", err)
	}

	openChannel := r.constructProtocol(data, 0)
	r.logger.Print("Opening Channel: ", pc)
	r.send(openChannel)

	response, _ := r.getMessages()
	openChannelResponse, _ := r.decodePacket(response[0], CONTROL)
	r.logger.Print("Received Response: ", openChannelResponse)
	channelResult := openChannelResponse.ControlPacket.GetChannelResult()

	if channelResult.GetOpened() == true {
		r.logger.Print("Channel Opened Successfully: ", channelResult.GetChannelIdentifier())
	}

	sCookie, _ := proto.GetExtension(channelResult, Protocol_Data_AuthHiddenService.E_ServerCookie)
	serverCookie, _ := sCookie.([]byte)

	r.logger.Print("Starting Authentication with Server Cookie: ", serverCookie)

	key := make([]byte, 32)
	copy(key[0:16], cookie[:])
	copy(key[16:], serverCookie)
	value := []byte(from + to)
	r.logger.Print("Got Hmac Key: ", key)
	r.logger.Print("Got Proof Value: ", string(value))
	mac := hmac.New(sha256.New, key)
	mac.Write(value)
	hmac := mac.Sum(nil)
	r.logger.Print("Got HMAC: ", hmac)

	privateKey, err := x509.ParsePKCS1PrivateKey(r.privateKey.Bytes)
	if err != nil {
		r.logger.Fatalf("Private key can't be decoded: %s", err)
	}

	// DER Encode the Public Key
	publickeybytes, err := asn1.Marshal(rsa.PublicKey{
		N: privateKey.PublicKey.N,
		E: privateKey.PublicKey.E,
	})

	signature, _ := rsa.SignPKCS1v15(nil, privateKey, crypto.SHA256, hmac)
	signatureBytes := make([]byte, 128)
	copy(signatureBytes[:], signature[:])

	r.logger.Print("Signature Length: ", len(signatureBytes))
	r.logger.Print("Public Key Length: ", len(publickeybytes), ", Bit Size: ", privateKey.PublicKey.N.BitLen())

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

	sendProof := r.constructProtocol(data, 1)
	r.logger.Print("Constructed Proof: ", ahsPacket)
	r.send(sendProof)

	response, _ = r.getMessages()
	resultResponse, _ := r.decodePacket(response[0], AUTH)
	r.logger.Print("Received Result: ", resultResponse)
	return nil
}

// OpenChannel opens a new channel with the given type and id
// Prerequisites:
//              * Must have Previously issued a successful Connect()
//              * If acting as the client, id must be odd (currently this is the
//                only supported option.
func (r *Ricochet) OpenChannel(channelType string, id int) error {
	oc := &Protocol_Data_Control.OpenChannel{
		ChannelIdentifier: proto.Int32(int32(id)),
		ChannelType:       proto.String(channelType),
	}

	pc := &Protocol_Data_Control.Packet{
		OpenChannel: oc,
	}

	data, _ := proto.Marshal(pc)
	openChannel := r.constructProtocol(data, 0)
	r.logger.Print("Opening Channel: ", pc)
	r.send(openChannel)
	return nil
}

// SendContactRequest initiates a contact request to the server.
// Prerequisites:
//              * Must have Previously issued a successful Connect()
func (r *Ricochet) SendContactRequest(nick string, message string) {
	// Construct a Contact Request Channel
	oc := &Protocol_Data_Control.OpenChannel{
		ChannelIdentifier: proto.Int32(3),
		ChannelType:       proto.String("im.ricochet.contact.request"),
	}

	contactRequest := &Protocol_Data_ContactRequest.ContactRequest{
		Nickname:    proto.String(nick),
		MessageText: proto.String(message),
	}

	err := proto.SetExtension(oc, Protocol_Data_ContactRequest.E_ContactRequest, contactRequest)
	pc := &Protocol_Data_Control.Packet{
		OpenChannel: oc,
	}
	data, err := proto.Marshal(pc)

	if err != nil {
		r.logger.Fatal("Cannot Marshal Open Channel Message: ", err)
	}

	openChannel := r.constructProtocol(data, 0)
	r.logger.Print("Opening Channel: ", pc)
	r.send(openChannel)
}

// SendMessage sends a Chat Message (message) to a give Channel (channel).
// Prerequisites:
//             * Must have previously issued a successful Connect()
//             * Must have previously opened channel with OpenChanel
func (r *Ricochet) SendMessage(message string, channel int) {
	// Construct a Contact Request Channel
	cm := &Protocol_Data_Chat.ChatMessage{
		MessageText: proto.String(message),
	}
	chatPacket := &Protocol_Data_Chat.Packet{
		ChatMessage: cm,
	}

	data, _ := proto.Marshal(chatPacket)
	chatMessageBytes := r.constructProtocol(data, channel)
	r.logger.Print("Sending Message: ", chatPacket)
	r.send(chatMessageBytes)
}

// negotiateVersion Perform version negotiation with the connected host.
func (r *Ricochet) negotiateVersion() {
	version := make([]byte, 4)
	version[0] = 0x49
	version[1] = 0x4D
	version[2] = 0x01
	version[3] = 0x01
	r.send(version)
	r.logger.Print("Negotiating Version ", version)
	res, err := r.recv()

	if len(res) != 1 || err != nil {
		r.logger.Fatal("Failed Version Negotiating: ", res, err)
	}

	if res[0] != 1 {
		r.logger.Fatal("Failed Version Negotiating - Invalid Version ", res)
	}

	r.logger.Print("Successfully Negotiated Version ", res[0])
}

// constructProtocol places the data into a structure needed for the client to
// decode the packet.
func (r *Ricochet) constructProtocol(data []byte, channel int) []byte {
	header := make([]byte, 4+len(data))
	r.logger.Print("Wrting Packet of Size: ", len(header))
	header[0] = byte(len(header) >> 8)
	header[1] = byte(len(header) & 0x00FF)
	header[2] = 0x00
	header[3] = byte(channel)
	copy(header[4:], data[:])
	return header
}

// send is a utility funtion to send data to the connected client.
func (r *Ricochet) send(data []byte) {
	fmt.Fprintf(r.conn, "%s", data)
}

// Listen blocks and waits for a new message to arrive from the connected user
// once a message has arrived, it returns the message and the channel it occured
// on, else it returns an error.
// Prerequisites:
//             * Must have previously issued a successful Connect()
//             * Must have previously ran "go ricochet.ListenAndWait()"
func (r *Ricochet) Listen() (string, int, error) {
	var message RicochetMessage
	message = <-r.channel
	r.logger.Print("Received Result: ", message)
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

	data, _ := proto.Marshal(pc)
	ack := r.constructProtocol(data, message.Channel)
	r.send(ack)
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
				message, _ := r.decodePacket(packet, CONTROL)

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

					data, _ := proto.Marshal(pc)
					openChannel := r.constructProtocol(data, 0)
					r.logger.Print("Opening Channel: ", pc)
					r.send(openChannel)
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

			} else if packet.Channel == 3 {
				// Contact Request
			} else {
				// At this point the only other expected type of message
				// is a Chat Message
				message, _ := r.decodePacket(packet, DATA)
				r.logger.Print("Receieved Data Packet: ", message)
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
		r.logger.Fatal("Error Unmarshalling Response", err)
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
		r.logger.Println(buf[pos+2 : pos+4])

		if pos+size > len(buf) {
			return datas, errors.New("Partial data packet received")
		}

		data := RicochetData{
			Channel: int(channel),
			Data:    buf[pos+4 : pos+size],
		}
		r.logger.Println("Got new Data:", data)
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
	r.logger.Print("Received Response From Service: ", n, err)
	if err != nil {
		return nil, err
	}
	ret := make([]byte, n)
	copy(ret[:], buf[:])
	return ret, nil
}
