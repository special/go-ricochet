package goricochet

import (
	"crypto"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/s-rah/go-ricochet/auth"
	"github.com/s-rah/go-ricochet/chat"
	"github.com/s-rah/go-ricochet/contact"
	"github.com/s-rah/go-ricochet/control"
	"h12.me/socks"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
)

// Ricochet is a protocol to conducting anonymous IM.
type Ricochet struct {
	conn       net.Conn
	privateKey *pem.Block
	logger     *log.Logger
}

// Init sets up the Ricochet object. It takes in a filename of a hidden service
// private_key file so it can successfully authenticate itself with other
// clients.
func (r *Ricochet) Init(filename string) {
	r.logger = log.New(os.Stdout, "[Ricochet]: ", log.Ltime|log.Lmicroseconds)
	pemData, err := ioutil.ReadFile(filename)

	if err != nil {
		r.logger.Print("Error Reading Private Key: ", err)
	}

	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		r.logger.Print("No valid PEM data found")
	}

	r.privateKey = block
}

func (r *Ricochet) send(data []byte) {
	fmt.Fprintf(r.conn, "%s", data)
}

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

func (r *Ricochet) decodePacket(response []byte) *Protocol_Data_Control.Packet {
	// TODO: Check Length and Channel are Sane
	if len(response) < 4 {
		r.logger.Fatal("Response is too short ", response)
		return nil
	}
	res := new(Protocol_Data_Control.Packet)
	err := proto.Unmarshal(response[4:], res)

	if err != nil {
		r.logger.Fatal("Error Unmarshalling Response", err)
		panic(err)
	}

	return res
}

func (r *Ricochet) decodeResult(response []byte) *Protocol_Data_AuthHiddenService.Packet {
	// TODO: Check Length and Channel are Sane
	if len(response) < 4 {
		r.logger.Fatal("Response is too short ", response)
		return nil
	}
	length := response[1]

	r.logger.Print(response)
	res := new(Protocol_Data_AuthHiddenService.Packet)
	err := proto.Unmarshal(response[4:length], res)

	if err != nil {
		r.logger.Fatal("Error Unmarshalling Response: ", err)
		panic(err)
	}

	return res
}

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
	err := proto.SetExtension(oc, Protocol_Data_AuthHiddenService.E_ClientCookie, []byte("0000000000000000"))
	pc := &Protocol_Data_Control.Packet{
		OpenChannel: oc,
	}
	data, err := proto.Marshal(pc)

	if err != nil {
		r.logger.Fatal("Cannot Marshal Open Channel Message: ", err)
		panic("Cannot Marshal Open Channel Message")
	}

	openChannel := r.constructProtocol(data, 0)
	r.logger.Print("Opening Channel: ", pc)
	r.send(openChannel)

	response, _ := r.recv()
	openChannelResponse := r.decodePacket(response)
	r.logger.Print("Received Response: ", openChannelResponse)
	channelResult := openChannelResponse.GetChannelResult()

	if channelResult.GetOpened() == true {
		r.logger.Print("Channel Opened Successfully: ", channelResult.GetChannelIdentifier())
	}

	sCookie, _ := proto.GetExtension(channelResult, Protocol_Data_AuthHiddenService.E_ServerCookie)
	serverCookie, _ := sCookie.([]byte)

	r.logger.Print("Starting Authentication with Server Cookie: ", serverCookie)

	key := make([]byte, 32)
	copy(key[0:16], []byte("0000000000000000"))
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

	response, _ = r.recv()
	resultResponse := r.decodeResult(response)
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
	response, _ := r.recv()
	openChannelResponse := r.decodePacket(response)
	r.logger.Print("Received Response: ", openChannelResponse)

	channelResult := openChannelResponse.GetChannelResult()

	if channelResult.GetOpened() == true {
		r.logger.Print("Channel Opened Successfully: ", channelResult.GetChannelIdentifier())
		return nil
	}
	return errors.New("failed to open channel")
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
		panic("Cannot Marshal Open Channel Message")
	}

	openChannel := r.constructProtocol(data, 0)
	r.logger.Print("Opening Channel: ", pc)
	r.send(openChannel)
	response, _ := r.recv()
	openChannelResponse := r.decodePacket(response)
	r.logger.Print("Received Response: ", openChannelResponse)
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
		panic("Failed Version Negotiating")
	}

	if res[0] != 1 {
		r.logger.Fatal("Failed Version Negotiating - Invalid Version ", res)
		panic("Failed Version Negotiating")
	}

	r.logger.Print("Successfully Negotiated Version ", res[0])
}
