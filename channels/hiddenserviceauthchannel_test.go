package channels

import (
	"bytes"
	"crypto/rsa"
	"github.com/golang/protobuf/proto"
	"github.com/s-rah/go-ricochet/utils"
	"github.com/s-rah/go-ricochet/wire/control"
	"testing"
)

func TestGenChallenge(t *testing.T) {
	authHandler := new(HiddenServiceAuthChannel)
	authHandler.AddClientCookie([]byte("abcdefghijklmnop"))
	authHandler.AddServerCookie([]byte("qrstuvwxyz012345"))
	challenge := authHandler.GenChallenge("test.onion", "notareal.onion")
	expectedChallenge := []byte{0xf5, 0xdb, 0xfd, 0xf0, 0x3d, 0x94, 0x14, 0xf1, 0x4b, 0x37, 0x93, 0xe2, 0xa5, 0x11, 0x4a, 0x98, 0x31, 0x90, 0xea, 0xb8, 0x95, 0x7a, 0x2e, 0xaa, 0xd0, 0xd2, 0x0c, 0x74, 0x95, 0xba, 0xab, 0x73}
	t.Log(challenge, expectedChallenge)
	if bytes.Compare(challenge[:], expectedChallenge[:]) != 0 {
		t.Errorf("HiddenServiceAuthChannel Challenge Is Invalid, Got %x, Expected %x", challenge, expectedChallenge)
	}
}

func TestGenClientCookie(t *testing.T) {
	authHandler := new(HiddenServiceAuthChannel)
	clientCookie := authHandler.GenClientCookie()
	if clientCookie != authHandler.clientCookie {
		t.Errorf("HiddenServiceAuthChannel Client Cookies are Different %x %x", clientCookie, authHandler.clientCookie)
	}
}

func TestGenServerCookie(t *testing.T) {
	authHandler := new(HiddenServiceAuthChannel)
	serverCookie := authHandler.GenServerCookie()
	if serverCookie != authHandler.serverCookie {
		t.Errorf("HiddenServiceAuthChannel Server Cookies are Different %x %x", serverCookie, authHandler.serverCookie)
	}
}

func TestHiddenServiceAuthChannelOptions(t *testing.T) {
	hiddenServiceAuthChannel := new(HiddenServiceAuthChannel)

	if hiddenServiceAuthChannel.Type() != "im.ricochet.auth.hidden-service" {
		t.Errorf("AuthHiddenService has wrong type %s", hiddenServiceAuthChannel.Type())
	}

	if !hiddenServiceAuthChannel.OnlyClientCanOpen() {
		t.Errorf("AuthHiddenService Should be Client Open Only")
	}
	if !hiddenServiceAuthChannel.Singleton() {
		t.Errorf("AuthHiddenService Should be a Singelton")
	}
	if hiddenServiceAuthChannel.Bidirectional() {
		t.Errorf("AuthHiddenService Should not be bidirectional")
	}
	if hiddenServiceAuthChannel.RequiresAuthentication() != "none" {
		t.Errorf("AuthHiddenService should require no authorization. Instead requires: %s", hiddenServiceAuthChannel.RequiresAuthentication())
	}
}

func GetOpenAuthenticationChannelMessage() *Protocol_Data_Control.OpenChannel {
	// Construct the Open Authentication Channel Message
	messageBuilder := new(utils.MessageBuilder)
	ocm := messageBuilder.OpenAuthenticationChannel(1, [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})

	// We have just constructed this so there is little
	// point in doing error checking here in the test
	res := new(Protocol_Data_Control.Packet)
	proto.Unmarshal(ocm[:], res)
	return res.GetOpenChannel()
}

func TestAuthenticationOpenInbound(t *testing.T) {
	privateKey, _ := utils.LoadPrivateKeyFromFile("../testing/private_key")
	opm := GetOpenAuthenticationChannelMessage()
	authHandler := new(HiddenServiceAuthChannel)
	authHandler.PrivateKey = privateKey
	channel := Channel{ID: 1}
	response, err := authHandler.OpenInbound(&channel, opm)

	if err == nil {
		res := new(Protocol_Data_Control.Packet)
		proto.Unmarshal(response[:], res)

		if res.GetChannelResult() == nil || !res.GetChannelResult().GetOpened() {
			t.Errorf("Response not a Open Channel Result %v", res)
		}
	} else {
		t.Errorf("HiddenServiceAuthChannel OpenOutbound Failed: %v", err)
	}
}

func TestAuthenticationOpenOutbound(t *testing.T) {
	privateKey, _ := utils.LoadPrivateKeyFromFile("../testing/private_key")
	authHandler := new(HiddenServiceAuthChannel)
	authHandler.PrivateKey = privateKey
	channel := Channel{ID: 1}
	response, err := authHandler.OpenOutbound(&channel)

	if err == nil {
		res := new(Protocol_Data_Control.Packet)
		proto.Unmarshal(response[:], res)

		if res.GetOpenChannel() == nil {
			t.Errorf("Open Channel Packet not included %v", res)
		}
	} else {
		t.Errorf("HiddenServiceAuthChannel OpenInbound Failed: %v", err)
	}

}

func TestAuthenticationOpenOutboundResult(t *testing.T) {

	privateKey, _ := utils.LoadPrivateKeyFromFile("../testing/private_key")

	authHandlerA := new(HiddenServiceAuthChannel)
	authHandlerB := new(HiddenServiceAuthChannel)

	authHandlerA.ServerHostname = "kwke2hntvyfqm7dr"
	authHandlerA.PrivateKey = privateKey
	authHandlerA.ClientAuthResult = func(accepted, known bool) {}
	channelA := Channel{ID: 1, Direction: Outbound}
	channelA.SendMessage = func(message []byte) {
		authHandlerB.Packet(message)
	}
	channelA.DelegateAuthorization = func() {}
	channelA.CloseChannel = func() {}
	response, _ := authHandlerA.OpenOutbound(&channelA)
	res := new(Protocol_Data_Control.Packet)
	proto.Unmarshal(response[:], res)

	authHandlerB.ServerHostname = "kwke2hntvyfqm7dr"
	authHandlerB.PrivateKey = privateKey
	authHandlerB.ServerAuthValid = func(hostname string, publicKey rsa.PublicKey) (allowed, known bool) { return true, true }
	authHandlerB.ServerAuthInvalid = func(err error) { t.Error("server received invalid auth") }
	channelB := Channel{ID: 1, Direction: Inbound}
	channelB.SendMessage = func(message []byte) {
		authHandlerA.Packet(message)
	}
	channelB.DelegateAuthorization = func() {}
	channelB.CloseChannel = func() {}
	response, _ = authHandlerB.OpenInbound(&channelB, res.GetOpenChannel())
	res = new(Protocol_Data_Control.Packet)
	proto.Unmarshal(response[:], res)

	authHandlerA.OpenOutboundResult(nil, res.GetChannelResult())

}
