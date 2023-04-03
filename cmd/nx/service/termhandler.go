package service

import (
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type _termHandler struct {
	mx       sync.Mutex
	handlers map[string]func() error
	Ctx      context.Context
	Cancel   func()
}

func (t *_termHandler) add(f func() error) string {
	s := uuid.NewString()
	t.mx.Lock()
	defer t.mx.Unlock()
	t.handlers[s] = f
	return s
}

func (t *_termHandler) remove(s string) {
	t.mx.Lock()
	defer t.mx.Unlock()
	delete(t.handlers, s)
}
func (t *_termHandler) TerminateSome(s ...string) {
	defer func() {
		for _, as := range s {
			t.remove(as)
		}
	}()

	for _, as := range s {
		v, ok := t.handlers[as]
		if ok {
			err := v()
			if err != nil {
				logrus.Warnf("error terminating: %s: %s", as, err.Error())
			}
		}
	}
}
func (t *_termHandler) terminate() {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.Cancel()
	for k, v := range t.handlers {
		err := v()
		if err != nil {
			logrus.Warnf("Error terminating %s: %s", k, err.Error())
		}
	}
}

var ctx, cancel = context.WithCancel(context.Background())
var TermHanlder = _termHandler{handlers: make(map[string]func() error), Ctx: ctx, Cancel: cancel}

func init() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		<-ch
		TermHanlder.terminate()
	}()
}
