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
	"time"

	"github.com/gin-gonic/gin"
	quantpb "github.com/wyfcoding/financialtrading/go-api/quant/v1"
	"github.com/wyfcoding/financialtrading/internal/quant/application"
	"github.com/wyfcoding/financialtrading/internal/quant/arbitrage"
	"github.com/wyfcoding/financialtrading/internal/quant/domain"
	"github.com/wyfcoding/financialtrading/internal/quant/infrastructure/client"
	"github.com/wyfcoding/financialtrading/internal/quant/infrastructure/persistence/elasticsearch"
	"github.com/wyfcoding/financialtrading/internal/quant/infrastructure/persistence/mysql"
	redisrepo "github.com/wyfcoding/financialtrading/internal/quant/infrastructure/persistence/redis"
	grpcserver "github.com/wyfcoding/financialtrading/internal/quant/interfaces/grpc"
	httpserver "github.com/wyfcoding/financialtrading/internal/quant/interfaces/http"
	"github.com/wyfcoding/pkg/cache"
	"github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/database"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/messagequeue/kafka"
	"github.com/wyfcoding/pkg/messagequeue/outbox"
	"github.com/wyfcoding/pkg/metrics"
	search_pkg "github.com/wyfcoding/pkg/search"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configPath = flag.String("config", "configs/quant/config.toml", "config file path")

func main() {
	flag.Parse()

	// 1. Config
	var cfg config.Config
	if err := config.Load(*configPath, &cfg); err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// 2. Logger
	logCfg := &logging.Config{Service: cfg.Server.Name, Level: cfg.Log.Level}
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
		if err := db.RawDB().AutoMigrate(
			&mysql.StrategyModel{},
			&mysql.BacktestResultModel{},
			&mysql.SignalModel{},
			&outbox.Message{},
		); err != nil {
			slog.Error("failed to migrate database", "error", err)
		}
	}

	// 5. Kafka & Outbox
	kafkaProducer := kafka.NewProducer(&cfg.MessageQueue.Kafka, logger, metricsImpl)
	outboxMgr := outbox.NewManager(db.RawDB(), logger.Logger)
	pusher := func(ctx context.Context, topic, key string, payload []byte) error {
		return kafkaProducer.PublishToTopic(ctx, topic, []byte(key), payload)
	}
	outboxProcessor := outbox.NewProcessor(outboxMgr, pusher, 100, 2*time.Second)

	// 6. Redis
	redisCache, err := cache.NewRedisCache(&cfg.Data.Redis, cfg.CircuitBreaker, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to init redis", "error", err)
		os.Exit(1)
	}
	redisClient := redisCache.GetClient()

	// 7. Repositories
	strategyRepo := mysql.NewStrategyRepository(db.RawDB())
	backtestRepo := mysql.NewBacktestResultRepository(db.RawDB())
	signalRepo := mysql.NewSignalRepository(db.RawDB())

	strategyReadRepo := redisrepo.NewStrategyRedisRepository(redisClient)
	backtestReadRepo := redisrepo.NewBacktestRedisRepository(redisClient)
	signalReadRepo := redisrepo.NewSignalRedisRepository(redisClient)

	publisher := outbox.NewPublisher(outboxMgr)

	var searchRepo domain.QuantSearchRepository
	esCfg := &search_pkg.Config{
		ServiceName:         cfg.Server.Name,
		ElasticsearchConfig: cfg.Data.Elasticsearch,
		BreakerConfig:       cfg.CircuitBreaker,
	}
	esClient, err := search_pkg.NewClient(esCfg, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to init elasticsearch", "error", err)
	} else {
		searchRepo = elasticsearch.NewQuantSearchRepository(esClient, "", "")
	}

	// 8. Clients
	marketAddr := cfg.GetGRPCAddr("marketdata")
	marketCli, err := client.NewMarketDataClient(marketAddr, metricsImpl, cfg.CircuitBreaker)
	if err != nil {
		slog.Error("failed to create market data client", "error", err)
		os.Exit(1)
	}
	marketGrpcCli, err := client.NewMarketDataGRPCClient(marketAddr, metricsImpl, cfg.CircuitBreaker)
	if err != nil {
		slog.Error("failed to create market data grpc client", "error", err)
		os.Exit(1)
	}
	arbEngine := arbitrage.NewEngine(marketGrpcCli)

	// 9. Application
	commandSvc := application.NewQuantCommandService(strategyRepo, backtestRepo, signalRepo, marketCli, publisher, logger.Logger)
	querySvc := application.NewQuantQueryService(strategyRepo, strategyReadRepo, backtestRepo, backtestReadRepo, signalRepo, signalReadRepo, searchRepo, arbEngine)

	// 10. Interfaces
	grpcSrv := grpc.NewServer()
	quantHandler := grpcserver.NewHandler(commandSvc, querySvc)
	quantpb.RegisterQuantServiceServer(grpcSrv, quantHandler)
	reflection.Register(grpcSrv)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	httpHandler := httpserver.NewQuantHandler(commandSvc, querySvc)
	httpHandler.RegisterRoutes(r.Group("/api"))

	// 11. Start
	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		outboxProcessor.Start()
		<-ctx.Done()
		outboxProcessor.Stop()
		return nil
	})

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
