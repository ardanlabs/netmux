package cluster

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewClient constructs a ProxyClient value.
func NewClient(target string, tk string) (ClusterClient, error) {
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

	return NewClusterClient(conn), nil
}
