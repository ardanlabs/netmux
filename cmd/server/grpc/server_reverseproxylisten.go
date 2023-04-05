package grpc

import (
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	pb "go.digitalcircle.com.br/dc/netmux/lib/proto/server"
	"go.digitalcircle.com.br/dc/netmux/lib/types"
)

func (s server) ReverseProxyListen(req pb.NXProxy_ReverseProxyListenServer) error {
	var in *pb.ConnOut
	var err error
	var l net.Listener
	var chErr = make(chan error)
	var bridge = &types.Bridge{}

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
		if bridge != nil {
			name = bridge.Name
		}

		if err != nil {
			warn("error closing listener for %s", name)
		} else {
			trace("closing listener for %s (context cancelled)", name)
		}
		maybeSend(err)
		return
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
					bridge.String(), err.Error())
				l.Close()
				l = nil
				maybeSend(err)
				return
			}

			uid := uuid.NewString()
			s.conns.Set(uid, c)
			trace("REV conn awaiting: %s", uid)
			err = req.Send(&pb.RevProxyRequest{
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
				bridge.FromPb(in.Bridge)
				logrus.Tracef("Proxy will listen: %s", bridge.String())

				l, err = bridge.ListenerOnRemoteHost()

				if err != nil {
					logrus.Warnf("Could not make proxy listener for reverse ep connection to %s: %s",
						bridge.String(), err.Error())
					maybeSend(err)
					closeListener()
				}
				logrus.Debugf("Listening %s %s: OK", bridge.Name, bridge.RemotePort)
				go loopListener()
			}
		}
	}

	go loopStream()

	err = <-chErr
	return err
}
