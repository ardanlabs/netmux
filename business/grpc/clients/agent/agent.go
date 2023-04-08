package agent

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewClient constructs an AgentClient value.
func NewClient(user string, password string) (AgentClient, error) {
	dialer := func(ctx context.Context, addr string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, "unix", addr)
	}

	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithContextDialer(dialer),
		grpc.WithUnaryInterceptor(UnaryClientInterceptor("")),
		grpc.WithStreamInterceptor(StreamClientInterceptor("")),
	}

	conn, err := grpc.Dial("/tmp/netmux.sock", options...)
	if err != nil {
		return nil, err
	}

	return NewAgentClient(conn), nil
}
