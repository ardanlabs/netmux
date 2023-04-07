package proxy

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	KeyUser  = "user"
	KeyPass  = "password"
	KeyToken = "token"
)

func UnaryClientInterceptor(tk string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx = metadata.AppendToOutgoingContext(ctx, KeyToken, tk)
		err := invoker(ctx, method, req, reply, cc, opts...)
		return err
	}
}

// StreamClientInterceptor allows us to log on each client stream opening
func StreamClientInterceptor(tk string) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		ctx = metadata.AppendToOutgoingContext(ctx, KeyToken, tk)
		return streamer(ctx, desc, cc, method)
	}
}
