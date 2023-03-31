package server

import (
	"go.digitalcircle.com.br/dc/netmux/lib/proto/interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func New(addr string, tk string) (NXProxyClient, error) {

	var (
		credentials = insecure.NewCredentials() // No SSL/TLS
		options     = []grpc.DialOption{
			grpc.WithTransportCredentials(credentials),
			grpc.WithBlock(),
			grpc.WithUnaryInterceptor(interceptors.UnaryClientInterceptor(tk)),
			grpc.WithStreamInterceptor(interceptors.StreamClientInterceptor(tk)),
		}
	)

	conn, err := grpc.Dial(addr, options...)
	cli := NewNXProxyClient(conn)
	if err != nil {
		return nil, err
	}

	return cli, nil
}
