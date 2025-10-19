package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	blog "github.com/yourEmotion/Blog_gRPC/api/go"
	"google.golang.org/grpc"
)

func main() {
	grpcPort := ":50051"
	httpPort := ":8080"

	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	blog.RegisterBlogServiceServer(grpcServer, &BlogServer{})

	go func() {
		log.Printf("gRPC server listening on %s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}

	err = blog.RegisterBlogServiceHandlerFromEndpoint(ctx, mux, grpcPort, opts)
	if err != nil {
		log.Fatalf("failed to start REST gateway: %v", err)
	}

	cwd, _ := os.Getwd()
	swaggerPath := filepath.Join(cwd, "api", "swagger")

	http.Handle("/swagger/", http.StripPrefix("/swagger/", http.FileServer(http.Dir(swaggerPath))))
	http.Handle("/", mux)

	log.Printf("REST gateway listening on %s", httpPort)
	if err := http.ListenAndServe(httpPort, nil); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}
