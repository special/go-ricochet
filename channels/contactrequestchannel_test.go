package channels

import (
	"github.com/golang/protobuf/proto"
	"github.com/s-rah/go-ricochet/utils"
	"github.com/s-rah/go-ricochet/wire/contact"
	"github.com/s-rah/go-ricochet/wire/control"
	"testing"
)

type TestContactRequestHandler struct {
	Received bool
}

func (tcrh *TestContactRequestHandler) ContactRequest(name string, message string) string {
	if name == "test_nickname" && message == "test_message" {
		tcrh.Received = true
	}
	return "Pending"
}

func (tcrh *TestContactRequestHandler) ContactRequestRejected() {
}
func (tcrh *TestContactRequestHandler) ContactRequestAccepted() {
}
func (tcrh *TestContactRequestHandler) ContactRequestError() {
}

func TestContactRequestOptions(t *testing.T) {
	contactRequestChannel := new(ContactRequestChannel)

	if contactRequestChannel.Type() != "im.ricochet.contact.request" {
		t.Errorf("ContactRequestChannel has wrong type %s", contactRequestChannel.Type())
	}

	if !contactRequestChannel.OnlyClientCanOpen() {
		t.Errorf("ContactRequestChannel Should be Client Open Only")
	}
	if !contactRequestChannel.Singleton() {
		t.Errorf("ContactRequestChannel Should be a Singelton")
	}
	if contactRequestChannel.Bidirectional() {
		t.Errorf("ContactRequestChannel Should not be bidirectional")
	}
	if contactRequestChannel.RequiresAuthentication() != "im.ricochet.auth.hidden-service" {
		t.Errorf("ContactRequestChannel should requires im.ricochet.auth.hidden-service Authentication. Instead defines: %s", contactRequestChannel.RequiresAuthentication())
	}
}

func TestContactRequestOpenOutbound(t *testing.T) {
	contactRequestChannel := new(ContactRequestChannel)
	handler := new(TestContactRequestHandler)
	contactRequestChannel.Handler = handler
	channel := Channel{ID: 1}
	response, err := contactRequestChannel.OpenOutbound(&channel)
	if err == nil {
		res := new(Protocol_Data_Control.Packet)
		proto.Unmarshal(response[:], res)
		if res.GetOpenChannel() != nil {
			// XXX
		} else {
			t.Errorf("ContactReuqest OpenOutbound was not an OpenChannelRequest %v", err)
		}
	} else {
		t.Errorf("Error while parsing openputput output: %v", err)
	}
}

func TestContactRequestOpenOutboundResult(t *testing.T) {
	contactRequestChannel := &ContactRequestChannel{
		Name:    "test_nickname",
		Message: "test_message",
		Handler: &TestContactRequestHandler{},
	}
	channel := Channel{ID: 1}
	contactRequestChannel.OpenOutbound(&channel)

	messageBuilder := new(utils.MessageBuilder)
	ack := messageBuilder.ReplyToContactRequestOnResponse(1, "Accepted")
	// We have just constructed this so there is little
	// point in doing error checking here in the test
	res := new(Protocol_Data_Control.Packet)
	proto.Unmarshal(ack[:], res)
	cr := res.GetChannelResult()

	contactRequestChannel.OpenOutboundResult(nil, cr)

}

func TestContactRequestOpenInbound(t *testing.T) {
	opm := BuildOpenChannel("test_nickname", "test_message")
	contactRequestChannel := new(ContactRequestChannel)
	handler := new(TestContactRequestHandler)
	contactRequestChannel.Handler = handler
	channel := Channel{ID: 1}
	response, err := contactRequestChannel.OpenInbound(&channel, opm)

	if err == nil {
		res := new(Protocol_Data_Control.Packet)
		proto.Unmarshal(response[:], res)

		responseI, err := proto.GetExtension(res.GetChannelResult(), Protocol_Data_ContactRequest.E_Response)
		if err == nil {
			response, check := responseI.(*Protocol_Data_ContactRequest.Response)
			if check {
				if response.GetStatus().String() != "Pending" {
					t.Errorf("Contact Request Response should have been Pending, but instead was: %v", response.GetStatus().String())
				}
			} else {
				t.Errorf("Error while parsing openinbound output: %v", err)
			}
		} else {
			t.Errorf("Error while parsing openinbound output: %v", err)
		}
	} else {
		t.Errorf("Error while parsing openinbound output: %v", err)
	}

	if !handler.Received {
		t.Errorf("Contact Request was not received by Handler")
	}
}

func TestContactRequestPacket(t *testing.T) {
	contactRequestChannel := new(ContactRequestChannel)
	handler := new(TestContactRequestHandler)
	contactRequestChannel.Handler = handler
	channel := Channel{ID: 1}
	contactRequestChannel.OpenOutbound(&channel)

	messageBuilder := new(utils.MessageBuilder)
	ack := messageBuilder.ReplyToContactRequestOnResponse(1, "Pending")
	// We have just constructed this so there is little
	// point in doing error checking here in the test
	res := new(Protocol_Data_Control.Packet)
	proto.Unmarshal(ack[:], res)
	cr := res.GetChannelResult()

	contactRequestChannel.OpenOutboundResult(nil, cr)

	ackp := messageBuilder.ReplyToContactRequest(1, "Accepted")
	contactRequestChannel.Packet(ackp)
}

func TestContactRequestRejected(t *testing.T) {
	contactRequestChannel := new(ContactRequestChannel)
	handler := new(TestContactRequestHandler)
	contactRequestChannel.Handler = handler
	channel := Channel{ID: 1}
	contactRequestChannel.OpenOutbound(&channel)

	messageBuilder := new(utils.MessageBuilder)
	ack := messageBuilder.ReplyToContactRequestOnResponse(1, "Pending")
	// We have just constructed this so there is little
	// point in doing error checking here in the test
	res := new(Protocol_Data_Control.Packet)
	proto.Unmarshal(ack[:], res)
	cr := res.GetChannelResult()

	contactRequestChannel.OpenOutboundResult(nil, cr)

	ackp := messageBuilder.ReplyToContactRequest(1, "Rejected")
	contactRequestChannel.Packet(ackp)
}

func TestContactRequestError(t *testing.T) {
	contactRequestChannel := new(ContactRequestChannel)
	handler := new(TestContactRequestHandler)
	contactRequestChannel.Handler = handler
	channel := Channel{ID: 1}
	contactRequestChannel.OpenOutbound(&channel)

	messageBuilder := new(utils.MessageBuilder)
	ack := messageBuilder.ReplyToContactRequestOnResponse(1, "Pending")
	// We have just constructed this so there is little
	// point in doing error checking here in the test
	res := new(Protocol_Data_Control.Packet)
	proto.Unmarshal(ack[:], res)
	cr := res.GetChannelResult()

	contactRequestChannel.OpenOutboundResult(nil, cr)

	ackp := messageBuilder.ReplyToContactRequest(1, "Error")
	contactRequestChannel.Packet(ackp)
}

func BuildOpenChannel(nickname string, message string) *Protocol_Data_Control.OpenChannel {
	// Construct the Open Authentication Channel Message
	messageBuilder := new(utils.MessageBuilder)
	ocm := messageBuilder.OpenContactRequestChannel(1, nickname, message)
	// We have just constructed this so there is little
	// point in doing error checking here in the test
	res := new(Protocol_Data_Control.Packet)
	proto.Unmarshal(ocm[:], res)
	return res.GetOpenChannel()
}

func TestInvalidNickname(t *testing.T) {
	opm := BuildOpenChannel("this nickname is far too long at well over the limit of 30 characters", "test_message")
	contactRequestChannel := new(ContactRequestChannel)
	handler := new(TestContactRequestHandler)
	contactRequestChannel.Handler = handler
	channel := Channel{ID: 1}
	_, err := contactRequestChannel.OpenInbound(&channel, opm)
	if err == nil {
		t.Errorf("Open Inbound should have failed because of invalid nickname")
	}
}

func TestInvalidMessage(t *testing.T) {
	var message string
	for i := 0; i < 2001; i++ {
		message += "a"
	}
	opm := BuildOpenChannel("test_nickname", message)
	contactRequestChannel := new(ContactRequestChannel)
	handler := new(TestContactRequestHandler)
	contactRequestChannel.Handler = handler
	channel := Channel{ID: 1}
	_, err := contactRequestChannel.OpenInbound(&channel, opm)
	if err == nil {
		t.Errorf("Open Inbound should have failed because of invalid message")
	}
}
