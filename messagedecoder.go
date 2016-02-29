package goricochet

import (
	"errors"
	"github.com/golang/protobuf/proto"
	"github.com/s-rah/go-ricochet/auth"
	"github.com/s-rah/go-ricochet/chat"
	"github.com/s-rah/go-ricochet/control"
)

type MessageDecoder struct {
}

// Conceptual Chat Message - we construct this to avoid polluting the
// the main ricochet code with protobuf cruft - and enable us to minimise the
// code that may break in the future.
type RicochetChatMessage struct {
	Ack       bool
	MessageID int32
	Message   string
	Accepted  bool
}

// Conceptual Control Message - we construct this to avoid polluting the
// the main ricochet code with protobuf cruft - and enable us to minimise the
// code that may break in the future.
type RicochetControlMessage struct {
	Ack          bool
	Type         string
	ChannelID    int32
	Accepted     bool
	ClientCookie [16]byte
	ServerCookie [16]byte
}

// DecodeAuthMessage
func (md *MessageDecoder) DecodeAuthMessage(data []byte) (bool, error) {
	res := new(Protocol_Data_AuthHiddenService.Packet)
	err := proto.Unmarshal(data[:], res)
	if err != nil {
		return false, errors.New("error unmarshalling control message type")
	}
	return res.GetResult().GetAccepted(), nil
}

// DecodeControlMessage
func (md *MessageDecoder) DecodeControlMessage(data []byte) (*RicochetControlMessage, error) {
	res := new(Protocol_Data_Control.Packet)
	err := proto.Unmarshal(data[:], res)

	if err != nil {
		return nil, errors.New("error unmarshalling control message type")
	}

	if res.GetOpenChannel() != nil {
		ricochetControlMessage := new(RicochetControlMessage)
		ricochetControlMessage.Ack = false

		if res.GetOpenChannel().GetChannelType() == "im.ricochet.auth.hidden-service" {
			ricochetControlMessage.Type = "openauthchannel"
		}

		ricochetControlMessage.Type = "openchannel"
		ricochetControlMessage.ChannelID = int32(res.GetOpenChannel().GetChannelIdentifier())
		return ricochetControlMessage, nil
	} else if res.GetChannelResult() != nil {
		ricochetControlMessage := new(RicochetControlMessage)
		ricochetControlMessage.Ack = true
		ricochetControlMessage.ChannelID = int32(res.GetOpenChannel().GetChannelIdentifier())

		serverCookie, err := proto.GetExtension(res.GetChannelResult(), Protocol_Data_AuthHiddenService.E_ServerCookie)

		if err == nil {
			ricochetControlMessage.Type = "openauthchannel"
			copy(ricochetControlMessage.ServerCookie[:], serverCookie.([]byte))
		} else {
			ricochetControlMessage.Type = "openchannel"

		}

		return ricochetControlMessage, nil
	}
	return nil, errors.New("unknown control message type")
}

// DecodeChatMessage takes a byte representing a data packet and returns a
// constructed RicochetControlMessage
func (md *MessageDecoder) DecodeChatMessage(data []byte) (*RicochetChatMessage, error) {
	res := new(Protocol_Data_Chat.Packet)
	err := proto.Unmarshal(data[:], res)

	if err != nil {
		return nil, err
	}

	if res.GetChatMessage() != nil {
		ricochetChatMessage := new(RicochetChatMessage)
		ricochetChatMessage.Ack = false
		ricochetChatMessage.MessageID = int32(res.GetChatMessage().GetMessageId())
		ricochetChatMessage.Message = res.GetChatMessage().GetMessageText()
		return ricochetChatMessage, nil
	} else if res.GetChatAcknowledge != nil {
		ricochetChatMessage := new(RicochetChatMessage)
		ricochetChatMessage.Ack = true
		ricochetChatMessage.MessageID = int32(res.GetChatAcknowledge().GetMessageId())
		ricochetChatMessage.Accepted = res.GetChatAcknowledge().GetAccepted()
		return ricochetChatMessage, nil
	}
	return nil, errors.New("chat message type not supported")
}
