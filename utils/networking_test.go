package utils

import "testing"
import "net"
import "time"

type MockConn struct {
	Written    []byte
	MockOutput []byte
}

func (mc *MockConn) Read(b []byte) (int, error) {
	copy(b[:], mc.MockOutput[:])
	return len(mc.MockOutput), nil
}

func (mc *MockConn) Write(written []byte) (int, error) {
	mc.Written = written
	return 0, nil
}

func (mc *MockConn) LocalAddr() net.Addr {
	return nil
}

func (mc *MockConn) RemoteAddr() net.Addr {
	return nil
}

func (mc *MockConn) Close() error {
	return nil
}

func (mc *MockConn) SetDeadline(t time.Time) error {
	return nil
}

func (mc *MockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (mc *MockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func TestSentRicochetPacket(t *testing.T) {
	conn := new(MockConn)
	rni := RicochetNetwork{}
	rni.SendRicochetPacket(conn, 1, []byte{})
	if len(conn.Written) != 4 && conn.Written[0] != 0x00 && conn.Written[1] != 0x00 && conn.Written[2] != 0x01 && conn.Written[3] != 0x00 {
		t.Errorf("Output of SentRicochetPacket was Unexpected: %x", conn.Written)
	}
}

func TestRecv(t *testing.T) {
	conn := new(MockConn)
	conn.MockOutput = []byte{0xDE, 0xAD, 0xBE, 0xEF}
	rni := RicochetNetwork{}
	buf, err := rni.Recv(conn)
	if err != nil || len(buf) != 4 || buf[0] != 0xDE || buf[1] != 0xAD || buf[2] != 0xBE || buf[3] != 0xEF {
		t.Errorf("Output of Recv was Unexpected: %x", buf)
	}
}

func TestRecvRicochetPacket(t *testing.T) {
	conn := new(MockConn)
	conn.MockOutput = []byte{00, 0x04, 0x00, 0x01}

	rni := RicochetNetwork{}
	rp, err := rni.RecvRicochetPackets(conn)

	if err != nil {
		t.Errorf("error extracting ricochet packets: %v", err)
		return
	}

	if len(rp) != 1 {
		t.Errorf("unexpected number of ricochet packets: %d", len(rp))
	} else {
		if rp[0].Channel != 1 {
			t.Errorf("channel number is Unexpected expected 1: %d", rp[0].Channel)
		}

		if len(rp[0].Data) != 0 {
			t.Errorf("expected emptry packet, instead got %x", rp[0].Data)
		}
	}

}

func TestRecvRicochetPacketInvalid(t *testing.T) {
	conn := new(MockConn)
	conn.MockOutput = []byte{00, 0x01, 0x00, 0x01}

	rni := RicochetNetwork{}
	_, err := rni.RecvRicochetPackets(conn)

	if err == nil {
		t.Errorf("recv should have errored due to invalid packets %v", err)
	}

	conn.MockOutput = []byte{00, 0x0A, 0x00, 0x01}

	_, err = rni.RecvRicochetPackets(conn)

	if err == nil {
		t.Errorf("recv should have errored due to invalid packets %v", err)
	}

}

func TestRecvRicochetPacketLong(t *testing.T) {
	conn := new(MockConn)
	conn.MockOutput = []byte{0x00, 0x08, 0x00, 0xFF, 0xDE, 0xAD, 0xBE, 0xEF}

	rni := RicochetNetwork{}
	rp, err := rni.RecvRicochetPackets(conn)

	if err != nil {
		t.Errorf("error extracting ricochet packets: %v", err)
		return
	}

	if len(rp) != 1 {
		t.Errorf("unexpected number of ricochet packets: %d", len(rp))
	} else {
		if rp[0].Channel != 255 {
			t.Errorf("channel number is Unexpected expected 255 got: %d", rp[0].Channel)
		}

		if len(rp[0].Data) != 4 || rp[0].Data[0] != 0xDE || rp[0].Data[1] != 0xAD || rp[0].Data[2] != 0xBE || rp[0].Data[3] != 0xEF {
			t.Errorf("expected 0xDEADBEEF packet, instead got %x", rp[0].Data)
		}
	}

}

func TestRecvRicochetPacketMultiplex(t *testing.T) {
	conn := new(MockConn)
	conn.MockOutput = []byte{0x00, 0x04, 0x00, 0x01, 0x00, 0x08, 0x00, 0xFF, 0xDE, 0xAD, 0xBE, 0xEF}

	rni := RicochetNetwork{}
	rp, err := rni.RecvRicochetPackets(conn)

	if err != nil {
		t.Errorf("error extracting ricochet packets: %v", err)
		return
	}

	if len(rp) != 2 {
		t.Errorf("unexpected number of ricochet packets, expected 2 gt: %d", len(rp))
	} else {

		if rp[0].Channel != 1 {
			t.Errorf("channel number is Unexpected expected 1: %d", rp[0].Channel)
		}

		if len(rp[0].Data) != 0 {
			t.Errorf("expected empty packet, instead got %x", rp[0].Data)
		}

		if rp[1].Channel != 255 {
			t.Errorf("channel number is Unexpected expected 255 got: %d", rp[0].Channel)
		}

		if len(rp[1].Data) != 4 || rp[1].Data[0] != 0xDE || rp[1].Data[1] != 0xAD || rp[1].Data[2] != 0xBE || rp[1].Data[3] != 0xEF {
			t.Errorf("expected 0xDEADBEEF packet, instead got %x", rp[0].Data)
		}
	}

}
