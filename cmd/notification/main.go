package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/fynnwu/FinancialTrading/go-api/notification"
	"github.com/fynnwu/FinancialTrading/internal/notification/application"
	"github.com/fynnwu/FinancialTrading/internal/notification/infrastructure"
	"github.com/fynnwu/FinancialTrading/internal/notification/interfaces"
	"github.com/fynnwu/FinancialTrading/pkg/config"
	"github.com/fynnwu/FinancialTrading/pkg/db"
	"github.com/fynnwu/FinancialTrading/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 1. 加载配置
	configPath := "configs/notification/config.toml"
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
	logger.Info(ctx, "Starting NotificationService", "version", cfg.Version)

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
	if err := gormDB.AutoMigrate(&infrastructure.NotificationModel{}); err != nil {
		logger.Fatal(ctx, "Failed to migrate database", "error", err)
	}

	// 初始化依赖
	emailSender := infrastructure.NewMockEmailSender()
	smsSender := infrastructure.NewMockSMSSender()
	repo := infrastructure.NewNotificationRepository(gormDB.DB)
	svc := application.NewNotificationService(repo, emailSender, smsSender)
	handler := interfaces.NewGRPCHandler(svc)

	// 启动 gRPC 服务
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPC.Port))
	if err != nil {
		logger.Fatal(ctx, "Failed to listen", "error", err)
	}

	s := grpc.NewServer()
	pb.RegisterNotificationServiceServer(s, handler)
	// 注册反射服务
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

	// 关闭 HTTP 服务器
	// shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// defer cancel()
	// if err := httpServer.Shutdown(shutdownCtx); err != nil {
	// 	logger.Error(ctx, "HTTP server shutdown error", "error", err)
	// }

	s.GracefulStop()
	logger.Info(ctx, "Server exited")
}
