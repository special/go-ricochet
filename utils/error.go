package utils

import (
	"fmt"
	"log"
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


        PrivateKeyNotSetError = Error("PrivateKeyNotSetError")
)

// RecoverFromError doesn't really recover from anything....see comment below
func RecoverFromError() {
	if r := recover(); r != nil {
		// This should only really happen if there is a failure de/serializing. If
		// this does happen then we currently error. In the future we might be
		// able to make this nicer.
		log.Fatalf("Recovered from panic() - this really shouldn't happen. Reason: %v", r)
	}
}

// CheckError is a helper function for panicing on errors which we need to handle
// but should be very rare e.g. failures deserializing a protobuf object that
// should only happen if there was a bug in the underlying library.
func CheckError(err error) {
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}
}
