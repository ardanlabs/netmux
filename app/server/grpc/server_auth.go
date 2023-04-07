package grpc

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"
	"go.digitalcircle.com.br/dc/netmux/app/server/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func doAuth(m string, md metadata.MD) error {
	if m == "/proxy.NXProxy/Login" {
		return nil
	}
	str, err := auth.Check(md.Get("token")[0])
	if err != nil {
		return err
	}
	logrus.Debugf("Got at auth %s:%#v - %s", m, md, str)
	return nil
}

func authUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		method := info.FullMethod
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, errors.New("no metadata available")
		}
		err := doAuth(method, md)
		if err != nil {
			return nil, err
		}

		return handler(ctx, req)

	}
}

func authStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		md, ok := metadata.FromIncomingContext(ss.Context())
		method := info.FullMethod
		if !ok {
			return errors.New("no metadata available")
		}
		err := doAuth(method, md)
		if err != nil {
			return err
		}
		return handler(srv, ss)
	}
}
