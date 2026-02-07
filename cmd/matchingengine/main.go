package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	clearingv1 "github.com/wyfcoding/financialtrading/go-api/clearing/v1"
	pb "github.com/wyfcoding/financialtrading/go-api/matchingengine/v1"
	orderv1 "github.com/wyfcoding/financialtrading/go-api/order/v1"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/application"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/infrastructure/persistence/elasticsearch"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/infrastructure/persistence/mysql"
	redisrepo "github.com/wyfcoding/financialtrading/internal/matchingengine/infrastructure/persistence/redis"
	meconsumer "github.com/wyfcoding/financialtrading/internal/matchingengine/interfaces/consumer"
	grpcserver "github.com/wyfcoding/financialtrading/internal/matchingengine/interfaces/grpc"
	httpserver "github.com/wyfcoding/financialtrading/internal/matchingengine/interfaces/http"
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

var configPath = flag.String("config", "configs/matchingengine/config.toml", "config file path")

type MatchingConfig struct {
	config.Config `mapstructure:",squash"`
	Matching      struct {
		Symbol string `mapstructure:"symbol" toml:"symbol"`
	} `mapstructure:"matching" toml:"matching"`
}

func main() {
	flag.Parse()

	// 1. Config
	var cfg MatchingConfig
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

	// 4. Database (MySQL)
	db, err := database.NewDB(cfg.Data.Database, cfg.CircuitBreaker, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		os.Exit(1)
	}

	if cfg.Server.Environment == "dev" {
		if err := db.RawDB().AutoMigrate(&mysql.OrderModel{}, &mysql.TradeModel{}, &mysql.OrderBookSnapshotModel{}, &outbox.Message{}); err != nil {
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
	tradeRepo := mysql.NewTradeRepository(db.RawDB())
	orderBookRepo := mysql.NewOrderBookRepository(db.RawDB())
	tradeReadRepo := redisrepo.NewTradeRedisRepository(redisClient)
	orderBookReadRepo := redisrepo.NewOrderBookRedisRepository(redisClient)

	publisher := outbox.NewPublisher(outboxMgr)
	var tradeSearchRepo domain.TradeSearchRepository
	esCfg := &search_pkg.Config{
		ServiceName:         cfg.Server.Name,
		ElasticsearchConfig: cfg.Data.Elasticsearch,
		BreakerConfig:       cfg.CircuitBreaker,
	}
	esClient, err := search_pkg.NewClient(esCfg, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to init elasticsearch", "error", err)
	} else {
		tradeSearchRepo = elasticsearch.NewTradeSearchRepository(esClient, "")
	}

	// 8. Engine
	symbol := cfg.Matching.Symbol
	if symbol == "" {
		symbol = "BTC-USDT"
	}
	engine, err := domain.NewDisruptionEngine(symbol, 1048576, logger.Logger)
	if err != nil {
		panic(fmt.Sprintf("failed to init matching engine: %v", err))
	}

	// 9. Application
	commandSvc := application.NewMatchingCommandService(symbol, engine, tradeRepo, orderBookRepo, publisher, logger.Logger)
	if err := commandSvc.StartEngine(); err != nil {
		slog.Error("failed to start matching engine", "error", err)
	}
	querySvc := application.NewMatchingQueryService(engine, tradeRepo, tradeReadRepo, tradeSearchRepo, orderBookReadRepo)
	projectionSvc := application.NewMatchingProjectionService(tradeReadRepo, tradeSearchRepo, logger.Logger)

	projectionHandler := meconsumer.NewProjectionHandler(projectionSvc, logger.Logger)
	projectionTopics := []string{domain.TradeExecutedEventType}
	for _, topic := range projectionTopics {
		consumerCfg := cfg.MessageQueue.Kafka
		consumerCfg.Topic = topic
		if consumerCfg.GroupID == "" {
			consumerCfg.GroupID = "matching-projection-group"
		}
		consumer := kafka.NewConsumer(&consumerCfg, logger, metricsImpl)
		consumer.Start(context.Background(), 3, projectionHandler.Handle)
	}

	// 10. Downstream Clients & Recover State
	orderAddr := cfg.GetGRPCAddr("order")
	if orderAddr == "" {
		orderAddr = "localhost:50051"
	}
	orderConn, err := grpc.Dial(orderAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to connect order service", "error", err)
		os.Exit(1)
	}
	commandSvc.SetOrderClient(orderv1.NewOrderServiceClient(orderConn))

	clearingAddr := cfg.GetGRPCAddr("clearing")
	if clearingAddr == "" {
		clearingAddr = "localhost:50053"
	}
	clearingConn, err := grpc.Dial(clearingAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to connect clearing service", "error", err)
		os.Exit(1)
	}
	commandSvc.SetClearingClient(clearingv1.NewClearingServiceClient(clearingConn))
	if err := commandSvc.RecoverState(context.Background()); err != nil {
		slog.Error("failed to recover engine state", "error", err)
	}

	// 11. Interfaces (gRPC)
	grpcSrv := grpc.NewServer()
	handler := grpcserver.NewHandler(commandSvc, querySvc)
	pb.RegisterMatchingEngineServiceServer(grpcSrv, handler)
	reflection.Register(grpcSrv)

	// 12. Interfaces (HTTP)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	matchingHTTP := httpserver.NewMatchingHandler(commandSvc, querySvc)
	matchingHTTP.RegisterRoutes(r.Group(""))

	sys := r.Group("/sys")
	{
		sys.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "UP"}) })
		sys.GET("/ready", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "READY"}) })
	}
	r.GET("/metrics", func(c *gin.Context) {
		c.String(http.StatusOK, "# HELP matching_engine_running Status of matching engine\n# TYPE matching_engine_running gauge\nmatching_engine_running 1")
	})
	pp := r.Group("/debug/pprof")
	{
		pp.GET("/", gin.WrapF(pprof.Index))
		pp.GET("/cmdline", gin.WrapF(pprof.Cmdline))
		pp.GET("/profile", gin.WrapF(pprof.Profile))
		pp.GET("/symbol", gin.WrapF(pprof.Symbol))
		pp.GET("/trace", gin.WrapF(pprof.Trace))
	}

	// 13. Start
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
		slog.Info("Starting gRPC server", "addr", addr)
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
