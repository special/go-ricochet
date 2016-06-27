package utils

import (
	"encoding/binary"
	"errors"
	"net"
	"strconv"
)

// RicochetData is a structure containing the raw data and the channel it the
// message originated on.
type RicochetData struct {
	Channel int32
	Data    []byte
}

// RicochetNetworkInterface abstract operations that interact with ricochet's
// packet layer.
type RicochetNetworkInterface interface {
	Recv(conn net.Conn) ([]byte, error)
	SendRicochetPacket(conn net.Conn, channel int32, data []byte)
	RecvRicochetPackets(conn net.Conn) ([]RicochetData, error)
}

// RicochetNetwork is a concrete implementation of the RicochetNetworkInterface
type RicochetNetwork struct {
}

// Recv reads data from the client, and returns the raw byte array, else error.
func (rn *RicochetNetwork) Recv(conn net.Conn) ([]byte, error) {
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	ret := make([]byte, n)
	copy(ret[:], buf[:])
	return ret, nil
}

// SendRicochetPacket places the data into a structure needed for the client to
// decode the packet and writes the packet to the network.
func (rn *RicochetNetwork) SendRicochetPacket(conn net.Conn, channel int32, data []byte) {
	header := make([]byte, 4+len(data))
	header[0] = byte(len(header) >> 8)
	header[1] = byte(len(header) & 0x00FF)
	header[2] = 0x00
	header[3] = byte(channel)
	copy(header[4:], data[:])
	conn.Write(header)
}

// RecvRicochetPackets returns an array of new messages received from the ricochet client
func (rn *RicochetNetwork) RecvRicochetPackets(conn net.Conn) ([]RicochetData, error) {
	buf, err := rn.Recv(conn)
	if err != nil && len(buf) < 4 {
		return nil, errors.New("failed to retrieve new messages from the client")
	}

	pos := 0
	finished := false
	var datas []RicochetData

	for !finished {
		size := int(binary.BigEndian.Uint16(buf[pos+0 : pos+2]))
		channel := int(binary.BigEndian.Uint16(buf[pos+2 : pos+4]))

		if size < 4 {
			return datas, errors.New("invalid ricochet packet received (size=" + strconv.Itoa(size) + ")")
		}

		if pos+size > len(buf) {
			return datas, errors.New("partial data packet received")
		}

		data := RicochetData{}
		data.Channel = int32(channel)

		if pos+4 >= len(buf) {
			data.Data = make([]byte, 0)
		} else {
			data.Data = buf[pos+4 : pos+size]
		}

		datas = append(datas, data)
		pos += size
		if pos >= len(buf) {
			finished = true
		}
	}
	return datas, nil
}
