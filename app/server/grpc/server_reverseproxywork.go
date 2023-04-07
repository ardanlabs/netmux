package grpc

import (
	"fmt"

	"github.com/ardanlabs.com/netmux/business/grpc/clients/proxy"
	"github.com/sirupsen/logrus"
)

func (s server) ReverseProxyWork(req proxy.Proxy_ReverseProxyWorkServer) error {
	var err error
	defer func() {
		if err != nil {
			logrus.Debugf("Leaving ReverseProxyWork conn w err: %s", err.Error())
		} else {
			logrus.Debugf("Leaving ReverseProxyWork conn - no err")
		}

	}()
	in, err := req.Recv()
	if err != nil {
		return err
	}
	c, _ := s.conns.Get(in.ConnId)
	if c == nil {
		return fmt.Errorf("connection not found for %s", in.ConnId)
	}

	defer func() {
		s.conns.Delete(in.ConnId)
	}()

	logrus.Debugf("REV conn working: %s", in.ConnId)
	chErr := make(chan error)

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := c.Read(buf)
			if err != nil {
				c.Close()
				chErr <- err
				return
			}
			err = req.Send(&proxy.RevProxyConnOut{
				Pl: buf[:n],
			})
			if err != nil {
				c.Close()
				chErr <- err
				return
			}
		}
	}()

	go func() {
		for {
			in, err = req.Recv()
			if err != nil {
				c.Close()
				chErr <- err
				return
			}
			_, err = c.Write(in.Pl)
			if err != nil {
				c.Close()
				chErr <- err
				return
			}
		}
	}()

	err = <-chErr
	return err
}
