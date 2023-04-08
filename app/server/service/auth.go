package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/ardanlabs.com/netmux/app/server/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func authUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	f := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, errors.New("no metadata available")
		}

		if err := doAuth(info.FullMethod, md); err != nil {
			return nil, fmt.Errorf("s.doAuth: %w", err)
		}

		return handler(ctx, req)
	}

	return f
}

func authStreamServerInterceptor() grpc.StreamServerInterceptor {
	f := func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return errors.New("no metadata available")
		}

		if err := doAuth(info.FullMethod, md); err != nil {
			return fmt.Errorf("s.doAuth: %w", err)
		}

		return handler(srv, ss)
	}

	return f
}

func doAuth(method string, md metadata.MD) error {
	if method == "/proxy.NXProxy/Login" {
		return nil
	}

	if _, err := auth.Check(md.Get("token")[0]); err != nil {
		return fmt.Errorf("auth.Check: %w", err)
	}

	return nil
}
