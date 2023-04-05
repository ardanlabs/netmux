package grpc

import (
	"context"
	"net"
	"time"

	"github.com/sirupsen/logrus"
	"go.digitalcircle.com.br/dc/netmux/cmd/server/auth"
	"go.digitalcircle.com.br/dc/netmux/foundation/db"
	"go.digitalcircle.com.br/dc/netmux/lib/chmux"
	pb "go.digitalcircle.com.br/dc/netmux/lib/proto/server"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnsafeNXProxyServer
	eps      *db.DB[*pb.Bridge]
	sessions *db.DB[string]
	conns    *db.DB[net.Conn]
	chmux    *chmux.ChMux[[]*pb.Bridge]
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
	ret := &pb.Bridges{Eps: s.eps.KeyValues().Values()}
	err := server.Send(ret)
	if err != nil {
		logrus.Warnf("Error sending initial cfg: %s", err.Error())
		return err
	}
	ch := s.chmux.New()

	logrus.Tracef("added cfg listener for local agent")
	go func() {
		<-server.Context().Done()
		logrus.Tracef("closing get config due to ctx cancellation")
		if ch != nil {
			s.chmux.Close(ch)
			ch = nil
		}
	}()

	defer func() {
		logrus.Tracef("closing cfg listener for local agent")
		if ch != nil {
			s.chmux.Close(ch)
		}
	}()

	for {
		logrus.Tracef("awaiting cfg")
		b, ok := <-ch
		logrus.Tracef("got cfg")
		if !ok {
			return nil
		}
		ret := &pb.Bridges{Eps: b}
		err = server.Send(ret)
		if err != nil {
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
	chmux:    chmux.New[[]*pb.Bridge](),
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
