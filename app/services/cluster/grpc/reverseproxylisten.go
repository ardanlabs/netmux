package grpc

import (
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/ardanlabs.com/netmux/business/grpc/bridge"
	"github.com/ardanlabs.com/netmux/business/grpc/cluster"
	"github.com/google/uuid"
)

// ReverseProxyListen is provided to implement the ProxyServer interface.
func (g *GRPC) ReverseProxyListen(listenServer cluster.Cluster_ReverseProxyListenServer) error {
	for {
		recv, err := listenServer.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				g.log.Info("reverseProxyListen: EOF")
				return nil
			}

			g.log.Infof("reverseProxyListen: listenServer.Recv: ERROR: %s", err)
			return fmt.Errorf("listenServer.Recv: %w", err)
		}

		if recv.Bridge == nil {
			err := errors.New("bridge info not provided")
			g.log.Infof("reverseProxyListen: ERROR: %s", err)
			return err
		}

		brd := bridge.New(recv.Bridge)

		if err := g.listener(listenServer, brd); err != nil {
			g.log.Infof("reverseProxyListen: s.listener: ERROR: %s", err)
			return fmt.Errorf("s.listener: %w", err)
		}
	}
}

func (g *GRPC) listener(listenServer cluster.Cluster_ReverseProxyListenServer, brd bridge.Bridge) error {
	g.log.Infof("reverseProxyListen: listening name[%s] remote[%s]", brd.Name, brd.RemotePort)

	listener, err := brd.RemotePortListener()
	if err != nil {
		return fmt.Errorf("brd.RemotePortListener: %w", err)
	}

	g.updateReverseProxyLister(listener)
	defer func() {
		listener.Close()
		g.updateReverseProxyLister(nil)
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("listener.Accept: %w", err)
		}

		uid := uuid.NewString()
		g.conns.Set(uid, conn)

		g.log.Infof("reverseProxyListen: connection accepted: %s", uid)

		req := cluster.RevProxyRequest{
			ConnId: uid,
		}

		if err := listenServer.Send(&req); err != nil {
			conn.Close()
			g.conns.Delete(uid)
			return fmt.Errorf("listenServer.Send: %w", err)
		}
	}
}

func (g *GRPC) updateReverseProxyLister(listener net.Listener) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.reverseProxyLister = listener
}

func (g *GRPC) shutdownReverseProxyListen() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.reverseProxyLister != nil {
		g.reverseProxyLister.Close()
	}
}
