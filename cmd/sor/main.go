package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/wyfcoding/financialTrading/internal/sor/application"
	"github.com/wyfcoding/financialTrading/internal/sor/domain"
	"github.com/wyfcoding/financialTrading/internal/sor/interfaces"
	pb "github.com/wyfcoding/financialtrading/go-api/sor/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 1. 初始化日志
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	logger.Info("starting SOR engine service")

	// 2. 依赖注入
	engine := &domain.DefaultSOREngine{}
	appService := application.NewSORApplicationService(engine, logger)
	handler := interfaces.NewSORHandler(appService)

	// 3. 启动 gRPC 服务
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50052"
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		logger.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterSOREngineServiceServer(grpcServer, handler)
	reflection.Register(grpcServer)

	// 4. 优雅关停
	go func() {
		logger.Info("gRPC server listening on", "port", port)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("failed to serve", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down SOR engine server...")
	grpcServer.GracefulStop()
	logger.Info("server exited")
}
