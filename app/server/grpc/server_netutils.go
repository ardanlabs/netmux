package grpc

import (
	"context"

	"github.com/ardanlabs.com/netmux/business/grpc/clients/proxy"
	"github.com/ardanlabs.com/netmux/foundation/shell"
	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
)

func (*Server) Ping(_ context.Context, req *proxy.StringMsg) (*proxy.StringMsg, error) {
	ret, err := shell.Ping(req.Msg)
	if err != nil {
		return nil, err
	}

	return &proxy.StringMsg{Msg: ret}, nil
}

func (*Server) PortScan(_ context.Context, req *proxy.StringMsg) (*proxy.StringMsg, error) {
	ret, err := shell.Nmap(req.Msg)
	if err != nil {
		return nil, err
	}

	return &proxy.StringMsg{Msg: ret}, nil
}

func (*Server) Nc(_ context.Context, req *proxy.StringMsg) (*proxy.StringMsg, error) {
	ret, err := shell.Nc(req.Msg) //cmd.Nc(req.Msg)
	if err != nil {
		return nil, err
	}

	return &proxy.StringMsg{Msg: ret}, nil
}

func (*Server) SpeedTest(ctx context.Context, req *proxy.StringMsg) (*proxy.BytesMsg, error) {
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
	return &proxy.BytesMsg{Msg: pl}, nil
}
