package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/wyfcoding/financialTrading/go-api/market-making"
	"github.com/wyfcoding/financialTrading/internal/market-making/application"
	"github.com/wyfcoding/financialTrading/internal/market-making/infrastructure"
	"github.com/wyfcoding/financialTrading/internal/market-making/interfaces"
	"github.com/wyfcoding/financialTrading/pkg/config"
	"github.com/wyfcoding/financialTrading/pkg/db"
	"github.com/wyfcoding/financialTrading/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 1. 加载配置
	configPath := "configs/market-making/config.toml"
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. 初始化日志
	loggerCfg := logger.Config{
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
	logger.Info(ctx, "Starting MarketMakingService", "version", cfg.Version)

	// 3. 初始化数据库
	dbConfig := db.Config{
		Driver:             cfg.Database.Driver,
		DSN:                cfg.Database.DSN,
		MaxOpenConns:       cfg.Database.MaxOpenConns,
		MaxIdleConns:       cfg.Database.MaxIdleConns,
		ConnMaxLifetime:    cfg.Database.ConnMaxLifetime,
		LogEnabled:         cfg.Database.LogEnabled,
		SlowQueryThreshold: cfg.Database.SlowQueryThreshold,
	}
	gormDB, err := db.Init(dbConfig)
	if err != nil {
		logger.Fatal(ctx, "Failed to connect to database", "error", err)
	}
	// defer gormDB.Close() // gorm.DB doesn't have Close()

	// 4. 自动迁移数据库
	if err := gormDB.AutoMigrate(&infrastructure.QuoteStrategyModel{}, &infrastructure.PerformanceModel{}); err != nil {
		logger.Fatal(ctx, "Failed to migrate database", "error", err)
	}

	// 5. 初始化层级依赖
	// Infrastructure
	strategyRepo := infrastructure.NewQuoteStrategyRepository(gormDB.DB)
	performanceRepo := infrastructure.NewPerformanceRepository(gormDB.DB)
	orderClient := infrastructure.NewMockOrderClient()
	marketDataClient := infrastructure.NewMockMarketDataClient()

	// Application
	marketMakingApp := application.NewMarketMakingService(strategyRepo, performanceRepo, orderClient, marketDataClient)

	// Interfaces
	grpcHandler := interfaces.NewGRPCHandler(marketMakingApp)

	// 6. 启动 gRPC 服务
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPC.Port))
	if err != nil {
		logger.Fatal(ctx, "Failed to listen", "error", err)
	}

	s := grpc.NewServer()
	pb.RegisterMarketMakingServiceServer(s, grpcHandler)
	reflection.Register(s)

	go func() {
		logger.Info(ctx, "gRPC server listening", "port", cfg.GRPC.Port)
		if err := s.Serve(lis); err != nil {
			logger.Fatal(ctx, "Failed to serve", "error", err)
		}
	}()

	// 7. 优雅关停
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(ctx, "Shutting down server...")
	s.GracefulStop()
	logger.Info(ctx, "Server exited")
}
