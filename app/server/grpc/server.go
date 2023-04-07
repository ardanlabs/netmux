package grpc

import (
	"context"
	"net"
	"time"

	"github.com/sirupsen/logrus"
	"go.digitalcircle.com.br/dc/netmux/app/server/auth"
	"go.digitalcircle.com.br/dc/netmux/foundation/db"
	"go.digitalcircle.com.br/dc/netmux/foundation/signal"
	pb "go.digitalcircle.com.br/dc/netmux/lib/proto/server"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnsafeNXProxyServer
	eps      *db.DB[*pb.Bridge]
	sessions *db.DB[string]
	conns    *db.DB[net.Conn]
	signal   *signal.Signal[*pb.Bridge]
}

func (s server) mustEmbedUnimplementedServerServiceServer() {

}

func (s server) Ver(_ context.Context, _ *pb.Noop) (*pb.Noop, error) {
	return &pb.Noop{}, nil
}

func (s server) Login(ctx context.Context, req *pb.LoginReq) (*pb.StringMsg, error) {
	logrus.Warnf("Will auth: %#v", req)

	str, err := auth.Login(req.User, req.Pass)

	return &pb.StringMsg{Msg: str}, err
}

func (s server) Logout(ctx context.Context, req *pb.StringMsg) (*pb.Noop, error) {
	err := auth.Logout(req.Msg)
	return &pb.Noop{}, err
}

func (s server) GetConfigs(ctx context.Context, req *pb.Noop) (*pb.Bridges, error) {
	ret := &pb.Bridges{Eps: s.eps.KeyValues().Values()}
	return ret, nil
}

func (s server) StreamConfig(req *pb.Noop, server pb.NXProxy_StreamConfigServer) error {
	brds := pb.Bridges{
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

		brds := pb.Bridges{
			Eps: []*pb.Bridge{eps},
		}

		if err := server.Send(&brds); err != nil {
			return err
		}
	}
}

func (s server) KeepAlive(req *pb.Noop, res pb.NXProxy_KeepAliveServer) error {
	for {
		res.Send(&pb.StringMsg{Msg: "PING"})
		time.Sleep(time.Second)
	}
}

var aServer = server{
	eps:      db.New[*pb.Bridge](db.NopReadWriter{}),
	sessions: db.New[string](db.NopReadWriter{}),
	conns:    db.New[net.Conn](db.NopReadWriter{}),
	signal:   signal.New[*pb.Bridge](),
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
	pb.RegisterNXProxyServer(grpcServer, Server())
	return grpcServer.Serve(l)
}
