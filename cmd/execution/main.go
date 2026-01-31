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
	executionv1 "github.com/wyfcoding/financialtrading/go-api/execution/v1"
	marketdatav1 "github.com/wyfcoding/financialtrading/go-api/marketdata/v1"
	orderv1 "github.com/wyfcoding/financialtrading/go-api/order/v1"
	"github.com/wyfcoding/financialtrading/internal/account/infrastructure/messaging"
	"github.com/wyfcoding/financialtrading/internal/execution/application"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
	"github.com/wyfcoding/financialtrading/internal/execution/infrastructure"
	"github.com/wyfcoding/financialtrading/internal/execution/infrastructure/client"
	"github.com/wyfcoding/financialtrading/internal/execution/infrastructure/persistence/mysql"
	execution_redis "github.com/wyfcoding/financialtrading/internal/execution/infrastructure/persistence/redis"
	"github.com/wyfcoding/financialtrading/internal/execution/infrastructure/search"
	"github.com/wyfcoding/financialtrading/internal/execution/interfaces/events"
	grpcserver "github.com/wyfcoding/financialtrading/internal/execution/interfaces/grpc"
	httpserver "github.com/wyfcoding/financialtrading/internal/execution/interfaces/http"
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

var configPath = flag.String("config", "configs/execution/config.toml", "config file path")

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
		Module:     "execution",
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

	// 4. Infrastructure
	db, err := database.NewDB(cfg.Data.Database, cfg.CircuitBreaker, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		os.Exit(1)
	}

	if cfg.Server.Environment == "dev" {
		if err := db.RawDB().AutoMigrate(&domain.Trade{}, &domain.AlgoOrder{}, &outbox.Message{}); err != nil {
			slog.Error("failed to migrate database", "error", err)
		}
	}

	// 5. Kafka & Outbox
	kafkaProducer := kafka.NewProducer(&cfg.MessageQueue.Kafka, logger, metricsImpl)
	outboxMgr := outbox.NewManager(db.RawDB(), logger.Logger)
	// 包装推送器以匹配签名
	pusher := func(ctx context.Context, topic, key string, payload []byte) error {
		return kafkaProducer.PublishToTopic(ctx, topic, []byte(key), payload)
	}
	outboxProcessor := outbox.NewProcessor(outboxMgr, pusher, 100, 2*time.Second)

	// Clients
	orderAddr := cfg.GetGRPCAddr("order")
	if orderAddr == "" {
		orderAddr = "localhost:50051"
	}
	orderConn, err := grpc.Dial(orderAddr, grpc.WithInsecure())
	if err != nil {
		slog.Error("failed to connect to order service", "error", err)
		os.Exit(1)
	}
	orderCli := orderv1.NewOrderServiceClient(orderConn)

	mdAddr := cfg.GetGRPCAddr("marketdata")
	if mdAddr == "" {
		mdAddr = "localhost:50052"
	}
	mdConn, err := grpc.Dial(mdAddr, grpc.WithInsecure())
	if err != nil {
		slog.Error("failed to connect to marketdata service", "error", err)
	}

	// 6. Application Services
	tradeRepo := mysql.NewTradeRepository(db.RawDB())
	algoRepo := mysql.NewAlgoOrderRepository(db.RawDB())
	outboxPub := messaging.NewOutboxPublisher(outboxMgr)

	// ES & Redis Repositories
	esCfg := &search_pkg.Config{
		ServiceName:         cfg.Server.Name,
		ElasticsearchConfig: cfg.Data.Elasticsearch,
	}
	esClient, err := search_pkg.NewClient(esCfg, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to init elasticsearch", "error", err)
	}
	tradeSearchRepo := search.NewTradeSearchRepository(esClient)

	redisCache, err := cache.NewRedisCache(&cfg.Data.Redis, cfg.CircuitBreaker, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to init redis", "error", err)
	}
	redisRepo := execution_redis.NewAlgoRedisRepository(redisCache.GetClient())

	mdCli := marketdatav1.NewMarketDataServiceClient(mdConn)
	mdProvider := client.NewGRPCMarketDataProvider(mdCli)
	volumeProvider := infrastructure.NewMockVolumeProfileProvider()

	executionSvc := application.NewExecutionService(
		tradeRepo,
		tradeSearchRepo,
		algoRepo,
		redisRepo,
		outboxPub,
		orderCli,
		mdProvider,
		volumeProvider,
		metricsImpl,
		db.RawDB(),
	)

	// 7. Event Handlers (Syncers)
	searchConsumer := kafka.NewConsumer(&cfg.MessageQueue.Kafka, logger, metricsImpl)
	tradeSearchHandler := events.NewTradeSearchHandler(tradeSearchRepo, tradeRepo, searchConsumer, 5)

	redisConsumer := kafka.NewConsumer(&cfg.MessageQueue.Kafka, logger, metricsImpl)
	algoRedisHandler := events.NewAlgoRedisHandler(redisRepo, algoRepo, redisConsumer, 5)

	// 8. Interfaces
	grpcSrv := grpc.NewServer()
	executionHandler := grpcserver.NewHandler(executionSvc)
	executionv1.RegisterExecutionServiceServer(grpcSrv, executionHandler)
	reflection.Register(grpcSrv)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	httpHandler := httpserver.NewExecutionHandler(executionSvc)
	httpHandler.RegisterRoutes(r.Group("/api"))

	// 9. Start
	g, ctx := errgroup.WithContext(context.Background())

	// Outbox
	g.Go(func() error {
		outboxProcessor.Start()
		<-ctx.Done()
		outboxProcessor.Stop()
		return nil
	})

	// Syncers
	g.Go(func() error {
		tradeSearchHandler.Start(ctx)
		return nil
	})
	g.Go(func() error {
		algoRedisHandler.Start(ctx)
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
		slog.Info("Execution algorithm worker starting")
		executionSvc.StartAlgoWorker(ctx)
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
