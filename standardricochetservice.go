package goricochet

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	//"encoding/binary"
	"encoding/pem"
	"io/ioutil"
)

type StandardRicochetService struct {
	ricochet    *Ricochet
	authHandler map[string]*AuthenticationHandler
	privateKey  *rsa.PrivateKey
	hostname    string
}

func (srs *StandardRicochetService) Init(filename string, hostname string) {
	srs.ricochet = new(Ricochet)
	srs.ricochet.Init(true)
	srs.authHandler = make(map[string]*AuthenticationHandler)
	srs.hostname = hostname

	pemData, err := ioutil.ReadFile(filename)

	if err != nil {
		// 	    r.logger.Print("Error Reading Private Key: ", err)
	}

	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		//		r.logger.Print("No valid PEM data found")
	}

	srs.privateKey, _ = x509.ParsePKCS1PrivateKey(block.Bytes)
}

func (srs *StandardRicochetService) OnConnect(serverHostname string) {
	srs.authHandler[serverHostname] = new(AuthenticationHandler)
	clientCookie := srs.authHandler[serverHostname].GenClientCookie()
	srs.ricochet.Authenticate(1, clientCookie)
}

// OnAuthenticationChallenge constructs a valid authentication challenge to the serverCookie
func (srs *StandardRicochetService) OnAuthenticationChallenge(channelID int32, serverHostname string, serverCookie [16]byte) {
	srs.authHandler[serverHostname].AddServerCookie(serverCookie[:])

	// DER Encode the Public Key
	publickeyBytes, _ := asn1.Marshal(rsa.PublicKey{
		N: srs.privateKey.PublicKey.N,
		E: srs.privateKey.PublicKey.E,
	})

	signature, _ := rsa.SignPKCS1v15(nil, srs.privateKey, crypto.SHA256, srs.authHandler[serverHostname].GenChallenge(srs.hostname, serverHostname))
	// TODO Handle Errors
	signatureBytes := make([]byte, 128)
	copy(signatureBytes[:], signature[:])
	srs.ricochet.SendProof(1, publickeyBytes, signatureBytes)
}

func (srs *StandardRicochetService) Ricochet() *Ricochet {
	return srs.ricochet
}

func (srs *StandardRicochetService) OnAuthenticationResult(channelID int32, serverHostname string, result bool) {

}

func (srs *StandardRicochetService) OnOpenChannelRequest(channelID int32, serverHostname string) {
	srs.ricochet.AckOpenChannel(channelID, true)
}

func (srs *StandardRicochetService) OnOpenChannelRequestAck(channelID int32, serverHostname string, result bool) {
}

func (srs *StandardRicochetService) OnChannelClose(channelID int32, serverHostname string) {
}

func (srs *StandardRicochetService) OnContactRequest(channelID string, serverHostname string, nick string, message string) {
}

func (srs *StandardRicochetService) OnChatMessage(channelID int32, serverHostname string, messageId int32, message string) {
	srs.ricochet.AckChatMessage(channelID, messageId)
}

func (srs *StandardRicochetService) OnChatMessageAck(channelID int32, serverHostname string, messageId int32) {
}
