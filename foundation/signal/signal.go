// Package signal provides support for subscribing to receive a value on a
// given signal that is represented by an instance of a signal value.
package signal

import (
	"sync"
)

// Signal maintains a set of channels that can receive values that are
// broadcast through the instance.
type Signal[T any] struct {
	mu       sync.RWMutex
	channels []chan T
}

// New constructs a Broadcast for use to signal values across G boundaries.
func New[T any]() *Signal[T] {
	return &Signal[T]{
		channels: make([]chan T, 0),
	}
}

// TODO: We really need to know that the G has shutdown. If not we could cause
//       problems. I rather accept a function to execute when a broadcast occurs
//       then we can properly shut things down.

// Shutdown closes all the known channels for this instance.
func (s *Signal[T]) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, chn := range s.channels {
		close(chn)
	}

	s.channels = make([]chan T, 0)
}

// TODO: If we keep this API, which I am against, the user needs to give us the
//       channel. They may want to use a buffered channel so they don't lose N
//       number of broadcasts.

// Aquire returns a non-buffered channel that a G can used to receive a signal
// that is being broadcast through the instance.
func (s *Signal[T]) Aquire() <-chan T {
	s.mu.Lock()
	defer s.mu.Unlock()

	chn := make(chan T)
	s.channels = append(s.channels, chn)

	return chn
}

// Broadcast signals all the existing channels in this instance the specified value.
func (s *Signal[T]) Broadcast(t T) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, chn := range s.channels {
		select {
		case chn <- t:
		default:
		}
	}
}
