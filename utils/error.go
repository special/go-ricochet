package utils

import (
	"fmt"
)

// Error captures various common ricochet errors
type Error string

func (e Error) Error() string { return string(e) }

// Defining Versions
const (
	VersionNegotiationError  = Error("VersionNegotiationError")
	VersionNegotiationFailed = Error("VersionNegotiationFailed")

	RicochetConnectionClosed = Error("RicochetConnectionClosed")
	RicochetProtocolError    = Error("RicochetProtocolError")

	UnknownChannelTypeError      = Error("UnknownChannelTypeError")
	UnauthorizedChannelTypeError = Error("UnauthorizedChannelTypeError")

	ActionTimedOutError = Error("ActionTimedOutError")

	ClientFailedToAuthenticateError = Error("ClientFailedToAuthenticateError")


        PrivateKeyNotSetError = Error("ClientFailedToAuthenticateError")
)

// CheckError is a helper function for panicing on errors which we need to handle
// but should be very rare e.g. failures deserializing a protobuf object that
// should only happen if there was a bug in the underlying library.
func CheckError(err error) {
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}
}
