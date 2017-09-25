package channels

import (
	"github.com/golang/protobuf/proto"
	"github.com/s-rah/go-ricochet/utils"
	"github.com/s-rah/go-ricochet/wire/chat"
	"github.com/s-rah/go-ricochet/wire/control"
	"testing"
	"time"
)

func TestChatChannelOptions(t *testing.T) {
	chatChannel := new(ChatChannel)

	if chatChannel.Type() != "im.ricochet.chat" {
		t.Errorf("ChatChannel has wrong type %s", chatChannel.Type())
	}

	if chatChannel.OnlyClientCanOpen() {
		t.Errorf("ChatChannel should be able to be opened by everyone")
	}
	if !chatChannel.Singleton() {
		t.Errorf("ChatChannel should be a Singelton")
	}
	if chatChannel.Bidirectional() {
		t.Errorf("ChatChannel should not be bidirectional")
	}
	if chatChannel.RequiresAuthentication() != "im.ricochet.auth.hidden-service" {
		t.Errorf("ChatChannel should require im.ricochet.auth.hidden-service. Instead requires: %s", chatChannel.RequiresAuthentication())
	}
}

func TestChatChannelOpenInbound(t *testing.T) {
	messageBuilder := new(utils.MessageBuilder)
	ocm := messageBuilder.OpenChannel(2, "im.ricochet.chat")

	// We have just constructed this so there is little
	// point in doing error checking here in the test
	res := new(Protocol_Data_Control.Packet)
	proto.Unmarshal(ocm[:], res)
	opm := res.GetOpenChannel()

	chatChannel := new(ChatChannel)
	channel := Channel{ID: 1}
	response, err := chatChannel.OpenInbound(&channel, opm)

	if err == nil {
		res := new(Protocol_Data_Control.Packet)
		proto.Unmarshal(response[:], res)
	} else {
		t.Errorf("Error while parsing chatchannel openinbound output: %v", err)
	}
}

func TestChatChannelOpenOutbound(t *testing.T) {
	chatChannel := new(ChatChannel)
	channel := Channel{ID: 1}
	response, err := chatChannel.OpenOutbound(&channel)
	if err == nil {
		res := new(Protocol_Data_Control.Packet)
		proto.Unmarshal(response[:], res)
		if res.GetOpenChannel() != nil {
			// XXX
		} else {
			t.Errorf("ChatChannel OpenOutbound was not an OpenChannelRequest %v", err)
		}
	} else {
		t.Errorf("Error while parsing openputput output: %v", err)
	}
}

type TestChatChannelHandler struct {
}

func (tcch *TestChatChannelHandler) ChatMessage(messageID uint32, when time.Time, message string) bool {
	return true
}

func (tcch *TestChatChannelHandler) ChatMessageAck(messageID uint32, accepted bool) {

}

func TestChatChannelOperations(t *testing.T) {

	// We test OpenOutboundElsewhere
	chatChannel := new(ChatChannel)
	chatChannel.Handler = new(TestChatChannelHandler)
	channel := Channel{ID: 5}
	channel.SendMessage = func(data []byte) {
		res := new(Protocol_Data_Chat.Packet)
		err := proto.Unmarshal(data, res)
		if res.GetChatMessage() != nil {
			if err == nil {
				if res.GetChatMessage().GetMessageId() != 0 {
					t.Log("Got Message ID:", res.GetChatMessage().GetMessageId())
					return
				}
				t.Errorf("message id was 0 should be random")
				return
			}
			t.Errorf("error sending chat message: %v", err)
		}
	}
	chatChannel.OpenOutbound(&channel)

	messageBuilder := new(utils.MessageBuilder)
	ack := messageBuilder.AckOpenChannel(5)
	// We have just constructed this so there is little
	// point in doing error checking here in the test
	res := new(Protocol_Data_Control.Packet)
	proto.Unmarshal(ack[:], res)
	cr := res.GetChannelResult()

	chatChannel.OpenOutboundResult(nil, cr)
	if channel.Pending {
		t.Errorf("After Successful Result ChatChannel Is Still Pending")
	}

	chat := messageBuilder.ChatMessage("message text", 0, 0)
	chatChannel.Packet(chat)

	chatChannel.SendMessage("hello")

}
