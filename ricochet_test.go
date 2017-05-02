package goricochet

import (
  "testing"
  "github.com/s-rah/go-ricochet/utils"
  "net"
  "time"
)


func SimpleServer() {
    ln,_ := net.Listen("tcp", "127.0.0.1:11000")
    conn,_ := ln.Accept()
    b := make([]byte, 4)
    n,err := conn.Read(b)
    if n == 4 && err == nil {
        conn.Write([]byte{0x01})
    }
    conn.Close()
}

func BadVersionNegotiation() {
    ln,_ := net.Listen("tcp", "127.0.0.1:11001")
    conn,_ := ln.Accept()
    // We are already testing negotiation bytes, we don't care, just send a termination.
    conn.Write([]byte{0x00})
    conn.Close()
}

func NotRicochetServer() {
    ln,_ := net.Listen("tcp", "127.0.0.1:11002")
    conn,_ := ln.Accept()
    conn.Close()
}

func TestRicochet(t *testing.T) {
    go SimpleServer()
    // Wait for Server to Initialize
    time.Sleep(time.Second)

    rc,err := Open("127.0.0.1:11000|abcdefghijklmno.onion")
    if err == nil {
        if rc.IsInbound {
            t.Errorf("RicochetConnection declares itself as an Inbound connection after an Outbound attempt...that shouldn't happen")
        }
        return
    }
    t.Errorf("RicochetProtocol: Open Failed: %v", err)
}

func TestBadVersionNegotiation(t*testing.T) {
    go BadVersionNegotiation()
    time.Sleep(time.Second)

    _,err := Open("127.0.0.1:11001|abcdefghijklmno.onion")
    if err != utils.VersionNegotiationFailed {
        t.Errorf("RicochetProtocol: Server Had No Correct Version - Should Have Failed: err = %v", err)
    }
}


func TestNotARicochetServer(t*testing.T) {
    go NotRicochetServer()
    time.Sleep(time.Second)

    _,err := Open("127.0.0.1:11002|abcdefghijklmno.onion")
    if err != utils.VersionNegotiationError {
        t.Errorf("RicochetProtocol: Server Had No Correct Version - Should Have Failed: err = %v", err)
    }
}
