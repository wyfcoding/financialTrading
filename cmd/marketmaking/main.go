package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	marketmakingv1 "github.com/wyfcoding/financialtrading/go-api/marketmaking/v1"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/application"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/infrastructure/client"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/infrastructure/persistence"
	grpcserver "github.com/wyfcoding/financialtrading/internal/marketmaking/interfaces/grpc"
	httpserver "github.com/wyfcoding/financialtrading/internal/marketmaking/interfaces/http"
	"github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/database"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/metrics"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configPath = flag.String("config", "configs/marketmaking/config.toml", "config file path")

func main() {
	flag.Parse()

	// 1. Config
	var cfg config.Config
	if err := config.Load(*configPath, &cfg); err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// 2. Logger
	logCfg := &logging.Config{
		Service:    cfg.Server.Name,
		Module:     "marketmaking",
		Level:      cfg.Log.Level,
		File:       cfg.Log.File,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
	}
	logger := logging.NewFromConfig(logCfg)
	slog.SetDefault(logger.Logger)

	// 3. Metrics
	metricsImpl := metrics.NewMetrics(cfg.Server.Name)
	if cfg.Metrics.Enabled {
		go metricsImpl.ExposeHTTP(cfg.Metrics.Port)
	}

	// 4. Infrastructure
	db, err := database.NewDB(cfg.Data.Database, cfg.CircuitBreaker, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		os.Exit(1)
	}

	if cfg.Server.Environment == "dev" {
		// 自动迁移数据库模型
		if err := db.RawDB().AutoMigrate(&domain.QuoteStrategy{}, &domain.MarketMakingPerformance{}); err != nil {
			slog.Error("failed to migrate database", "error", err)
		}
	}

	strategyRepo := persistence.NewMarketMakingRepository(db.RawDB())

	// Clients
	orderAddr := cfg.GetGRPCAddr("order")
	orderCli, err := client.NewOrderClient(orderAddr, metricsImpl, cfg.CircuitBreaker)
	if err != nil {
		slog.Error("failed to create order client", "error", err)
		os.Exit(1)
	}

	marketAddr := cfg.GetGRPCAddr("marketdata")
	marketCli, err := client.NewMarketDataClient(marketAddr, metricsImpl, cfg.CircuitBreaker)
	if err != nil {
		slog.Error("failed to create market data client", "error", err)
		os.Exit(1)
	}

	// 5. Application

	// 创建事件发布者（简单实现）
	eventPublisher := &simpleEventPublisher{}

	appService := application.NewMarketMakingService(strategyRepo, orderCli, marketCli, eventPublisher)

	// 6. Interfaces
	grpcSrv := grpc.NewServer()

	// MarketMaking Service
	mmHandler := grpcserver.NewHandler(appService)
	marketmakingv1.RegisterMarketMakingServiceServer(grpcSrv, mmHandler)

	reflection.Register(grpcSrv)

	gin.SetMode(gin.ReleaseMode)
	if cfg.Server.Environment == "dev" {
		gin.SetMode(gin.DebugMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())

	httpHandler := httpserver.NewMarketMakingHandler(appService)
	httpHandler.RegisterRoutes(r.Group("/api"))

	// 7. Start
	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		addr := fmt.Sprintf(":%d", cfg.Server.GRPC.Port)
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		slog.Info("gRPC server starting", "addr", addr)
		return grpcSrv.Serve(lis)
	})

	g.Go(func() error {
		addr := fmt.Sprintf(":%d", cfg.Server.HTTP.Port)
		server := &http.Server{Addr: addr, Handler: r}
		slog.Info("HTTP server starting", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})

	g.Go(func() error {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-quit:
			slog.Info("shutting down servers...")
		case <-ctx.Done():
			slog.Info("context cancelled, shutting down...")
		}
		grpcSrv.GracefulStop()
		return nil
	})

	if err := g.Wait(); err != nil {
		slog.Error("server exited with error", "error", err)
	}
}

// simpleEventPublisher 简单的事件发布者实现
type simpleEventPublisher struct{}

// Publish 发布一个普通事件
func (p *simpleEventPublisher) Publish(ctx context.Context, topic string, key string, event any) error {
	// 简单实现，仅记录日志
	slog.Debug("Publishing event", "topic", topic, "key", key, "event", event)
	return nil
}

// PublishInTx 在事务中发布事件
func (p *simpleEventPublisher) PublishInTx(ctx context.Context, tx any, topic string, key string, event any) error {
	// 简单实现，仅记录日志
	slog.Debug("Publishing event in transaction", "topic", topic, "key", key, "event", event)
	return nil
}
