package utils

import "fmt"
import "log"

func RecoverFromError() {
	if r := recover(); r != nil {
		// This should only really happen if there is a failure de/serializing. If
		// this does happen then we currently error. In the future we might be
		// able to make this nicer.
		log.Fatalf("Recovered from panic() - this really shouldn't happen. Reason: %v", r)
	}
}

func CheckError(err error) {
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}
}
