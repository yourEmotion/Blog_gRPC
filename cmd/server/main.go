package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	blog "github.com/yourEmotion/Blog_gRPC/api/go"
	"github.com/yourEmotion/Blog_gRPC/internal/config"
	"github.com/yourEmotion/Blog_gRPC/internal/middleware"
	"github.com/yourEmotion/Blog_gRPC/internal/models"
	"github.com/yourEmotion/Blog_gRPC/internal/service"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	httpPort = flag.Int("port", 8080, "HTTP server port")
	grpcPort = flag.Int("grpc-port", 50051, "gRPC server port")
)

func main() {
	flag.Parse()
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

	zap.L().Info("Starting Blog gRPC service")

	db, err := config.InitPostgres()
	if err != nil {
		zap.L().Fatal("failed to connect to Postgres", zap.Error(err))
	}
	zap.L().Info("Connected to Postgres!")

	if err := db.AutoMigrate(&models.Post{}); err != nil {
		zap.L().Fatal("failed to migrate Postgres", zap.Error(err))
	}

	redisClient, err := config.InitRedis(context.Background())
	if err != nil {
		zap.L().Fatal("failed to connect to Redis", zap.Error(err))
	}
	zap.L().Info("Connected to Redis!")

	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.UnaryLoggingInterceptor(),
			grpc_prometheus.UnaryServerInterceptor,
		),
	)
	grpc_prometheus.Register(grpcSrv)

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		zap.L().Info("Prometheus metrics listening on :2112/metrics")
		if err := http.ListenAndServe(":2112", nil); err != nil {
			zap.L().Fatal("failed to serve metrics", zap.Error(err))
		}
	}()

	blog.RegisterBlogServiceServer(grpcSrv, service.NewBlogService(db, redisClient))

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *grpcPort))
	if err != nil {
		zap.L().Fatal("failed to listen", zap.Error(err))
	}

	go func() {
		zap.L().Info("gRPC server started", zap.Int("port", *grpcPort))
		if err := grpcSrv.Serve(lis); err != nil {
			zap.L().Fatal("failed to serve gRPC", zap.Error(err))
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
		zap.L().Fatal("failed to start REST gateway", zap.Error(err))
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
		zap.L().Info("REST gateway started", zap.Int("port", *httpPort))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Fatal("failed to serve HTTP", zap.Error(err))
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	zap.L().Info("Shutting down...")
	grpcSrv.GracefulStop()
	if err := httpSrv.Shutdown(ctx); err != nil {
		zap.L().Fatal("HTTP server shutdown failed", zap.Error(err))
	}
	zap.L().Info("Servers stopped gracefully")
}
