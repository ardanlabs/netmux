// Package service provides support for running a grpc proxy service.
package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ardanlabs.com/netmux/app/server/auth"
	"github.com/ardanlabs.com/netmux/business/grpc/clients/proxy"
	"github.com/ardanlabs.com/netmux/foundation/db"
	"github.com/ardanlabs.com/netmux/foundation/shell"
	"github.com/ardanlabs.com/netmux/foundation/signal"
	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// Service represents a grpc proxy service.
type Service struct {
	proxy.UnsafeProxyServer
	log                *logrus.Logger
	auth               *auth.Auth
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
func Start(log *logrus.Logger, auth *auth.Auth) (*Service, error) {
	srv := Service{
		log:  log,
		auth: auth,
		grpc: grpc.NewServer(
			grpc.UnaryInterceptor(authUnaryServerInterceptor(auth)),
			grpc.StreamInterceptor(authStreamServerInterceptor(auth)),
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

// AddBridge adds the specified proxy bridge to the set of known bridges.
func (s *Service) AddBridge(brd *proxy.Bridge) error {
	brd.Bridgeop = "A"

	existing, err := s.bridges.Get(brd.Name)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			s.bridges.Set(brd.Name, brd)
			s.propagate(brd)
			return nil
		}

		return fmt.Errorf("s.bridges.Get: %w", err)
	}

	if existing.K8Snamespace == brd.K8Snamespace && existing.K8Sname == brd.K8Sname && existing.K8Skind == brd.K8Skind {
		s.bridges.Set(brd.Name, brd)
		s.propagate(brd)
	}

	return nil
}

// RemoveBridge removes the specified proxy bridge from the set of known bridges.
func (s *Service) RemoveBridge(brd *proxy.Bridge) error {
	brd.Bridgeop = "D"

	if _, err := s.bridges.Get(brd.Name); err != nil {
		return fmt.Errorf("s.bridges.Get: %w", err)
	}

	s.bridges.Delete(brd.Name)
	s.propagate(brd)

	return nil
}

// =============================================================================

// Ver is provided to implement the ProxyServer interface.
func (s *Service) Ver(ctx context.Context, nop *proxy.Noop) (*proxy.Noop, error) {
	return &proxy.Noop{}, nil
}

// Ping is provided to implement the ProxyServer interface.
func (s *Service) Ping(ctx context.Context, msg *proxy.StringMsg) (*proxy.StringMsg, error) {
	resp, err := shell.Ping(msg.Msg)
	if err != nil {
		return nil, fmt.Errorf("shell.Ping: %w", err)
	}

	return &proxy.StringMsg{Msg: resp}, nil
}

// PortScan is provided to implement the ProxyServer interface.
func (s *Service) PortScan(ctx context.Context, msg *proxy.StringMsg) (*proxy.StringMsg, error) {
	resp, err := shell.Nmap(msg.Msg)
	if err != nil {
		return nil, fmt.Errorf("shell.Nmap: %w", err)
	}

	return &proxy.StringMsg{Msg: resp}, nil
}

// Nc is provided to implement the ProxyServer interface.
func (s *Service) Nc(ctx context.Context, msg *proxy.StringMsg) (*proxy.StringMsg, error) {
	resp, err := shell.Netcat(msg.Msg) //cmd.Netcat(req.Msg)
	if err != nil {
		return nil, fmt.Errorf("shell.Netcat: %w", err)
	}

	return &proxy.StringMsg{Msg: resp}, nil
}

// SpeedTest is provided to implement the ProxyServer interface.
func (s *Service) SpeedTest(ctx context.Context, msg *proxy.StringMsg) (*proxy.BytesMsg, error) {
	numOfBytes, err := humanize.ParseBytes(msg.Msg)
	if err != nil {
		return nil, fmt.Errorf("humanize.ParseBytes: %w", err)
	}

	s.log.Infof("server: speedtest: generating the payload: %s: %d", msg.Msg, numOfBytes)
	pl := make([]byte, int(numOfBytes))

	return &proxy.BytesMsg{Msg: pl}, nil
}

// KeepAlive is provided to implement the ProxyServer interface.
func (s *Service) KeepAlive(nop *proxy.Noop, keepAliveServer proxy.Proxy_KeepAliveServer) error {
	// TODO: Review this code.

	for {
		keepAliveServer.Send(&proxy.StringMsg{Msg: "PING"})
		time.Sleep(time.Second)
	}
}

// GetConfigs is provided to implement the ProxyServer interface.
func (s *Service) GetConfigs(ctx context.Context, nop *proxy.Noop) (*proxy.Bridges, error) {
	brds := proxy.Bridges{
		Eps: s.bridges.KeyValues().Values(),
	}

	return &brds, nil
}

// StreamConfig is provided to implement the ProxyServer interface.
func (s *Service) StreamConfig(nop *proxy.Noop, streamConfigServer proxy.Proxy_StreamConfigServer) error {
	brds := proxy.Bridges{
		Eps: s.bridges.KeyValues().Values(),
	}

	if err := streamConfigServer.Send(&brds); err != nil {
		s.log.Infof("StreamConfig: streamConfigServer.Send: ERROR: %s", err)
		return fmt.Errorf("streamConfigServer.Send: %w", err)
	}

	ch := s.signal.Aquire()
	s.log.Info("added cfg listener for local agent")

	for eps := range ch {
		s.log.Info("got cfg")

		brds := proxy.Bridges{
			Eps: []*proxy.Bridge{eps},
		}

		if err := streamConfigServer.Send(&brds); err != nil {
			return err
		}
	}

	return nil
}

// Login is provided to implement the ProxyServer interface.
func (s *Service) Login(ctx context.Context, loginReq *proxy.LoginReq) (*proxy.StringMsg, error) {
	s.log.Infof("login: %s", loginReq.User)

	userID, err := s.auth.Login(loginReq.User, loginReq.Pass)
	if err != nil {
		return nil, fmt.Errorf("auth.Login: %w", err)
	}

	return &proxy.StringMsg{Msg: userID}, err
}

// Logout is provided to implement the ProxyServer interface.
func (s *Service) Logout(ctx context.Context, msg *proxy.StringMsg) (*proxy.Noop, error) {
	if err := s.auth.Logout(msg.Msg); err != nil {
		return nil, fmt.Errorf("auth.Logout: %w", err)
	}

	return &proxy.Noop{}, nil
}

// =============================================================================

// propagate broadcasts the brd to the G's that are listening.
func (s *Service) propagate(brd *proxy.Bridge) error {
	s.log.Infof("service: propagate: Name[%s] Bridgeop[%s]", brd.Name, brd.Bridgeop)

	f := func(k string, brd *proxy.Bridge) error {
		s.log.Infof("%s => %s", k, brd)
		return nil
	}
	if err := s.bridges.ForEach(f); err != nil {
		return fmt.Errorf("s.bridges.ForEach: %w", err)
	}

	s.signal.Broadcast(brd)

	return nil
}
