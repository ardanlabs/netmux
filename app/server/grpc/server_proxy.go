package grpc

import (
	"fmt"

	"github.com/ardanlabs.com/netmux/business/grpc/bridge"
	"github.com/ardanlabs.com/netmux/business/grpc/clients/proxy"
	"github.com/sirupsen/logrus"
)

func (s server) Proxy(connectServer proxy.Proxy_ProxyServer) error {
	co, err := connectServer.Recv()
	if err != nil {
		return err
	}
	if co.Bridge == nil {
		return fmt.Errorf("bridge info not provided at Proxy")
	}
	var b bridge.Bridge
	if co.Bridge != nil {
		b = bridge.ToBridge(co.Bridge)
	}
	logrus.Debugf("Got Proxy conn: %s", b.String())

	eps, _ := s.eps.Get(b.Name)
	if eps == nil {
		logrus.Warnf("could not find ep for %s", b.String())
		return fmt.Errorf("could not find ep for %s", b.String())
	}
	c, err := b.RemoteDial()
	if err != nil {
		err = fmt.Errorf("could not make proxy ep connection to %s: %w", b.String(), err)
		logrus.Warnf(err.Error())
		return err
	}
	logrus.Debugf("Connected to: %s", b.String())

	chErr := make(chan error)

	go func() {
		for {
			co, err := connectServer.Recv()
			if err != nil {
				chErr <- fmt.Errorf("error receiving data from local %s: %w", b.Name, err)
				c.Close()
				chErr <- err
				return
			}
			if len(co.Pl) > 0 {
				_, err = c.Write(co.Pl)
				if err != nil {
					c.Close()
					chErr <- fmt.Errorf("error sending data from proxy %s: %w", b.Name, err)
					return
				}
			}
		}
	}()

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := c.Read(buf)
			if err != nil {
				chErr <- fmt.Errorf("error receiving data from proxy %s: %s", b.Name, err.Error())
				c.Close()
				chErr <- err
				return
			}

			err = connectServer.Send(&proxy.ConnIn{
				Pl:  buf[:n],
				Err: "",
			})
			if err != nil {
				chErr <- fmt.Errorf("error sending data to local %s: %w", b.Name, err)
				c.Close()

				chErr <- err
				return
			}

		}
	}()
	err = <-chErr
	if err != nil {
		logrus.Warnf(err.Error())
	}
	return err
}
