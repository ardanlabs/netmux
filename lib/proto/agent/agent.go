package agent

import (
	"context"
	"go.digitalcircle.com.br/dc/netmux/lib/proto/interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net"
)

func NewUnixDefault() (AgentClient, error) {

	var (
		credentials = insecure.NewCredentials() // No SSL/TLS
		dialer      = func(ctx context.Context, addr string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", addr)
		}
		options = []grpc.DialOption{
			grpc.WithTransportCredentials(credentials),
			grpc.WithBlock(),
			grpc.WithContextDialer(dialer),
			grpc.WithUnaryInterceptor(interceptors.UnaryClientInterceptor("")),
			grpc.WithStreamInterceptor(interceptors.StreamClientInterceptor("")),
		}
	)

	conn, err := grpc.Dial("/tmp/netmux.sock", options...)
	if err != nil {
		return nil, err
	}
	cli := NewAgentClient(conn)

	return cli, nil
}
