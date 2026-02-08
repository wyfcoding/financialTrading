// Package main 资金服务启动入口
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/treasury/application"
	"github.com/wyfcoding/financialtrading/internal/treasury/infrastructure"
	"github.com/wyfcoding/financialtrading/internal/treasury/interfaces"
	"github.com/wyfcoding/pkg/messagequeue"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// noopEventPublisher 空操作事件发布者
type noopEventPublisher struct{}

var _ messagequeue.EventPublisher = (*noopEventPublisher)(nil)

func (p *noopEventPublisher) Publish(_ context.Context, _ string, _ string, _ any) error { return nil }
func (p *noopEventPublisher) PublishInTx(_ context.Context, _ any, _ string, _ string, _ any) error {
	return nil
}

// Config 服务配置
type Config struct {
	HTTPPort    int
	GRPCPort    int
	MySQLDSN    string
	KafkaBroker string
	LogLevel    string
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// 配置
	cfg := &Config{
		HTTPPort:    8084,
		GRPCPort:    9084,
		MySQLDSN:    "root:password@tcp(localhost:3306)/treasury?charset=utf8mb4&parseTime=True&loc=Local",
		KafkaBroker: "localhost:9092",
	}

	// 数据库
	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{})
	if err != nil {
		logger.Error("failed to connect database", "error", err)
		os.Exit(1)
	}

	// 仓储
	accountRepo := infrastructure.NewGormAccountRepository(db)
	txRepo := infrastructure.NewGormTransactionRepository(db)

	// 事件
	eventPublisher := &noopEventPublisher{}

	// 服务
	cmdService := application.NewCommandService(accountRepo, txRepo, eventPublisher, logger)
	queryService := application.NewQueryService(accountRepo, txRepo, logger)

	// Handler
	httpHandler := interfaces.NewHTTPHandler(cmdService, queryService)

	// Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Health
	router.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	// API
	api := router.Group("/api/v1")
	httpHandler.RegisterRoutes(api)

	// HTTP Server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// gRPC Server
	grpcServer := grpc.NewServer()

	// Lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)

	// Start HTTP
	g.Go(func() error {
		logger.Info("starting HTTP server", "port", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("HTTP server error: %w", err)
		}
		return nil
	})

	// Start gRPC
	g.Go(func() error {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
		if err != nil {
			return fmt.Errorf("failed to listen gRPC: %w", err)
		}
		logger.Info("starting gRPC server", "port", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			return fmt.Errorf("gRPC server error: %w", err)
		}
		return nil
	})

	// Signals
	g.Go(func() error {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
		}

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		httpServer.Shutdown(shutdownCtx)
		grpcServer.GracefulStop()
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
