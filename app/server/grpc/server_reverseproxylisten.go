package grpc

import (
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/ardanlabs.com/netmux/business/grpc/bridge"
	"github.com/ardanlabs.com/netmux/business/grpc/clients/proxy"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func (s server) ReverseProxyListen(req proxy.Proxy_ReverseProxyListenServer) error {
	var in *proxy.ConnOut
	var err error
	var l net.Listener
	var chErr = make(chan error)
	var b bridge.Bridge

	trace := func(s string, p ...interface{}) {
		logrus.Tracef("ReverseProxyListen: %s", fmt.Sprintf(s, p...))
	}
	warn := func(s string, p ...interface{}) {
		logrus.Warnf("ReverseProxyListen: %s", fmt.Sprintf(s, p...))
	}

	maybeSend := func(err error) {
		select {
		case chErr <- err:
		default:

		}
	}

	closeListener := func() {
		var err error
		if l != nil {
			trace("Closing listener")
			err = l.Close()
		} else {
			trace("Listener is nil, no further action")
		}

		name := "no bridge"
		if !b.IsZero() {
			name = b.Name
		}

		if err != nil {
			warn("error closing listener for %s", name)
		} else {
			trace("closing listener for %s (context cancelled)", name)
		}
		maybeSend(err)
	}

	loopListener := func() {
		for {
			if l == nil {
				maybeSend(nil)
				return
			}
			c, err := l.Accept()
			if err != nil {
				warn("Could accept conn for reverse ep connection to %s: %s",
					b.String(), err.Error())
				l.Close()
				l = nil
				maybeSend(err)
				return
			}

			uid := uuid.NewString()
			s.conns.Set(uid, c)
			trace("REV conn awaiting: %s", uid)
			err = req.Send(&proxy.RevProxyRequest{
				ConnId: uid,
			})
			if err != nil {
				c.Close()
				s.conns.Delete(uid)
				maybeSend(err)
				return
			}
		}
	}

	loopStream := func() {
		for {
			in, err = req.Recv()

			switch {

			case errors.Is(err, io.EOF):
				trace("Received EOF")
				closeListener()
				return
			case err != nil:
				warn("Received error: %s", err.Error())
				maybeSend(err)
				closeListener()
				return
			case in.Bridge == nil:
				err := fmt.Errorf("bridge info not provided at ReverseProxyListen")
				warn("Error: %s", err.Error())
				maybeSend(err)
				return
			case l != nil:
				trace("Received call, but listener still in place")
				return
			default:
				b = bridge.ToBridge(in.Bridge)
				logrus.Tracef("Proxy will listen: %s", b.String())

				l, err = b.RemotePortListener()

				if err != nil {
					logrus.Warnf("Could not make proxy listener for reverse ep connection to %s: %s",
						b.String(), err.Error())
					maybeSend(err)
					closeListener()
				}
				logrus.Debugf("Listening %s %s: OK", b.Name, b.RemotePort)
				go loopListener()
			}
		}
	}

	go loopStream()

	err = <-chErr
	return err
}
