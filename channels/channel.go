package channels

// Direction indicated whether we or the remote peer opened the channel
type Direction int

const (
	// Inbound indcates the channel was opened by the remote peer
	Inbound Direction = iota
	// Outbound indicated the channel was opened by us
	Outbound
)

// AuthChannelResult captures the result of an authentication flow
type AuthChannelResult struct {
	Hostname       string
	Accepted       bool
	IsKnownContact bool
}

// Channel holds the state of a channel on an open connection
type Channel struct {
	ID int32

	Type           string
	Direction      Direction
	Handler        Handler
	Pending        bool
	ServerHostname string
	ClientHostname string

	// Functions for updating the underlying Connection
	SendMessage           func([]byte)
	CloseChannel          func()
	DelegateAuthorization func()
}
