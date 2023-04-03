package grpc

import (
	"context"

	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
	"go.digitalcircle.com.br/dc/netmux/foundation/shell"
	pb "go.digitalcircle.com.br/dc/netmux/lib/proto/server"
)

func (server) Ping(_ context.Context, req *pb.StringMsg) (*pb.StringMsg, error) {
	ret, err := shell.Ping(req.Msg)
	if err != nil {
		return nil, err
	}

	return &pb.StringMsg{Msg: ret}, nil
}

func (server) PortScan(_ context.Context, req *pb.StringMsg) (*pb.StringMsg, error) {
	ret, err := shell.Nmap(req.Msg)
	if err != nil {
		return nil, err
	}

	return &pb.StringMsg{Msg: ret}, nil
}

func (server) Nc(_ context.Context, req *pb.StringMsg) (*pb.StringMsg, error) {
	ret, err := shell.Nc(req.Msg) //cmd.Nc(req.Msg)
	if err != nil {
		return nil, err
	}

	return &pb.StringMsg{Msg: ret}, nil
}

func (server) SpeedTest(ctx context.Context, req *pb.StringMsg) (*pb.BytesMsg, error) {
	sz, err := humanize.ParseBytes(req.Msg)
	if err != nil {
		return nil, err
	}
	logrus.Infof("Generating a payload of: %s: %v", req.Msg, sz)
	szint := int(sz)
	pl := make([]byte, szint)
	for i := 0; i < len(pl); i++ {
		pl[i] = []byte("x")[0]
	}
	return &pb.BytesMsg{Msg: pl}, nil
}
