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
	"github.com/shopspring/decimal"
	marketdatav1 "github.com/wyfcoding/financialtrading/go-api/marketdata/v1"
	"github.com/wyfcoding/financialtrading/internal/marketdata/application"
	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
	"github.com/wyfcoding/financialtrading/internal/marketdata/infrastructure/analysis"
	"github.com/wyfcoding/financialtrading/internal/marketdata/infrastructure/persistence/elasticsearch"
	"github.com/wyfcoding/financialtrading/internal/marketdata/infrastructure/persistence/mysql"
	redisrepo "github.com/wyfcoding/financialtrading/internal/marketdata/infrastructure/persistence/redis"
	mdconsumer "github.com/wyfcoding/financialtrading/internal/marketdata/interfaces/consumer"
	grpcserver "github.com/wyfcoding/financialtrading/internal/marketdata/interfaces/grpc"
	httpserver "github.com/wyfcoding/financialtrading/internal/marketdata/interfaces/http"
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

var configPath = flag.String("config", "configs/marketdata/config.toml", "config file path")

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
			&mysql.QuoteModel{},
			&mysql.KlineModel{},
			&mysql.TradeModel{},
			&mysql.OrderBookModel{},
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
	mysqlRepo := mysql.NewMarketDataRepository(db.RawDB())
	quoteReadRepo := redisrepo.NewQuoteRedisRepository(redisClient)
	klineReadRepo := redisrepo.NewKlineRedisRepository(redisClient)
	tradeReadRepo := redisrepo.NewTradeRedisRepository(redisClient)
	orderBookReadRepo := redisrepo.NewOrderBookRedisRepository(redisClient)

	publisher := outbox.NewPublisher(outboxMgr)
	var searchRepo domain.MarketDataSearchRepository
	esCfg := &search_pkg.Config{
		ServiceName:         cfg.Server.Name,
		ElasticsearchConfig: cfg.Data.Elasticsearch,
		BreakerConfig:       cfg.CircuitBreaker,
	}
	esClient, err := search_pkg.NewClient(esCfg, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to init elasticsearch", "error", err)
	} else {
		searchRepo = elasticsearch.NewMarketDataSearchRepository(esClient, "", "")
	}

	// 8. Application Services
	historyAnalyzer := analysis.NewPSTHistoryAnalyzer(1_000_000)
	historySvc := application.NewHistoryService(historyAnalyzer)
	commandSvc := application.NewMarketDataCommandService(mysqlRepo, logger.Logger, publisher, historySvc)
	querySvc := application.NewMarketDataQueryService(mysqlRepo, quoteReadRepo, klineReadRepo, tradeReadRepo, orderBookReadRepo, searchRepo, historySvc)
	projectionSvc := application.NewMarketDataProjectionService(quoteReadRepo, klineReadRepo, tradeReadRepo, orderBookReadRepo, searchRepo, logger.Logger)

	// 9. Kafka Consumers (Projection)
	projectionHandler := mdconsumer.NewMarketDataProjectionHandler(projectionSvc, logger.Logger)
	projectionTopics := []string{
		domain.QuoteUpdatedEventType,
		domain.KlineUpdatedEventType,
		domain.TradeExecutedEventType,
		domain.OrderBookUpdatedEventType,
	}
	for _, topic := range projectionTopics {
		consumerCfg := cfg.MessageQueue.Kafka
		consumerCfg.Topic = topic
		if consumerCfg.GroupID == "" {
			consumerCfg.GroupID = "marketdata-projection-group"
		}
		consumer := kafka.NewConsumer(&consumerCfg, logger, metricsImpl)
		consumer.Start(context.Background(), 3, projectionHandler.Handle)
	}

	// 10. Kafka Consumer (Market Price Feed)
	feedCfg := cfg.MessageQueue.Kafka
	if feedCfg.Topic == "" {
		feedCfg.Topic = "market.price"
	}
	if feedCfg.GroupID == "" {
		feedCfg.GroupID = "marketdata-feed-group"
	}
	feedConsumer := kafka.NewConsumer(&feedCfg, logger, metricsImpl)
	feedHandler := mdconsumer.NewMarketDataEventHandler(commandSvc)
	feedConsumer.Start(context.Background(), 1, feedHandler.HandleMarketPrice)

	// 11. Interfaces
	grpcSrv := grpc.NewServer()
	mdHandler := grpcserver.NewHandler(querySvc)
	marketdatav1.RegisterMarketDataServiceServer(grpcSrv, mdHandler)
	reflection.Register(grpcSrv)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	httpHandler := httpserver.NewMarketDataHandler(querySvc)
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
		if err := commandSvc.SaveQuote(c.Request.Context(), application.SaveQuoteCommand{
			Symbol:    cmd.Symbol,
			LastPrice: decimal.Zero,
			LastSize:  decimal.Zero,
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 12. Start
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
