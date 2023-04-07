// Package proxy provides support for running a grpc proxy service.
package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ardanlabs.com/netmux/app/server/auth"
	"github.com/ardanlabs.com/netmux/business/grpc/bridge"
	"github.com/ardanlabs.com/netmux/business/grpc/clients/proxy"
	"github.com/ardanlabs.com/netmux/foundation/db"
	"github.com/ardanlabs.com/netmux/foundation/signal"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// Proxy represents a grpc proxy service.
type Proxy struct {
	proxy.UnsafeProxyServer
	log      *logrus.Logger
	eps      *db.DB[*proxy.Bridge]
	sessions *db.DB[string]
	conns    *db.DB[net.Conn]
	signal   *signal.Signal[*proxy.Bridge]
	grpc     *grpc.Server
	wg       sync.WaitGroup
}

// Start starts the proxy service.
func Start(log *logrus.Logger) (*Proxy, error) {
	prx := Proxy{
		log:      log,
		eps:      db.New[*proxy.Bridge](db.NopReadWriter{}),
		sessions: db.New[string](db.NopReadWriter{}),
		conns:    db.New[net.Conn](db.NopReadWriter{}),
		signal:   signal.New[*proxy.Bridge](),
		grpc: grpc.NewServer(
			grpc.UnaryInterceptor(authUnaryServerInterceptor()),
			grpc.StreamInterceptor(authStreamServerInterceptor()),
		),
	}

	proxy.RegisterProxyServer(prx.grpc, &prx)

	l, err := net.Listen("tcp", "0.0.0.0:48080")
	if err != nil {
		return nil, fmt.Errorf("net.Listen: %w", err)
	}

	prx.wg.Add(1)

	go func() {
		log.Info("proxy: started")
		defer func() {
			log.Info("proxy: shutdown")
			prx.wg.Done()
		}()

		prx.log.Infof("proxy: running server at 0.0.0.0:48080")
		prx.grpc.Serve(l)
	}()

	return &prx, nil
}

// Shutdown requests the proxy service to stop and waits.
func (s *Proxy) Shutdown() {
	s.log.Infof("proxy: starting shutdown")
	defer s.log.Infof("proxy: shutdown")

	s.grpc.GracefulStop()
}

func (s *Proxy) mustEmbedUnimplementedServerServiceServer() {

}

func (s *Proxy) Ver(_ context.Context, _ *proxy.Noop) (*proxy.Noop, error) {
	return &proxy.Noop{}, nil
}

func (s *Proxy) Login(ctx context.Context, req *proxy.LoginReq) (*proxy.StringMsg, error) {
	logrus.Warnf("Will auth: %#v", req)

	str, err := auth.Login(req.User, req.Pass)

	return &proxy.StringMsg{Msg: str}, err
}

func (s *Proxy) Logout(ctx context.Context, req *proxy.StringMsg) (*proxy.Noop, error) {
	err := auth.Logout(req.Msg)
	return &proxy.Noop{}, err
}

func (s *Proxy) GetConfigs(ctx context.Context, req *proxy.Noop) (*proxy.Bridges, error) {
	ret := &proxy.Bridges{Eps: s.eps.KeyValues().Values()}
	return ret, nil
}

func (s *Proxy) StreamConfig(req *proxy.Noop, server proxy.Proxy_StreamConfigServer) error {
	brds := proxy.Bridges{
		Eps: s.eps.KeyValues().Values(),
	}

	if err := server.Send(&brds); err != nil {
		logrus.Warnf("Error sending initial cfg: %s", err.Error())
		return err
	}

	defer func() {
		logrus.Tracef("shutting down signal system")
		s.signal.Shutdown()
	}()

	ch := s.signal.Aquire()
	logrus.Tracef("added cfg listener for local agent")

	for {
		logrus.Tracef("awaiting cfg")

		eps := <-ch
		logrus.Tracef("got cfg")

		brds := proxy.Bridges{
			Eps: []*proxy.Bridge{eps},
		}

		if err := server.Send(&brds); err != nil {
			return err
		}
	}
}

func (s *Proxy) KeepAlive(req *proxy.Noop, res proxy.Proxy_KeepAliveServer) error {
	for {
		res.Send(&proxy.StringMsg{Msg: "PING"})
		time.Sleep(time.Second)
	}
}

func (s *Proxy) Proxy(connectServer proxy.Proxy_ProxyServer) error {
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
