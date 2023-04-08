// Package service provides support for running a grpc proxy service.
package service

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ardanlabs.com/netmux/app/server/auth"
	"github.com/ardanlabs.com/netmux/business/grpc/clients/proxy"
	"github.com/ardanlabs.com/netmux/foundation/db"
	"github.com/ardanlabs.com/netmux/foundation/signal"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// Service represents a grpc proxy service.
type Service struct {
	proxy.UnsafeProxyServer
	log                *logrus.Logger
	grpc               *grpc.Server
	signal             *signal.Signal[*proxy.Bridge]
	bridges            *db.DB[*proxy.Bridge]
	sessions           *db.DB[string]
	conns              *db.DB[net.Conn]
	wg                 sync.WaitGroup
	mu                 sync.Mutex
	reverseProxyLister net.Listener
}

// Start starts the proxy service.
func Start(log *logrus.Logger) (*Service, error) {
	srv := Service{
		log: log,
		grpc: grpc.NewServer(
			grpc.UnaryInterceptor(authUnaryServerInterceptor()),
			grpc.StreamInterceptor(authStreamServerInterceptor()),
		),
		signal:   signal.New[*proxy.Bridge](),
		bridges:  db.New[*proxy.Bridge](db.NopReadWriter{}),
		sessions: db.New[string](db.NopReadWriter{}),
		conns:    db.New[net.Conn](db.NopReadWriter{}),
	}

	proxy.RegisterProxyServer(srv.grpc, &srv)

	l, err := net.Listen("tcp", "0.0.0.0:48080")
	if err != nil {
		return nil, fmt.Errorf("net.Listen: %w", err)
	}

	srv.wg.Add(1)

	go func() {
		log.Info("proxy: started")
		defer func() {
			log.Info("proxy: shutdown")
			srv.wg.Done()
		}()

		srv.log.Infof("proxy: running server at 0.0.0.0:48080")
		srv.grpc.Serve(l)
	}()

	return &srv, nil
}

// Shutdown requests the proxy service to stop and waits.
func (s *Service) Shutdown() {
	s.log.Infof("proxy: starting shutdown")
	defer s.log.Infof("proxy: shutdown")

	s.shutdownReverseProxyListen()
	s.grpc.GracefulStop()
}

// =============================================================================

// Ver(context.Context, *Noop) (*Noop, error)
// Proxy(Proxy_ProxyServer) error
// ReverseProxyListen(Proxy_ReverseProxyListenServer) error
// ReverseProxyWork(Proxy_ReverseProxyWorkServer) error
// Ping(context.Context, *StringMsg) (*StringMsg, error)
// PortScan(context.Context, *StringMsg) (*StringMsg, error)
// Nc(context.Context, *StringMsg) (*StringMsg, error)
// SpeedTest(context.Context, *StringMsg) (*BytesMsg, error)
// KeepAlive(*Noop, Proxy_KeepAliveServer) error
// GetConfigs(context.Context, *Noop) (*Bridges, error)
// StreamConfig(*Noop, Proxy_StreamConfigServer) error
// Login(context.Context, *LoginReq) (*StringMsg, error)
// Logout(context.Context, *StringMsg) (*Noop, error)
// mustEmbedUnimplementedProxyServer()

// Ver is provided to implement the ProxyServer interface.
func (s *Service) Ver(_ context.Context, _ *proxy.Noop) (*proxy.Noop, error) {
	return &proxy.Noop{}, nil
}

func (s *Service) Login(ctx context.Context, req *proxy.LoginReq) (*proxy.StringMsg, error) {
	logrus.Warnf("Will auth: %#v", req)

	str, err := auth.Login(req.User, req.Pass)

	return &proxy.StringMsg{Msg: str}, err
}

func (s *Service) Logout(ctx context.Context, req *proxy.StringMsg) (*proxy.Noop, error) {
	err := auth.Logout(req.Msg)
	return &proxy.Noop{}, err
}

func (s *Service) GetConfigs(ctx context.Context, req *proxy.Noop) (*proxy.Bridges, error) {
	ret := &proxy.Bridges{Eps: s.bridges.KeyValues().Values()}
	return ret, nil
}

func (s *Service) StreamConfig(req *proxy.Noop, server proxy.Proxy_StreamConfigServer) error {
	brds := proxy.Bridges{
		Eps: s.bridges.KeyValues().Values(),
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

func (s *Service) KeepAlive(req *proxy.Noop, res proxy.Proxy_KeepAliveServer) error {
	for {
		res.Send(&proxy.StringMsg{Msg: "PING"})
		time.Sleep(time.Second)
	}
}
