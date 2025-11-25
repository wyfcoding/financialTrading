package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/wyfcoding/financialTrading/go-api/pricing"
	"github.com/wyfcoding/financialTrading/internal/pricing/application"
	"github.com/wyfcoding/financialTrading/internal/pricing/infrastructure"
	"github.com/wyfcoding/financialTrading/internal/pricing/interfaces"
	"github.com/wyfcoding/financialTrading/pkg/config"
	"github.com/wyfcoding/financialTrading/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 加载配置
	configPath := "configs/pricing/config.toml"
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. 初始化日志
	loggerCfg := logger.Config{ // Detailed logger initialization
		Level:      cfg.Logger.Level,
		Format:     cfg.Logger.Format,
		Output:     cfg.Logger.Output,
		FilePath:   cfg.Logger.FilePath,
		MaxSize:    cfg.Logger.MaxSize,
		MaxBackups: cfg.Logger.MaxBackups,
		MaxAge:     cfg.Logger.MaxAge,
		Compress:   cfg.Logger.Compress,
		WithCaller: cfg.Logger.WithCaller,
	}
	if err := logger.Init(loggerCfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	logger.Info(ctx, "Starting PricingService", "version", cfg.Version) // Access cfg.Version

	// 3. 初始化层级依赖
	// Infrastructure
	// PricingService 主要是计算服务，可能不需要数据库，或者只需要读取市场数据
	marketDataClient := infrastructure.NewMockMarketDataClient()

	// Application
	svc := application.NewPricingService(marketDataClient) // Renamed pricingApp to svc

	// Interfaces
	handler := interfaces.NewGRPCHandler(svc) // Renamed grpcHandler to handler

	// 4. 启动 gRPC 服务
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPC.Port)) // Access cfg.GRPC.Port
	if err != nil {
		logger.Fatal(ctx, "Failed to listen", "error", err) // Changed message
	}

	s := grpc.NewServer()
	pb.RegisterPricingServiceServer(s, handler) // Use new handler name
	reflection.Register(s)

	// 启动 gRPC 服务
	go func() {
		logger.Info(ctx, "Starting gRPC server", "port", cfg.GRPC.Port)
		if err := s.Serve(lis); err != nil {
			logger.Fatal(ctx, "Failed to serve", "error", err)
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info(ctx, "Shutting down server...")

	s.GracefulStop()
	logger.Info(ctx, "Server exited")
}
