package application

import (
	"errors"
	"github.com/s-rah/go-ricochet/channels"
	"github.com/s-rah/go-ricochet/connection"
)

// RicochetApplication bundles many useful constructs that are
// likely standard in a ricochet application
type RicochetApplication struct {
	connection *connection.Connection
}

// NewRicochetApplication ...
func NewRicochetApplication(connection *connection.Connection) *RicochetApplication {
	ra := new(RicochetApplication)
	ra.connection = connection
	return ra
}

// SendMessage ...
func (ra *RicochetApplication) SendChatMessage(message string) error {
	return ra.connection.Do(func() error {
		channel := ra.connection.Channel("im.ricochet.chat", channels.Outbound)
		if channel != nil {
			chatchannel, ok := (*channel.Handler).(*channels.ChatChannel)
			if ok {
				chatchannel.SendMessage(message)
			}
		} else {
			return errors.New("")
		}
		return nil
	})
}
