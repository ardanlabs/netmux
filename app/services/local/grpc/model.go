package grpc

import "encoding/json"

const (
	eventTypeDisconnected = "disconnected"
	eventTypeConnected    = "connected"
	eventKATimeOut        = "katimeout"
)

// =============================================================================

type event struct {
	Type    string
	Ctx     string
	Payload any
}

func (e *event) PayloadJson() []byte {
	bs, _ := json.Marshal(e.Payload)
	return bs
}
