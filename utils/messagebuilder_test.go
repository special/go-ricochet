package utils

import (
	"github.com/golang/protobuf/proto"
	"github.com/s-rah/go-ricochet/wire/control"
	"testing"
)

func TestOpenChatChannel(t *testing.T) {
	messageBuilder := new(MessageBuilder)
	messageBuilder.OpenChannel(1, "im.ricochet.chat")
	// TODO: More Indepth Test Of Output
}

func TestOpenContactRequestChannel(t *testing.T) {
	messageBuilder := new(MessageBuilder)
	messageBuilder.OpenContactRequestChannel(3, "Nickname", "Message")
	// TODO: More Indepth Test Of Output
}

func TestOpenAuthenticationChannel(t *testing.T) {
	messageBuilder := new(MessageBuilder)
	messageBuilder.OpenAuthenticationChannel(1, [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	// TODO: More Indepth Test Of Output
}

func TestChatMessage(t *testing.T) {
	messageBuilder := new(MessageBuilder)
	messageBuilder.ChatMessage("Hello World", 0, 0)
	// TODO: More Indepth Test Of Output
}

func TestKeepAlive(t *testing.T) {
	messageBuilder := new(MessageBuilder)
	raw := messageBuilder.KeepAlive(true)
	res := new(Protocol_Data_Control.Packet)
	err := proto.Unmarshal(raw, res)
	if err != nil || res.GetKeepAlive() == nil || !res.GetKeepAlive().GetResponseRequested() {
		t.Errorf("Decoding Keep Alive Packet failed or no response requested: %v %v", err, res)
	}
}

func TestFeaturesEnabled(t *testing.T) {
	messageBuilder := new(MessageBuilder)
	features := []string{"feature1", "feature2"}
	raw := messageBuilder.FeaturesEnabled(features)
	res := new(Protocol_Data_Control.Packet)
	err := proto.Unmarshal(raw, res)
	if err != nil || res.GetFeaturesEnabled() == nil {
		t.Errorf("Decoding FeaturesEnabled Packet failed: %v %v", err, res)
	}

	for i, v := range res.GetFeaturesEnabled().GetFeature() {
		if v != features[i] {
			t.Errorf("Requested Features do not match %v %v", res.GetFeaturesEnabled().GetFeature(), features)
		}
	}
}

func TestEnableFeatures(t *testing.T) {
	messageBuilder := new(MessageBuilder)
	features := []string{"feature1", "feature2"}
	raw := messageBuilder.EnableFeatures(features)
	res := new(Protocol_Data_Control.Packet)
	err := proto.Unmarshal(raw, res)
	if err != nil || res.GetEnableFeatures() == nil {
		t.Errorf("Decoding EnableFeatures Packet failed: %v %v", err, res)
	}
	for i, v := range res.GetEnableFeatures().GetFeature() {
		if v != features[i] {
			t.Errorf("Requested Features do not match %v %v", res.GetFeaturesEnabled().GetFeature(), features)
		}
	}
}
