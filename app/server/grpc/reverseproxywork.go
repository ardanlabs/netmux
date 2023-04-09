package grpc

import (
	"fmt"
	"sync"

	"github.com/ardanlabs.com/netmux/business/grpc/clients/proxy"
)

// ReverseProxyWork is provided to implement the ProxyServer interface.
func (s *Service) ReverseProxyWork(workServer proxy.Proxy_ReverseProxyWorkServer) error {
	recv, err := workServer.Recv()
	if err != nil {
		s.log.Info("reverseProxyWork: workServer.Recv: ERROR: %s", err)
		return fmt.Errorf("workServer.Recv: %w", err)
	}

	conn, err := s.conns.Get(recv.ConnId)
	if err != nil {
		s.log.Info("reverseProxyWork: s.conns.Get: recv.ConnId %s: ERROR: %s", recv.ConnId, err)
		return fmt.Errorf("s.conns.Get: recv.ConnId %s: %w", recv.ConnId, err)
	}

	defer func() {
		s.conns.Delete(recv.ConnId)
	}()

	s.log.Infof("reverseProxyWork: connection accepted: %s", recv.ConnId)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer func() {
			s.log.Infof("shutting down %q for proxy local to remote", recv.ConnId)
			wg.Done()
		}()

		for {
			localRecv, err := workServer.Recv()
			if err != nil {
				s.log.Infof("error receiving data from local %s: %s", recv.ConnId, err)
				conn.Close()
				return
			}

			s.log.Infof("receiving data from local %s: bytes[%d]", recv.ConnId, len(localRecv.Pl))

			if len(localRecv.Pl) > 0 {
				n, err := conn.Write(localRecv.Pl)
				if err != nil {
					s.log.Infof("error sending data to remote %s: %s", recv.ConnId, err)
					conn.Close()
					return
				}

				s.log.Infof("sent data to remote %s: bytes[%d]", recv.ConnId, n)
			}
		}
	}()

	go func() {
		defer func() {
			s.log.Infof("shutting down %q for proxy remote to local", recv.ConnId)
			wg.Done()
		}()

		buf := make([]byte, 4096)

		for {
			n, err := conn.Read(buf)
			if err != nil {
				s.log.Infof("error receiving data from remote %s: %s", recv.ConnId, err)
				conn.Close()
				return
			}

			s.log.Infof("receiving data from remote %s: bytes[%d]", recv.ConnId, n)

			connOut := &proxy.RevProxyConnOut{
				Pl: buf[:n],
			}

			if err := workServer.Send(connOut); err != nil {
				s.log.Infof("error sending data to local %s: bytes[%d]", recv.ConnId, n)
				conn.Close()
				return
			}
		}
	}()

	wg.Wait()
	return nil
}
