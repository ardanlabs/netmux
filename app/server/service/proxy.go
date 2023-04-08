package service

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ardanlabs.com/netmux/business/grpc/bridge"
	"github.com/ardanlabs.com/netmux/business/grpc/clients/proxy"
)

// Proxy is provided to implement the ProxyServer interface.
func (s *Service) Proxy(proxyServer proxy.Proxy_ProxyServer) error {
	localRecv, err := proxyServer.Recv()
	if err != nil {
		return fmt.Errorf("proxyServer.Recv: %w", err)
	}

	if localRecv.Bridge == nil {
		return errors.New("bridge info not provided by the local proxy")
	}

	if _, err := s.bridges.Get(localRecv.Bridge.Name); err == nil {
		return fmt.Errorf("could not find remote bridge for %q", localRecv.Bridge.Name)
	}

	brd := bridge.New(localRecv.Bridge)

	remoteConn, err := brd.RemoteDial()
	if err != nil {
		err = fmt.Errorf("could not make proxy ep connection to %s: %w", brd, err)
		return err
	}
	s.log.Infof("connected to local proxy: %s", brd)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer func() {
			s.log.Infof("shutting down bridge %q for proxy local to remote", brd.Name)
			wg.Done()
		}()

		for {
			localRecv, err := proxyServer.Recv()
			if err != nil {
				s.log.Infof("error receiving data from local %s: %s", brd.Name, err)
				remoteConn.Close()
				return
			}

			s.log.Infof("receiving data from local %s: bytes[%d]", brd.Name, len(localRecv.Pl))

			if len(localRecv.Pl) > 0 {
				n, err := remoteConn.Write(localRecv.Pl)
				if err != nil {
					s.log.Infof("error sending data to remote %s: %s", brd.Name, err)
					remoteConn.Close()
					return
				}

				s.log.Infof("sent data to remote %s: bytes[%d]", brd.Name, n)
			}
		}
	}()

	go func() {
		defer func() {
			s.log.Infof("shutting down bridge %q for proxy remote to local", brd.Name)
			wg.Done()
		}()

		buf := make([]byte, 4096)

		for {
			n, err := remoteConn.Read(buf)
			if err != nil {
				s.log.Infof("error receiving data from remote %s: %s", brd.Name, err)
				remoteConn.Close()
				return
			}

			s.log.Infof("receiving data from remote %s: bytes[%d]", brd.Name, n)

			connIn := &proxy.ConnIn{
				Pl:  buf[:n],
				Err: "",
			}

			if err := proxyServer.Send(connIn); err != nil {
				s.log.Infof("error sending data to local %s: bytes[%d]", brd.Name, n)
				remoteConn.Close()
				return
			}
		}
	}()

	wg.Wait()
	return nil
}
