package application

import (
	"crypto/rsa"
	"github.com/yawning/bulb"
	"net"
)

func SetupOnion(proxyServer string, authentication string, pk *rsa.PrivateKey, onionport uint16) (net.Listener, error) {
	c, err := bulb.Dial("tcp4", proxyServer)
	if err != nil {
		return nil, err
	}

	if err := c.Authenticate(authentication); err != nil {
		return nil, err
	}

	cfg := &bulb.NewOnionConfig{
		DiscardPK:  true,
		PrivateKey: pk,
	}

	return c.NewListener(cfg, onionport)
}
