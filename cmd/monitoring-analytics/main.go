package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/wyfcoding/financialTrading/go-api/monitoring-analytics"
	"github.com/wyfcoding/financialTrading/internal/monitoring-analytics/application"
	"github.com/wyfcoding/financialTrading/internal/monitoring-analytics/infrastructure"
	"github.com/wyfcoding/financialTrading/internal/monitoring-analytics/interfaces"
	"github.com/wyfcoding/financialTrading/pkg/config"
	"github.com/wyfcoding/financialTrading/pkg/db"
	"github.com/wyfcoding/financialTrading/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 1. 加载配置
	configPath := "configs/monitoring-analytics/config.toml"
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志
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
	logger.Info(ctx, "Starting MonitoringAnalyticsService", "version", cfg.Version)

	// 初始化数据库
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

	// 自动迁移
	if err := gormDB.AutoMigrate(&infrastructure.MetricModel{}, &infrastructure.SystemHealthModel{}); err != nil {
		logger.Fatal(ctx, "Failed to migrate database", "error", err)
	}

	// 初始化依赖
	metricRepo := infrastructure.NewMetricRepository(gormDB.DB)
	healthRepo := infrastructure.NewSystemHealthRepository(gormDB.DB)
	svc := application.NewMonitoringAnalyticsService(metricRepo, healthRepo)
	handler := interfaces.NewGRPCHandler(svc)

	// 启动 gRPC 服务
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPC.Port))
	if err != nil {
		logger.Fatal(ctx, "Failed to listen", "error", err)
	}

	s := grpc.NewServer()
	pb.RegisterMonitoringAnalyticsServiceServer(s, handler)
	// 注册反射服务
	reflection.Register(s)

	// 启动 gRPC 服务
	go func() {
		logger.Info(ctx, "Starting gRPC server", "port", cfg.GRPC.Port)
		if err := s.Serve(lis); err != nil {
			logger.Fatal(ctx, "Failed to serve", "error", err)
		}
	}()

	// 等待中断信号以优雅地关闭服务器（设置 5 秒的超时时间）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info(ctx, "Shutting down server...")

	s.GracefulStop()
	// gormDB.Close() // gorm.DB doesn't have Close()
	logger.Info(ctx, "Server exited")
}
