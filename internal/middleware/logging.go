package middleware

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

func UnaryLoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		start := time.Now()
		log.Printf("[gRPC START] method=%s", info.FullMethod)
		resp, err = handler(ctx, req)
		duration := time.Since(start)
		if err != nil {
			log.Printf("[gRPC ERROR] method=%s duration=%s err=%v", info.FullMethod, duration, err)
		} else {
			log.Printf("[gRPC END] method=%s duration=%s", info.FullMethod, duration)
		}
		return resp, err
	}
}
