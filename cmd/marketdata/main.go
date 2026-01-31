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
	"github.com/shopspring/decimal"
	marketdatav1 "github.com/wyfcoding/financialtrading/go-api/marketdata/v1"
	"github.com/wyfcoding/financialtrading/internal/marketdata/application"
	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
	"github.com/wyfcoding/financialtrading/internal/marketdata/infrastructure/persistence"
	"github.com/wyfcoding/financialtrading/internal/marketdata/infrastructure/persistence/mysql"
	persistence_redis "github.com/wyfcoding/financialtrading/internal/marketdata/infrastructure/persistence/redis"
	"github.com/wyfcoding/financialtrading/internal/marketdata/interfaces/events"
	grpcserver "github.com/wyfcoding/financialtrading/internal/marketdata/interfaces/grpc"
	httpserver "github.com/wyfcoding/financialtrading/internal/marketdata/interfaces/http"
	"github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/database"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/messagequeue/kafka"
	"github.com/wyfcoding/pkg/metrics"
	"github.com/wyfcoding/pkg/redis"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configPath = flag.String("config", "configs/marketdata/config.toml", "config file path")

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
		Module:     "marketdata",
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

	// Auto Migrate
	if cfg.Server.Environment == "dev" {
		if err := db.RawDB().AutoMigrate(&domain.Quote{}, &domain.Kline{}, &domain.Trade{}, &domain.OrderBook{}); err != nil {
			slog.Error("failed to migrate database", "error", err)
		}
	}

	// 5. Redis
	redisClient, redisCleanup, err := redis.NewClient(&cfg.Data.Redis, logger)
	if err != nil {
		slog.Error("failed to connect redis", "error", err)
		os.Exit(1)
	}
	defer redisCleanup()

	// 6. Repository & Application
	mysqlRepo := mysql.NewMarketDataRepository(db.RawDB())
	redisRepo := persistence_redis.NewQuoteRepository(redisClient)

	// Combine into composite repository (standard CQRS pattern)
	repo := persistence.NewCompositeMarketDataRepository(mysqlRepo, redisRepo)

	// 创建事件发布者
	eventPublisher := &dummyEventPublisher{}

	serviceFacade := application.NewMarketDataService(repo, slog.Default(), eventPublisher)

	// Kafka Consumer
	kafkaCfg := &cfg.MessageQueue.Kafka
	kafkaCfg.GroupID = "marketdata-group"
	kafkaCfg.Topic = "market.price"

	consumer := kafka.NewConsumer(kafkaCfg, logger, metricsImpl)
	eventHandler := events.NewMarketDataEventHandler(serviceFacade)
	eventHandler.Subscribe(context.Background(), consumer)

	// 6. Interfaces
	grpcSrv := grpc.NewServer()
	mdHandler := grpcserver.NewHandler(serviceFacade)
	marketdatav1.RegisterMarketDataServiceServer(grpcSrv, mdHandler)
	reflection.Register(grpcSrv)

	gin.SetMode(gin.ReleaseMode)
	if cfg.Server.Environment == "dev" {
		gin.SetMode(gin.DebugMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())

	httpHandler := httpserver.NewMarketDataHandler(serviceFacade)
	httpHandler.RegisterRoutes(r.Group("/api"))

	// Temporary: Ingest endpoints for testing
	r.POST("/api/v1/marketdata/quote", func(c *gin.Context) {
		var cmd struct {
			Symbol string `json:"symbol"`
		}
		if err := c.ShouldBindJSON(&cmd); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := serviceFacade.SaveQuote(c.Request.Context(), cmd.Symbol, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, 0, ""); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

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

// dummyEventPublisher 简单的事件发布者实现
type dummyEventPublisher struct{}

// Publish 发布一个普通事件
func (p *dummyEventPublisher) Publish(ctx context.Context, topic string, key string, event any) error {
	// 简单实现，仅记录日志
	slog.Debug("Publishing event", "topic", topic, "key", key, "event", event)
	return nil
}

// PublishInTx 在事务中发布事件
func (p *dummyEventPublisher) PublishInTx(ctx context.Context, tx any, topic string, key string, event any) error {
	// 简单实现，仅记录日志
	slog.Debug("Publishing event in transaction", "topic", topic, "key", key, "event", event)
	return nil
}
