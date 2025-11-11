package middleware

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func UnaryLoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		start := time.Now()
		zap.L().Info("gRPC request started",
			zap.String("method", info.FullMethod),
		)

		resp, err := handler(ctx, req)

		zap.L().Info("gRPC request finished",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
		)

		return resp, err
	}
}
