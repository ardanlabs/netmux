package chmux

import (
	"github.com/sirupsen/logrus"
	"sync"
)

type ChMux[T any] struct {
	mx    sync.RWMutex
	chans []chan T
}

func (c *ChMux[T]) Len() int {
	return len(c.chans)
}
func (c *ChMux[T]) New() <-chan T {
	c.mx.Lock()
	defer c.mx.Unlock()
	ret := make(chan T)
	c.chans = append(c.chans, ret)
	logrus.Tracef("added ch to chmux - total: %v", len(c.chans))
	return ret
}

func (c *ChMux[T]) Close(ac <-chan T) {
	c.mx.Lock()
	defer c.mx.Unlock()
	for i := range c.chans {
		if c.chans[i] == ac {
			rc := c.chans[i]
			c.chans[i] = c.chans[len(c.chans)-1]
			c.chans = c.chans[:len(c.chans)-1]
			close(rc)
		}
	}
}

func (c *ChMux[T]) Broadcast(t T) {
	logrus.Tracef("bcasting chmux - len: %v", c.Len())
	c.mx.RLock()
	defer c.mx.RUnlock()
	for i := range c.chans {
		ch := c.chans[i]
		select {
		case ch <- t:
		default:
			logrus.Infof("blocked while writting to chan %v", i)
		}
	}
}

func New[T any]() *ChMux[T] {
	ret := &ChMux[T]{
		chans: make([]chan T, 0),
	}
	return ret
}
