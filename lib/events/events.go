package events

import "encoding/json"

type EventType string

const (
	EventTypeDisconnected = "disconnected"
	EventTypeConnected    = "connected"
	EventKATimeOut        = "katimeout"
)

type Event struct {
	Type    string
	Ctx     string
	Payload any
}

func (e *Event) PayloadJson() []byte {
	bs, _ := json.Marshal(e.Payload)
	return bs
}

var listeners []chan *Event

func NewListener() <-chan *Event {
	ret := make(chan *Event)
	listeners = append(listeners, ret)
	return ret
}

func Send(e *Event) {
	for _, v := range listeners {
		select {
		case v <- e:
		default:
		}
	}
}

func Close(c <-chan *Event) {
	for i, v := range listeners {
		if v == c {
			close(v)
			listeners[i] = listeners[len(listeners)-1]
			listeners = listeners[:len(listeners)-1]
		}
	}
}
