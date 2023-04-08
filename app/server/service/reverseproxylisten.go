package service

import (
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/ardanlabs.com/netmux/business/grpc/bridge"
	"github.com/ardanlabs.com/netmux/business/grpc/clients/proxy"
	"github.com/google/uuid"
)

// ReverseProxyListen is provided to implement the ProxyServer interface.
func (s *Service) ReverseProxyListen(listenServer proxy.Proxy_ReverseProxyListenServer) error {
	for {
		recv, err := listenServer.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				s.log.Info("reverseProxyListen: EOF")
				return nil
			}

			s.log.Info("reverseProxyListen: ERROR: %s", err)
			return err
		}

		if recv.Bridge == nil {
			err := errors.New("reverseProxyListen: bridge info not provided")
			s.log.Info(err)
			return err
		}

		brd := bridge.New(recv.Bridge)

		if err := s.listener(listenServer, brd); err != nil {
			s.log.Infof("reverseProxyListen: listener: %s", err)
			return err
		}
	}
}

func (s *Service) listener(listenServer proxy.Proxy_ReverseProxyListenServer, brd bridge.Bridge) error {
	s.log.Infof("reverseProxyListen: listening name[%s] remote[%s]", brd.Name, brd.RemotePort)

	listener, err := brd.RemotePortListener()
	if err != nil {
		s.log.Infof("reverseProxyListen: could not make proxy listener for reverse ep connection to %s: %s", brd, err)
		return err
	}

	s.updateReverseProxyLister(listener)
	defer func() {
		listener.Close()
		s.updateReverseProxyLister(nil)
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("listener.Accept: %w", err)
		}

		uid := uuid.NewString()
		s.conns.Set(uid, conn)

		s.log.Infof("reverseProxyListen: connection accepted: %s", uid)

		req := proxy.RevProxyRequest{
			ConnId: uid,
		}

		if err := listenServer.Send(&req); err != nil {
			conn.Close()
			s.conns.Delete(uid)
			return fmt.Errorf("listenServer.Send: %w", err)
		}
	}
}

func (s *Service) updateReverseProxyLister(listener net.Listener) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.reverseProxyLister = listener
}

func (s *Service) shutdownReverseProxyListen() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.reverseProxyLister != nil {
		s.reverseProxyLister.Close()
	}
}
