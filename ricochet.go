package goricochet

import (
    "github.com/s-rah/go-ricochet/utils"
    "github.com/s-rah/go-ricochet/connection"
    "io"
    "net"
    "log"
)
// Open establishes a protocol session on an established net.Conn, and returns a new
// OpenConnection instance representing this connection. On error, the connection
// will be closed. This function blocks until version negotiation has completed.
// The application should call Process() on the returned OpenConnection to continue
// handling protocol messages.
func Open(remoteHostname string) (*connection.Connection, error) {
    networkResolver := utils.NetworkResolver{}
    log.Printf("Connecting...")
    conn, remoteHostname, err := networkResolver.Resolve(remoteHostname)

    if err != nil {
        return nil, err
    }

    log.Printf("Connected...negotiating version")
    rc, err := negotiateVersion(conn, remoteHostname)
    if err != nil {
        conn.Close()
        return nil, err
    }
        log.Printf("Connected...negotiated version")
    return rc, nil
}


// negotiate version takes an open network connection and executes
// the ricochet version negotiation procedure.
func negotiateVersion(conn net.Conn, remoteHostname string) (*connection.Connection, error) {
    versions := []byte{0x49, 0x4D, 0x01, 0x01}
    if n, err := conn.Write(versions); err != nil || n < len(versions) {
        return nil, utils.VersionNegotiationError
    }

    res := make([]byte, 1)
    if _, err := io.ReadAtLeast(conn, res, len(res)); err != nil {
        return nil, utils.VersionNegotiationError
    }

    if res[0] != 0x01 {
        return nil, utils.VersionNegotiationFailed
    }
    rc := connection.NewOutboundConnection(conn,remoteHostname)
    return rc, nil
}


