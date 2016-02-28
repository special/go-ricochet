package goricochet

import (
	"github.com/golang/protobuf/proto"
	"github.com/s-rah/go-ricochet/auth"
	"github.com/s-rah/go-ricochet/chat"
	"github.com/s-rah/go-ricochet/contact"
	"github.com/s-rah/go-ricochet/control"
)

// MessageBuilder allows a client to construct specific data packets for the
// ricochet protocol.
type MessageBuilder struct {
}

// OpenChatChannel contructs a message which will request to open a channel for
// chat on the given channelID.
func (mb *MessageBuilder) OpenChatChannel(channelID int32) ([]byte, error) {
	oc := &Protocol_Data_Control.OpenChannel{
		ChannelIdentifier: proto.Int32(channelID),
		ChannelType:       proto.String("im.ricochet.chat"),
	}
	pc := &Protocol_Data_Control.Packet{
		OpenChannel: oc,
	}
	return proto.Marshal(pc)
}

// OpenContactRequestChannel contructs a message which will reuqest to open a channel for
// a contact request on the given channelID, with the given nick and message.
func (mb *MessageBuilder) OpenContactRequestChannel(channelID int32, nick string, message string) ([]byte, error) {
	// Construct a Contact Request Channel
	oc := &Protocol_Data_Control.OpenChannel{
		ChannelIdentifier: proto.Int32(channelID),
		ChannelType:       proto.String("im.ricochet.contact.request"),
	}

	contactRequest := &Protocol_Data_ContactRequest.ContactRequest{
		Nickname:    proto.String(nick),
		MessageText: proto.String(message),
	}

	err := proto.SetExtension(oc, Protocol_Data_ContactRequest.E_ContactRequest, contactRequest)

	if err != nil {
		return nil, err
	}

	pc := &Protocol_Data_Control.Packet{
		OpenChannel: oc,
	}
	return proto.Marshal(pc)
}

// OpenAuthenticationChannel constructs a message which will reuqest to open a channel for
// authentication on the given channelID, with the given cookie
func (mb *MessageBuilder) OpenAuthenticationChannel(channelID int32, clientCookie [16]byte) ([]byte, error) {
	oc := &Protocol_Data_Control.OpenChannel{
		ChannelIdentifier: proto.Int32(channelID),
		ChannelType:       proto.String("im.ricochet.auth.hidden-service"),
	}
	err := proto.SetExtension(oc, Protocol_Data_AuthHiddenService.E_ClientCookie, clientCookie[:])
	if err != nil {
		return nil, err
	}
	pc := &Protocol_Data_Control.Packet{
		OpenChannel: oc,
	}
	return proto.Marshal(pc)
}

// ChatMessage constructs a chat message with the given content.
func (mb *MessageBuilder) ChatMessage(message string) ([]byte, error) {
	cm := &Protocol_Data_Chat.ChatMessage{
		MessageText: proto.String(message),
	}
	chatPacket := &Protocol_Data_Chat.Packet{
		ChatMessage: cm,
	}
	return proto.Marshal(chatPacket)
}
