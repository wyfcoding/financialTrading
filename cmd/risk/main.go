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
	"github.com/shopspring/decimal"
	accountv1 "github.com/wyfcoding/financialtrading/go-api/account/v1"
	marketdatav1 "github.com/wyfcoding/financialtrading/go-api/marketdata/v1"
	positionv1 "github.com/wyfcoding/financialtrading/go-api/position/v1"
	riskv1 "github.com/wyfcoding/financialtrading/go-api/risk/v1"
	"github.com/wyfcoding/financialtrading/internal/risk/application"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
	riskclient "github.com/wyfcoding/financialtrading/internal/risk/infrastructure/client"
	"github.com/wyfcoding/financialtrading/internal/risk/infrastructure/persistence/elasticsearch"
	"github.com/wyfcoding/financialtrading/internal/risk/infrastructure/persistence/mysql"
	redisrepo "github.com/wyfcoding/financialtrading/internal/risk/infrastructure/persistence/redis"
	riskconsumer "github.com/wyfcoding/financialtrading/internal/risk/interfaces/consumer"
	grpcserver "github.com/wyfcoding/financialtrading/internal/risk/interfaces/grpc"
	httpserver "github.com/wyfcoding/financialtrading/internal/risk/interfaces/http"
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

var configPath = flag.String("config", "configs/risk/config.toml", "config file path")

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
			&mysql.RiskLimitModel{},
			&mysql.RiskAssessmentModel{},
			&mysql.RiskMetricsModel{},
			&mysql.RiskAlertModel{},
			&mysql.CircuitBreakerModel{},
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
	repo := mysql.NewRiskRepository(db.RawDB())
	readRepo := redisrepo.NewRiskReadRepository(redisClient)

	publisher := outbox.NewPublisher(outboxMgr)

	var searchRepo domain.RiskSearchRepository
	esCfg := &search_pkg.Config{
		ServiceName:         cfg.Server.Name,
		ElasticsearchConfig: cfg.Data.Elasticsearch,
		BreakerConfig:       cfg.CircuitBreaker,
	}
	esClient, err := search_pkg.NewClient(esCfg, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to init elasticsearch", "error", err)
	} else {
		searchRepo = elasticsearch.NewRiskSearchRepository(esClient, "", "")
	}

	// 8. Clients
	accAddr := cfg.GetGRPCAddr("account")
	accConn, err := grpc.Dial(accAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to connect to account", "error", err)
		os.Exit(1)
	}
	accClient := accountv1.NewAccountServiceClient(accConn)

	posAddr := cfg.GetGRPCAddr("position")
	posConn, err := grpc.Dial(posAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to connect to position", "error", err)
		os.Exit(1)
	}
	posClient := positionv1.NewPositionServiceClient(posConn)

	mdAddr := cfg.GetGRPCAddr("marketdata")
	mdConn, err := grpc.Dial(mdAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to connect to marketdata", "error", err)
		os.Exit(1)
	}
	mdClient := riskclient.NewGRPCMarketDataClient(marketdatav1.NewMarketDataServiceClient(mdConn))

	marginCalc := domain.NewVolatilityAdjustedMarginCalculator(
		decimal.NewFromFloat(0.05),
		decimal.NewFromFloat(2.0),
		mdClient,
	)

	// 9. Application
	commandSvc := application.NewRiskCommandService(repo, readRepo, accClient, posClient, publisher, marginCalc)
	querySvc := application.NewRiskQueryService(repo, readRepo, searchRepo)
	projectionSvc := application.NewRiskProjectionService(repo, readRepo, searchRepo, logger.Logger)

	// 10. Kafka Consumers (Projection)
	projectionHandler := riskconsumer.NewProjectionHandler(projectionSvc, logger.Logger)
	projectionTopics := []string{
		domain.RiskAssessmentCreatedEventType,
		domain.RiskLimitUpdatedEventType,
		domain.RiskLimitExceededEventType,
		domain.RiskMetricsUpdatedEventType,
		domain.RiskAlertGeneratedEventType,
		domain.CircuitBreakerFiredEventType,
		domain.CircuitBreakerResetEventType,
	}
	for _, topic := range projectionTopics {
		consumerCfg := cfg.MessageQueue.Kafka
		consumerCfg.Topic = topic
		if consumerCfg.GroupID == "" {
			consumerCfg.GroupID = "risk-projection-group"
		}
		consumer := kafka.NewConsumer(&consumerCfg, logger, metricsImpl)
		consumer.Start(context.Background(), 2, projectionHandler.Handle)
	}

	// Liquidation Engine
	liqEngine := application.NewLiquidationEngine(accClient, posClient, publisher, logger.Logger)

	// 11. Interfaces
	grpcSrv := grpc.NewServer()
	riskHandler := grpcserver.NewHandler(commandSvc, querySvc)
	riskv1.RegisterRiskServiceServer(grpcSrv, riskHandler)
	reflection.Register(grpcSrv)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	httpHandler := httpserver.NewRiskHandler(commandSvc, querySvc)
	httpHandler.RegisterRoutes(r.Group("/api"))

	// Health/pprof
	sys := r.Group("/sys")
	{
		sys.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "UP"}) })
		sys.GET("/ready", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "READY"}) })
	}
	pp := r.Group("/debug/pprof")
	{
		pp.GET("/", gin.WrapF(pprof.Index))
		pp.GET("/cmdline", gin.WrapF(pprof.Cmdline))
		pp.GET("/profile", gin.WrapF(pprof.Profile))
		pp.GET("/symbol", gin.WrapF(pprof.Symbol))
		pp.GET("/trace", gin.WrapF(pprof.Trace))
	}

	// 12. Start
	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		outboxProcessor.Start()
		<-ctx.Done()
		outboxProcessor.Stop()
		return nil
	})

	g.Go(func() error {
		liqEngine.Start(ctx)
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
