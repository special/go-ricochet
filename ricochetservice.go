package goricochet

type RicochetService interface {
	OnConnect(id string) error
	OnContactRequest(id string) error
	OnMessage(id string, message string, channel int) error
}
