package proxy

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// New constructs a ProxyClient value.
func New(target string, tk string) (ProxyClient, error) {
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithUnaryInterceptor(UnaryClientInterceptor(tk)),
		grpc.WithStreamInterceptor(StreamClientInterceptor(tk)),
	}

	conn, err := grpc.Dial(target, options...)
	if err != nil {
		return nil, err
	}

	return NewProxyClient(conn), nil
}
