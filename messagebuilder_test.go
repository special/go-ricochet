package goricochet

import "testing"

func TestOpenChatChannel(t *testing.T) {
	messageBuilder := new(MessageBuilder)
	_, err := messageBuilder.OpenChannel(1, "im.ricochet.chat")
	if err != nil {
		t.Errorf("Error building open chat channel message: %s", err)
	}
	// TODO: More Indepth Test Of Output
}

func TestOpenContactRequestChannel(t *testing.T) {
	messageBuilder := new(MessageBuilder)
	_, err := messageBuilder.OpenContactRequestChannel(3, "Nickname", "Message")
	if err != nil {
		t.Errorf("Error building open contact request channel message: %s", err)
	}
	// TODO: More Indepth Test Of Output
}

func TestOpenAuthenticationChannel(t *testing.T) {
	messageBuilder := new(MessageBuilder)
	_, err := messageBuilder.OpenAuthenticationChannel(1, [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	if err != nil {
		t.Errorf("Error building open authentication channel message: %s", err)
	}
	// TODO: More Indepth Test Of Output
}

func TestChatMessage(t *testing.T) {
	messageBuilder := new(MessageBuilder)
	_, err := messageBuilder.ChatMessage("Hello World", 0)
	if err != nil {
		t.Errorf("Error building chat message: %s", err)
	}
	// TODO: More Indepth Test Of Output
}
