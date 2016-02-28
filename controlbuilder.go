package goricochet

import (
	"github.com/s-rah/go-ricochet/control"
	"github.com/s-rah/go-ricochet/auth"
	"github.com/golang/protobuf/proto"
)

type ControlBuilder struct {

}

func (cb *ControlBuilder) OpenChatChannel(channelId int32) ([]byte,error) {
	oc := &Protocol_Data_Control.OpenChannel{
		ChannelIdentifier: proto.Int32(channelId),
		ChannelType:       proto.String("im.ricochet.chat"),
	}
	pc := &Protocol_Data_Control.Packet{
		OpenChannel: oc,
	}
	return proto.Marshal(pc)
}

func (cb* ControlBuilder) OpenAuthenticationChannel(channelId int32, clientCookie [16]byte) ([]byte,error) {
	oc := &Protocol_Data_Control.OpenChannel{
		ChannelIdentifier: proto.Int32(channelId),
		ChannelType:       proto.String("im.ricochet.auth.hidden-service" ),
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
