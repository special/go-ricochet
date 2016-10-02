package utils

import (
	"bytes"
	"io"
	"testing"
	"testing/iotest"
)

// Valid packets and their encoded forms
var packetTests = []struct {
	packet  RicochetData
	encoded []byte
}{
	{RicochetData{1, []byte{}}, []byte{0x00, 0x04, 0x00, 0x01}},
	{RicochetData{65535, []byte{0xDE, 0xAD, 0xBE, 0xEF}}, []byte{0x00, 0x08, 0xFF, 0xFF, 0xDE, 0xAD, 0xBE, 0xEF}},
	{RicochetData{2, make([]byte, 65531)}, append([]byte{0xFF, 0xFF, 0x00, 0x02}, make([]byte, 65531)...)},
}

// Test sending valid packets
func TestSendRicochetPacket(t *testing.T) {
	rni := RicochetNetwork{}
	for _, td := range packetTests {
		var buf bytes.Buffer
		err := rni.SendRicochetPacket(&buf, td.packet.Channel, td.packet.Data)
		if err != nil {
			t.Errorf("Error sending packet %v: %v", td.packet, err)
		} else if !bytes.Equal(buf.Bytes(), td.encoded) {
			t.Errorf("Expected serialized packet %x but got %x", td.encoded, buf.Bytes())
		}
	}
}

// Test sending invalid packets
func TestSendRicochetPacket_Invalid(t *testing.T) {
	rni := RicochetNetwork{}
	invalidPackets := []RicochetData{
		RicochetData{-1, []byte{}},
		RicochetData{65536, []byte{}},
		RicochetData{0, make([]byte, 65532)},
	}

	for _, td := range invalidPackets {
		var buf bytes.Buffer
		err := rni.SendRicochetPacket(&buf, td.Channel, td.Data)
		// Expect error
		if err == nil {
			t.Errorf("Expected error when sending invalid packet %v", td)
		}
	}
}

// Test receiving valid packets
func TestRecvRicochetPacket(t *testing.T) {
	var buf bytes.Buffer
	for _, td := range packetTests {
		if _, err := buf.Write(td.encoded); err != nil {
			t.Error(err)
			return
		}
	}

	// Use a HalfReader to test behavior on short socket reads also
	reader := iotest.HalfReader(&buf)
	rni := RicochetNetwork{}

	for _, td := range packetTests {
		packet, err := rni.RecvRicochetPacket(reader)
		if err != nil {
			t.Errorf("Error receiving packet %v: %v", td.packet, err)
			return
		} else if !packet.Equals(td.packet) {
			t.Errorf("Expected unserialized packet %v but got %v", td.packet, packet)
		}
	}

	if packet, err := rni.RecvRicochetPacket(reader); err != io.EOF {
		if err != nil {
			t.Errorf("Expected EOF on packet stream but received error: %v", err)
		} else {
			t.Errorf("Expected EOF but received packet: %v", packet)
		}
	}
}

// Test receiving invalid packets
func TestRecvRicochetPacket_Invalid(t *testing.T) {
	rni := RicochetNetwork{}
	invalidPackets := [][]byte{
		[]byte{0x00, 0x00, 0x00, 0x00},
		[]byte{0x00, 0x03, 0x00, 0x00},
		[]byte{0xff},
		[]byte{0x00, 0x06, 0x00, 0x00, 0x00},
		[]byte{},
	}

	for _, td := range invalidPackets {
		buf := bytes.NewBuffer(td)
		packet, err := rni.RecvRicochetPacket(buf)
		// Expect error
		if err == nil {
			t.Errorf("Expected error when sending invalid packet %x, got packet %v", td, packet)
		}
	}
}
