package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/fynnwu/FinancialTrading/go-api/monitoring-analytics"
	"github.com/fynnwu/FinancialTrading/internal/monitoring-analytics/application"
	"github.com/fynnwu/FinancialTrading/internal/monitoring-analytics/infrastructure"
	"github.com/fynnwu/FinancialTrading/internal/monitoring-analytics/interfaces"
	"github.com/fynnwu/FinancialTrading/pkg/config"
	"github.com/fynnwu/FinancialTrading/pkg/db"
	"github.com/fynnwu/FinancialTrading/pkg/logger"
	"go.uber.org/zap"
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
	logger.Init(logger.Config{
		Level:      cfg.Logger.Level,
		Format:     cfg.Logger.Format,
		Output:     cfg.Logger.Output,
		FilePath:   cfg.Logger.FilePath,
		MaxSize:    cfg.Logger.MaxSize,
		MaxBackups: cfg.Logger.MaxBackups,
		MaxAge:     cfg.Logger.MaxAge,
		Compress:   cfg.Logger.Compress,
		WithCaller: cfg.Logger.WithCaller,
	})
	defer logger.Sync()

	logger.Info("Starting MonitoringAnalyticsService", zap.String("version", cfg.Version))

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
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}

	// 自动迁移
	if err := gormDB.AutoMigrate(&infrastructure.MetricModel{}, &infrastructure.SystemHealthModel{}); err != nil {
		logger.Fatal("Failed to migrate database", zap.Error(err))
	}

	// 初始化依赖
	metricRepo := infrastructure.NewMetricRepository(gormDB.DB)
	healthRepo := infrastructure.NewSystemHealthRepository(gormDB.DB)
	svc := application.NewMonitoringAnalyticsService(metricRepo, healthRepo)
	handler := interfaces.NewGRPCHandler(svc)

	// 启动 gRPC 服务
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPC.Port))
	if err != nil {
		logger.Fatal("Failed to listen", zap.Error(err))
	}

	s := grpc.NewServer()
	pb.RegisterMonitoringAnalyticsServiceServer(s, handler)
	// 注册反射服务
	reflection.Register(s)

	// 启动 gRPC 服务
	go func() {
		logger.Info("Starting gRPC server", zap.Int("port", cfg.GRPC.Port))
		if err := s.Serve(lis); err != nil {
			logger.Fatal("Failed to serve", zap.Error(err))
		}
	}()

	// 等待中断信号以优雅地关闭服务器（设置 5 秒的超时时间）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	s.GracefulStop()
	logger.Info("Server exited")
}
