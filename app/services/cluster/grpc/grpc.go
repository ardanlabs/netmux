// Package grpc provides support for running a grpc proxy service.
package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ardanlabs.com/netmux/app/services/cluster/auth"
	"github.com/ardanlabs.com/netmux/business/grpc/cluster"
	"github.com/ardanlabs.com/netmux/foundation/db"
	"github.com/ardanlabs.com/netmux/foundation/shell"
	"github.com/ardanlabs.com/netmux/foundation/signal"
	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// GRPC implements the grpc interface.
type GRPC struct {
	cluster.UnsafeProxyServer
	log                *logrus.Logger
	auth               *auth.Auth
	server             *grpc.Server
	signal             *signal.Signal[*cluster.Bridge]
	bridges            *db.DB[*cluster.Bridge]
	sessions           *db.DB[string]
	conns              *db.DB[net.Conn]
	wg                 sync.WaitGroup
	mu                 sync.Mutex
	reverseProxyLister net.Listener
}

// Start starts the grpc server.
func Start(log *logrus.Logger, auth *auth.Auth) (*GRPC, error) {
	g := GRPC{
		log:  log,
		auth: auth,
		server: grpc.NewServer(
			grpc.UnaryInterceptor(authUnaryServerInterceptor(auth)),
			grpc.StreamInterceptor(authStreamServerInterceptor(auth)),
		),
		signal:   signal.New[*cluster.Bridge](),
		bridges:  db.New[*cluster.Bridge](db.NopReadWriter{}),
		sessions: db.New[string](db.NopReadWriter{}),
		conns:    db.New[net.Conn](db.NopReadWriter{}),
	}

	l, err := net.Listen("tcp", "0.0.0.0:48080")
	if err != nil {
		return nil, fmt.Errorf("net.Listen: %w", err)
	}

	cluster.RegisterProxyServer(g.server, &g)

	g.wg.Add(1)

	go func() {
		log.Info("proxy: started")
		defer func() {
			log.Info("proxy: shutdown")
			g.wg.Done()
		}()

		g.log.Infof("proxy: running server at 0.0.0.0:48080")
		g.server.Serve(l)
	}()

	return &g, nil
}

// Shutdown requests the proxy service to stop and waits.
func (g *GRPC) Shutdown() {
	g.log.Infof("proxy: starting shutdown")
	defer g.log.Infof("proxy: shutdown")

	g.shutdownReverseProxyListen()
	g.server.GracefulStop()
}

// AddBridge adds the specified proxy bridge to the set of known bridges.
func (g *GRPC) AddBridge(brd *cluster.Bridge) error {
	brd.Bridgeop = "A"

	existing, err := g.bridges.Get(brd.Name)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			g.bridges.Set(brd.Name, brd)
			g.propagate(brd)
			return nil
		}

		return fmt.Errorf("s.bridges.Get: %w", err)
	}

	if existing.K8Snamespace == brd.K8Snamespace && existing.K8Sname == brd.K8Sname && existing.K8Skind == brd.K8Skind {
		g.bridges.Set(brd.Name, brd)
		g.propagate(brd)
	}

	return nil
}

// RemoveBridge removes the specified proxy bridge from the set of known bridges.
func (s *GRPC) RemoveBridge(brd *cluster.Bridge) error {
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
func (s *GRPC) Ver(ctx context.Context, nop *cluster.Noop) (*cluster.Noop, error) {
	return &cluster.Noop{}, nil
}

// Ping is provided to implement the ProxyServer interface.
func (s *GRPC) Ping(ctx context.Context, msg *cluster.StringMsg) (*cluster.StringMsg, error) {
	resp, err := shell.Ping(msg.Msg)
	if err != nil {
		return nil, fmt.Errorf("shell.Ping: %w", err)
	}

	return &cluster.StringMsg{Msg: resp}, nil
}

// PortScan is provided to implement the ProxyServer interface.
func (s *GRPC) PortScan(ctx context.Context, msg *cluster.StringMsg) (*cluster.StringMsg, error) {
	resp, err := shell.Nmap(msg.Msg)
	if err != nil {
		return nil, fmt.Errorf("shell.Nmap: %w", err)
	}

	return &cluster.StringMsg{Msg: resp}, nil
}

// Nc is provided to implement the ProxyServer interface.
func (s *GRPC) Nc(ctx context.Context, msg *cluster.StringMsg) (*cluster.StringMsg, error) {
	resp, err := shell.Netcat(msg.Msg) //cmd.Netcat(req.Msg)
	if err != nil {
		return nil, fmt.Errorf("shell.Netcat: %w", err)
	}

	return &cluster.StringMsg{Msg: resp}, nil
}

// SpeedTest is provided to implement the ProxyServer interface.
func (s *GRPC) SpeedTest(ctx context.Context, msg *cluster.StringMsg) (*cluster.BytesMsg, error) {
	numOfBytes, err := humanize.ParseBytes(msg.Msg)
	if err != nil {
		return nil, fmt.Errorf("humanize.ParseBytes: %w", err)
	}

	s.log.Infof("server: speedtest: generating the payload: %s: %d", msg.Msg, numOfBytes)
	pl := make([]byte, int(numOfBytes))

	return &cluster.BytesMsg{Msg: pl}, nil
}

// KeepAlive is provided to implement the ProxyServer interface.
func (s *GRPC) KeepAlive(nop *cluster.Noop, keepAliveServer cluster.Cluster_KeepAliveServer) error {
	// TODO: Review this code.

	for {
		keepAliveServer.Send(&cluster.StringMsg{Msg: "PING"})
		time.Sleep(time.Second)
	}
}

// GetConfigs is provided to implement the ProxyServer interface.
func (s *GRPC) GetConfigs(ctx context.Context, nop *cluster.Noop) (*cluster.Bridges, error) {
	brds := cluster.Bridges{
		Eps: s.bridges.KeyValues().Values(),
	}

	return &brds, nil
}

// StreamConfig is provided to implement the ProxyServer interface.
func (s *GRPC) StreamConfig(nop *cluster.Noop, streamConfigServer cluster.Cluster_StreamConfigServer) error {
	brds := cluster.Bridges{
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

		brds := cluster.Bridges{
			Eps: []*cluster.Bridge{eps},
		}

		if err := streamConfigServer.Send(&brds); err != nil {
			return err
		}
	}

	return nil
}

// Login is provided to implement the ProxyServer interface.
func (s *GRPC) Login(ctx context.Context, loginReq *cluster.LoginReq) (*cluster.StringMsg, error) {
	s.log.Infof("login: %s", loginReq.User)

	userID, err := s.auth.Login(loginReq.User, loginReq.Pass)
	if err != nil {
		return nil, fmt.Errorf("auth.Login: %w", err)
	}

	return &cluster.StringMsg{Msg: userID}, err
}

// Logout is provided to implement the ProxyServer interface.
func (s *GRPC) Logout(ctx context.Context, msg *cluster.StringMsg) (*cluster.Noop, error) {
	if err := s.auth.Logout(msg.Msg); err != nil {
		return nil, fmt.Errorf("auth.Logout: %w", err)
	}

	return &cluster.Noop{}, nil
}

// =============================================================================

// propagate broadcasts the brd to the G's that are listening.
func (s *GRPC) propagate(brd *cluster.Bridge) error {
	s.log.Infof("service: propagate: Name[%s] Bridgeop[%s]", brd.Name, brd.Bridgeop)

	f := func(k string, brd *cluster.Bridge) error {
		s.log.Infof("%s => %s", k, brd)
		return nil
	}
	if err := s.bridges.ForEach(f); err != nil {
		return fmt.Errorf("s.bridges.ForEach: %w", err)
	}

	s.signal.Broadcast(brd)

	return nil
}
