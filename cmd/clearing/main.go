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
	accountv1 "github.com/wyfcoding/financialtrading/go-api/account/v1"
	clearingv1 "github.com/wyfcoding/financialtrading/go-api/clearing/v1"
	"github.com/wyfcoding/financialtrading/internal/clearing/application"
	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
	"github.com/wyfcoding/financialtrading/internal/clearing/infrastructure/persistence/elasticsearch"
	"github.com/wyfcoding/financialtrading/internal/clearing/infrastructure/persistence/mysql"
	clearingredis "github.com/wyfcoding/financialtrading/internal/clearing/infrastructure/persistence/redis"
	clearingconsumer "github.com/wyfcoding/financialtrading/internal/clearing/interfaces/consumer"
	grpcserver "github.com/wyfcoding/financialtrading/internal/clearing/interfaces/grpc"
	httpserver "github.com/wyfcoding/financialtrading/internal/clearing/interfaces/http"
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

var configPath = flag.String("config", "configs/clearing/config.toml", "config file path")

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

	// 4. Database
	db, err := database.NewDB(cfg.Data.Database, cfg.CircuitBreaker, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		os.Exit(1)
	}
	if cfg.Server.Environment == "dev" {
		if err := db.RawDB().AutoMigrate(&mysql.SettlementModel{}, &outbox.Message{}); err != nil {
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
	accountAddr := cfg.GetGRPCAddr("account")
	if accountAddr == "" {
		accountAddr = "localhost:9090"
	}
	accountConn, err := grpc.NewClient(accountAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to connect account service", "error", err)
		os.Exit(1)
	}
	accountClient := accountv1.NewAccountServiceClient(accountConn)

	// 7. Repositories
	repo := mysql.NewSettlementRepository(db.RawDB())
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
	searchRepo := elasticsearch.NewSettlementSearchRepository(esClient, "settlements")

	redisCache, err := cache.NewRedisCache(&cfg.Data.Redis, cfg.CircuitBreaker, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to init redis", "error", err)
	}
	marginRepo := clearingredis.NewMarginRedisRepository(redisCache.GetClient())
	settlementReadRepo := clearingredis.NewSettlementRedisRepository(redisCache.GetClient())

	// 8. Application
	commandSvc := application.NewClearingCommandService(repo, marginRepo, publisher, accountClient)
	querySvc := application.NewClearingQueryService(repo, searchRepo, settlementReadRepo, marginRepo)

	projectionSvc := application.NewClearingProjectionService(repo, settlementReadRepo, searchRepo, logger.Logger)
	projectionHandler := clearingconsumer.NewSettlementProjectionHandler(projectionSvc, logger.Logger)
	tradeSettlementHandler := clearingconsumer.NewTradeSettlementHandler(commandSvc, logger.Logger)

	projectionTopics := []string{
		domain.SettlementCreatedEventType,
		domain.SettlementCompletedEventType,
		domain.SettlementFailedEventType,
	}
	projectionConsumers := make([]*kafka.Consumer, 0, len(projectionTopics))
	for _, topic := range projectionTopics {
		consumerCfg := cfg.MessageQueue.Kafka
		consumerCfg.Topic = topic
		if consumerCfg.GroupID == "" {
			consumerCfg.GroupID = "clearing-projection-group"
		}
		consumer := kafka.NewConsumer(&consumerCfg, logger, metricsImpl)
		consumer.Start(context.Background(), 3, projectionHandler.Handle)
		projectionConsumers = append(projectionConsumers, consumer)
	}

	settlementConsumerCfg := cfg.MessageQueue.Kafka
	settlementConsumerCfg.Topic = "matching.trade.executed"
	if settlementConsumerCfg.GroupID == "" {
		settlementConsumerCfg.GroupID = "clearing-settlement-group"
	}
	tradeSettlementConsumer := kafka.NewConsumer(&settlementConsumerCfg, logger, metricsImpl)
	tradeSettlementConsumer.Start(context.Background(), 3, tradeSettlementHandler.Handle)

	// 9. Interfaces
	grpcSrv := grpc.NewServer()
	clearingSrv := grpcserver.NewHandler(commandSvc, querySvc)
	clearingv1.RegisterClearingServiceServer(grpcSrv, clearingSrv)
	reflection.Register(grpcSrv)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	httpHandler := httpserver.NewClearingHandler(commandSvc, querySvc)
	httpHandler.RegisterRoutes(r.Group("/api"))

	// 10. Start
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
		if tradeSettlementConsumer != nil {
			_ = tradeSettlementConsumer.Close()
		}
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
