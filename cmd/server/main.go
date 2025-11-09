package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	blog "github.com/yourEmotion/Blog_gRPC/api/go"
	"github.com/yourEmotion/Blog_gRPC/internal/config"
	"github.com/yourEmotion/Blog_gRPC/internal/models"
	"github.com/yourEmotion/Blog_gRPC/internal/service"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

var (
	httpPort = flag.Int("port", 8080, "HTTP server port")
	grpcPort = flag.Int("grpc-port", 50051, "gRPC server port")
)

func main() {
	flag.Parse()

	db, err := config.InitPostgres()
	if err != nil {
		log.Fatalf("failed to connect to Postgres: %v", err)
	}
	log.Println("Connected to Postgres!")

	if err := db.AutoMigrate(&models.Post{}); err != nil {
		log.Fatalf("failed to migrate Postgres: %v", err)
	}

	redisClient, err := config.InitRedis(context.Background())
	if err != nil {
		log.Fatalf("failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis!")

	grpcSrv := grpc.NewServer()
	blog.RegisterBlogServiceServer(grpcSrv, service.NewBlogService(db, redisClient))

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *grpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go func() {
		log.Printf("gRPC server listening on :%d", *grpcPort)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(func(header string) (string, bool) {
			if header == "User-Id" {
				return "User-Id", true
			}
			return runtime.DefaultHeaderMatcher(header)
		}),
	)

	opts := []grpc.DialOption{grpc.WithInsecure()}
	if err := blog.RegisterBlogServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", *grpcPort), opts); err != nil {
		log.Fatalf("failed to start REST gateway: %v", err)
	}

	httpMux := http.NewServeMux()
	httpMux.Handle("/", mux)
	httpMux.Handle("/swagger/", http.StripPrefix("/swagger/", http.FileServer(http.Dir("api/swagger"))))
	httpMux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			http.ServeFile(w, r, "api/swagger/blog.swagger.json")
		}
	})

	httpSrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", *httpPort),
		Handler: httpMux,
	}

	go func() {
		log.Printf("REST gateway listening on :%d", *httpPort)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to serve HTTP: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("Shutting down...")
	grpcSrv.GracefulStop()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Fatalf("HTTP server shutdown failed: %v", err)
	}
	log.Println("Servers stopped")
}
