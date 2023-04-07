package grpc

import (
	"context"
	"net"
	"time"

	"github.com/ardanlabs.com/netmux/app/server/auth"
	"github.com/ardanlabs.com/netmux/business/grpc/clients/proxy"
	"github.com/ardanlabs.com/netmux/foundation/db"
	"github.com/ardanlabs.com/netmux/foundation/signal"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type server struct {
	proxy.UnsafeProxyServer
	eps      *db.DB[*proxy.Bridge]
	sessions *db.DB[string]
	conns    *db.DB[net.Conn]
	signal   *signal.Signal[*proxy.Bridge]
}

func (s server) mustEmbedUnimplementedServerServiceServer() {

}

func (s server) Ver(_ context.Context, _ *proxy.Noop) (*proxy.Noop, error) {
	return &proxy.Noop{}, nil
}

func (s server) Login(ctx context.Context, req *proxy.LoginReq) (*proxy.StringMsg, error) {
	logrus.Warnf("Will auth: %#v", req)

	str, err := auth.Login(req.User, req.Pass)

	return &proxy.StringMsg{Msg: str}, err
}

func (s server) Logout(ctx context.Context, req *proxy.StringMsg) (*proxy.Noop, error) {
	err := auth.Logout(req.Msg)
	return &proxy.Noop{}, err
}

func (s server) GetConfigs(ctx context.Context, req *proxy.Noop) (*proxy.Bridges, error) {
	ret := &proxy.Bridges{Eps: s.eps.KeyValues().Values()}
	return ret, nil
}

func (s server) StreamConfig(req *proxy.Noop, server proxy.Proxy_StreamConfigServer) error {
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

func (s server) KeepAlive(req *proxy.Noop, res proxy.Proxy_KeepAliveServer) error {
	for {
		res.Send(&proxy.StringMsg{Msg: "PING"})
		time.Sleep(time.Second)
	}
}

var aServer = server{
	eps:      db.New[*proxy.Bridge](db.NopReadWriter{}),
	sessions: db.New[string](db.NopReadWriter{}),
	conns:    db.New[net.Conn](db.NopReadWriter{}),
	signal:   signal.New[*proxy.Bridge](),
}

func Run() error {
	logrus.Infof("Running server at 0.0.0.0:48080")
	l, err := net.Listen("tcp", "0.0.0.0:48080")
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(authUnaryServerInterceptor()),
		grpc.StreamInterceptor(authStreamServerInterceptor()),
	)
	proxy.RegisterProxyServer(grpcServer, Server())
	return grpcServer.Serve(l)
}
