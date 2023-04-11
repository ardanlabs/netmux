package grpc

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ardanlabs.com/netmux/business/grpc/bridge"
	"github.com/ardanlabs.com/netmux/business/grpc/cluster"
)

// Proxy is provided to implement the ProxyServer interface.
func (g *GRPC) Proxy(proxyServer cluster.Cluster_ProxyServer) error {
	recv, err := proxyServer.Recv()
	if err != nil {
		return fmt.Errorf("proxyServer.Recv: %w", err)
	}

	if recv.Bridge == nil {
		return errors.New("bridge info not provided by the local proxy")
	}

	if _, err := g.bridges.Get(recv.Bridge.Name); err == nil {
		return fmt.Errorf("could not find remote bridge for %q", recv.Bridge.Name)
	}

	brd := bridge.New(recv.Bridge)

	conn, err := brd.RemoteDial()
	if err != nil {
		err = fmt.Errorf("could not make proxy ep connection to %s: %w", brd, err)
		return err
	}
	g.log.Infof("connected to local proxy: %s", brd)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer func() {
			g.log.Infof("shutting down bridge %q for proxy local to remote", brd.Name)
			wg.Done()
		}()

		for {
			recv, err := proxyServer.Recv()
			if err != nil {
				g.log.Infof("error receiving data from local %s: %s", brd.Name, err)
				conn.Close()
				return
			}

			g.log.Infof("receiving data from local %s: bytes[%d]", brd.Name, len(recv.Pl))

			if len(recv.Pl) > 0 {
				n, err := conn.Write(recv.Pl)
				if err != nil {
					g.log.Infof("error sending data to remote %s: %s", brd.Name, err)
					conn.Close()
					return
				}

				g.log.Infof("sent data to remote %s: bytes[%d]", brd.Name, n)
			}
		}
	}()

	go func() {
		defer func() {
			g.log.Infof("shutting down bridge %q for proxy remote to local", brd.Name)
			wg.Done()
		}()

		buf := make([]byte, 4096)

		for {
			n, err := conn.Read(buf)
			if err != nil {
				g.log.Infof("error receiving data from remote %s: %s", brd.Name, err)
				conn.Close()
				return
			}

			g.log.Infof("receiving data from remote %s: bytes[%d]", brd.Name, n)

			connIn := &cluster.ConnIn{
				Pl:  buf[:n],
				Err: "",
			}

			if err := proxyServer.Send(connIn); err != nil {
				g.log.Infof("error sending data to local %s: bytes[%d]", brd.Name, n)
				conn.Close()
				return
			}
		}
	}()

	wg.Wait()
	return nil
}
