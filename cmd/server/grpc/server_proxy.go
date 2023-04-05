package grpc

import (
	"fmt"

	"github.com/sirupsen/logrus"
	pb "go.digitalcircle.com.br/dc/netmux/lib/proto/server"
	"go.digitalcircle.com.br/dc/netmux/lib/types"
)

func (s server) Proxy(connectServer pb.NXProxy_ProxyServer) error {
	co, err := connectServer.Recv()
	if err != nil {
		return err
	}
	if co.Bridge == nil {
		return fmt.Errorf("bridge info not provided at Proxy")
	}
	var bridge = &types.Bridge{}
	if co.Bridge != nil {
		bridge.FromPb(co.Bridge)
	}
	logrus.Debugf("Got Proxy conn: %s", bridge.String())

	eps, _ := s.eps.Get(bridge.Name)
	if eps == nil {
		logrus.Warnf("could not find ep for %s", bridge.String())
		return fmt.Errorf("could not find ep for %s", bridge.String())
	}
	c, err := bridge.DialRemote()
	if err != nil {
		err = fmt.Errorf("could not make proxy ep connection to %s: %w", bridge.String(), err)
		logrus.Warnf(err.Error())
		return err
	}
	logrus.Debugf("Connected to: %s", bridge.String())

	chErr := make(chan error)

	go func() {
		for {
			co, err := connectServer.Recv()
			if err != nil {
				chErr <- fmt.Errorf("error receiving data from local %s: %w", bridge.Name, err)
				c.Close()
				chErr <- err
				return
			}
			if len(co.Pl) > 0 {
				_, err = c.Write(co.Pl)
				if err != nil {
					c.Close()
					chErr <- fmt.Errorf("error sending data from proxy %s: %w", bridge.Name, err)
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
				chErr <- fmt.Errorf("error receiving data from proxy %s: %s", bridge.Name, err.Error())
				c.Close()
				chErr <- err
				return
			}

			err = connectServer.Send(&pb.ConnIn{
				Pl:  buf[:n],
				Err: "",
			})
			if err != nil {
				chErr <- fmt.Errorf("error sending data to local %s: %w", bridge.Name, err)
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
