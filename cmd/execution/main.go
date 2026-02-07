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
	"github.com/wyfcoding/financialtrading/internal/execution/application"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
	"github.com/wyfcoding/financialtrading/internal/execution/infrastructure"
	"github.com/wyfcoding/financialtrading/internal/execution/infrastructure/client"
	"github.com/wyfcoding/financialtrading/internal/execution/infrastructure/persistence/elasticsearch"
	"github.com/wyfcoding/financialtrading/internal/execution/infrastructure/persistence/mysql"
	executionredis "github.com/wyfcoding/financialtrading/internal/execution/infrastructure/persistence/redis"
	executionconsumer "github.com/wyfcoding/financialtrading/internal/execution/interfaces/consumer"
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
	"google.golang.org/grpc/credentials/insecure"
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
	logCfg := &logging.Config{Service: cfg.Server.Name, Level: cfg.Log.Level}
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
		if err := db.RawDB().AutoMigrate(&mysql.TradeModel{}, &mysql.AlgoOrderModel{}, &mysql.EventPO{}, &outbox.Message{}); err != nil {
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

	// 6. Downstream Clients
	orderAddr := cfg.GetGRPCAddr("order")
	if orderAddr == "" {
		orderAddr = "localhost:50051"
	}
	orderConn, err := grpc.Dial(orderAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to connect to order service", "error", err)
		os.Exit(1)
	}
	orderCli := orderv1.NewOrderServiceClient(orderConn)

	mdAddr := cfg.GetGRPCAddr("marketdata")
	if mdAddr == "" {
		mdAddr = "localhost:50052"
	}
	mdConn, err := grpc.Dial(mdAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to connect to marketdata service", "error", err)
	}
	mdCli := marketdatav1.NewMarketDataServiceClient(mdConn)

	// 7. Repositories
	tradeRepo := mysql.NewTradeRepository(db.RawDB())
	algoRepo := mysql.NewAlgoOrderRepository(db.RawDB())
	eventStore := mysql.NewEventStore(db.RawDB())
	publisher := outbox.NewPublisher(outboxMgr)

	esCfg := &search_pkg.Config{
		ServiceName:         cfg.Server.Name,
		ElasticsearchConfig: cfg.Data.Elasticsearch,
		BreakerConfig:       cfg.CircuitBreaker,
	}
	esClient, err := search_pkg.NewClient(esCfg, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to init elasticsearch", "error", err)
	}
	tradeSearchRepo := elasticsearch.NewTradeSearchRepository(esClient, "trades")

	redisCache, err := cache.NewRedisCache(&cfg.Data.Redis, cfg.CircuitBreaker, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to init redis", "error", err)
	}
	algoRedisRepo := executionredis.NewAlgoRedisRepository(redisCache.GetClient())
	tradeReadRepo := executionredis.NewTradeRedisRepository(redisCache.GetClient())

	mdProvider := client.NewGRPCMarketDataProvider(mdCli)
	volumeProvider := infrastructure.NewMockVolumeProfileProvider()

	// 8. Application Services
	commandSvc := application.NewExecutionCommandService(
		tradeRepo,
		algoRepo,
		algoRedisRepo,
		eventStore,
		publisher,
		orderCli,
		mdProvider,
		volumeProvider,
	)
	querySvc := application.NewExecutionQueryService(tradeRepo, tradeSearchRepo, tradeReadRepo)

	projectionSvc := application.NewExecutionProjectionService(tradeRepo, tradeReadRepo, tradeSearchRepo, algoRepo, algoRedisRepo, logger.Logger)
	tradeProjectionHandler := executionconsumer.NewTradeProjectionHandler(projectionSvc, logger.Logger)
	algoProjectionHandler := executionconsumer.NewAlgoProjectionHandler(projectionSvc, logger.Logger)

	projectionTopics := []string{domain.TradeExecutedEventType, domain.AlgoOrderStartedEventType}
	projectionConsumers := make([]*kafka.Consumer, 0, len(projectionTopics))
	for _, topic := range projectionTopics {
		consumerCfg := cfg.MessageQueue.Kafka
		consumerCfg.Topic = topic
		if consumerCfg.GroupID == "" {
			consumerCfg.GroupID = "execution-projection-group"
		}
		consumer := kafka.NewConsumer(&consumerCfg, logger, metricsImpl)
		if topic == domain.TradeExecutedEventType {
			consumer.Start(context.Background(), 3, tradeProjectionHandler.Handle)
		} else {
			consumer.Start(context.Background(), 3, algoProjectionHandler.Handle)
		}
		projectionConsumers = append(projectionConsumers, consumer)
	}

	// 9. Interfaces
	grpcSrv := grpc.NewServer()
	executionHandler := grpcserver.NewHandler(commandSvc, querySvc)
	executionv1.RegisterExecutionServiceServer(grpcSrv, executionHandler)
	reflection.Register(grpcSrv)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	httpHandler := httpserver.NewExecutionHandler(commandSvc, querySvc)
	httpHandler.RegisterRoutes(r.Group("/api"))

	// 10. Start
	g, ctx := errgroup.WithContext(context.Background())

	// Outbox
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
		slog.Info("Execution algorithm worker starting")
		commandSvc.StartAlgoWorker(ctx)
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
		for _, c := range projectionConsumers {
			if c != nil {
				_ = c.Close()
			}
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		slog.Error("server exited with error", "error", err)
	}
}
