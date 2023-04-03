package grpc

import (
	"context"
	"github.com/sirupsen/logrus"
	"go.digitalcircle.com.br/dc/netmux/cmd/server/auth"
	"go.digitalcircle.com.br/dc/netmux/lib/chmux"
	"go.digitalcircle.com.br/dc/netmux/lib/config"
	"go.digitalcircle.com.br/dc/netmux/lib/memdb"
	pb "go.digitalcircle.com.br/dc/netmux/lib/proto/server"
	"google.golang.org/grpc"
	"net"
	"time"
)

type ServerImpl struct {
	pb.UnsafeNXProxyServer
	eps      *memdb.Memdb[*pb.Bridge]
	sessions *memdb.Memdb[string]
	conns    *memdb.Memdb[net.Conn]

	chmux *chmux.ChMux[[]*pb.Bridge]
}

func (s *ServerImpl) mustEmbedUnimplementedServerServiceServer() {

}

func (s *ServerImpl) Ver(_ context.Context, _ *pb.Noop) (*pb.Noop, error) {
	return &pb.Noop{}, nil
}

func (s *ServerImpl) Login(_ context.Context, req *pb.LoginReq) (*pb.StringMsg, error) {
	logrus.Warnf("Will auth: %#v", req)

	str, err := auth.Login(req.User, req.Pass)

	return &pb.StringMsg{Msg: str}, err
}

func (s *ServerImpl) Logout(_ context.Context, req *pb.StringMsg) (*pb.Noop, error) {
	err := auth.Logout(req.Msg)
	return &pb.Noop{}, err
}

func (s *ServerImpl) GetConfigs(_ context.Context, _ *pb.Noop) (*pb.Bridges, error) {
	ret := &pb.Bridges{Eps: s.eps.Items()}
	return ret, nil
}

func (s *ServerImpl) StreamConfig(_ *pb.Noop, server pb.NXProxy_StreamConfigServer) error {
	ret := &pb.Bridges{Eps: s.eps.Items()}
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

func (s *ServerImpl) KeepAlive(_ *pb.Noop, res pb.NXProxy_KeepAliveServer) error {
	for {
		err := res.Send(&pb.StringMsg{Msg: "PING"})
		time.Sleep(time.Second)
		if err != nil {
			return err
		}
	}
}

var aServer = ServerImpl{
	eps:      memdb.New[*pb.Bridge](),
	sessions: memdb.New[string](),
	conns:    memdb.New[net.Conn](),
	chmux:    chmux.New[[]*pb.Bridge](),
}

func Server() *ServerImpl {
	return &aServer
}

func Run() error {
	logrus.Infof("Running ServerImpl at 0.0.0.0:48080")
	l, err := net.Listen("tcp", "0.0.0.0:48080")
	if err != nil {
		return err
	}
	err = config.Default().Load()
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(authUnaryServerInterceptor()),
		grpc.StreamInterceptor(authStreamServerInterceptor()),
	)
	pb.RegisterNXProxyServer(grpcServer, serverImpl())
	return grpcServer.Serve(l)
}
