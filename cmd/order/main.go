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
	"github.com/spf13/viper"
	orderv1 "github.com/wyfcoding/financialtrading/go-api/order/v1"
	"github.com/wyfcoding/financialtrading/internal/order/application"
	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"github.com/wyfcoding/financialtrading/internal/order/infrastructure/persistence/mysql"
	"github.com/wyfcoding/financialtrading/internal/order/infrastructure/search"
	"github.com/wyfcoding/financialtrading/internal/order/interfaces/events"
	grpc_server "github.com/wyfcoding/financialtrading/internal/order/interfaces/grpc"
	http_server "github.com/wyfcoding/financialtrading/internal/order/interfaces/http"
	configpkg "github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/messagequeue/kafka"
	"github.com/wyfcoding/pkg/messagequeue/outbox"
	"github.com/wyfcoding/pkg/metrics"
	search_pkg "github.com/wyfcoding/pkg/search"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/order/config.toml", "path to config file")
	flag.Parse()

	// 1. Config
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("read config failed: %v", err))
	}

	// 2. Logger
	logger := logging.NewFromConfig(&logging.Config{
		Service: "order-service",
		Level:   "info",
	})
	slog.SetDefault(logger.Logger)

	// 3. Database
	dsn := viper.GetString("database.source")
	db, err := gorm.Open(gorm_mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("connect db failed: %v", err))
	}

	// 4. Metrics & Kafka
	metricsImpl := metrics.NewMetrics("order-service")

	kafkaCfg := &configpkg.KafkaConfig{
		Brokers: viper.GetStringSlice("kafka.brokers"),
		Topic:   "order-events",
	}
	kafkaProducer := kafka.NewProducer(kafkaCfg, logger, metricsImpl)

	outboxMgr := outbox.NewManager(db, logger.Logger)
	// 包装推送器以匹配签名
	pusher := func(ctx context.Context, topic, key string, payload []byte) error {
		return kafkaProducer.PublishToTopic(ctx, topic, []byte(key), payload)
	}
	outboxProcessor := outbox.NewProcessor(outboxMgr, pusher, 100, 2*time.Second)

	// Auto Migrate
	if err := db.AutoMigrate(&domain.Order{}, &outbox.Message{}); err != nil {
		panic(fmt.Sprintf("migrate db failed: %v", err))
	}

	// 5. Infrastructure & Domain
	repo := mysql.NewOrderRepository(db)

	// ES Initialization
	esCfg := &search_pkg.Config{
		ServiceName: "order-service",
		ElasticsearchConfig: configpkg.ElasticsearchConfig{
			Addresses: viper.GetStringSlice("elasticsearch.addresses"),
		},
	}
	esClient, err := search_pkg.NewClient(esCfg, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to connect elasticsearch", "error", err)
	}
	searchRepo := search.NewOrderSearchRepository(esClient)

	// 6. Application
	orderService, err := application.NewOrderService(repo, searchRepo, db)
	if err != nil {
		panic(fmt.Sprintf("failed to init order service: %v", err))
	}

	// 7. Event Handlers (Syncers)
	consumerCfg := &configpkg.KafkaConfig{
		Brokers: viper.GetStringSlice("kafka.brokers"),
		Topic:   "order-events",
		GroupID: "order-search-syncer",
	}
	kafkaConsumer := kafka.NewConsumer(consumerCfg, logger, metricsImpl)
	searchHandler := events.NewOrderSearchHandler(searchRepo, repo, kafkaConsumer, 5)

	// 8. Interfaces
	grpcSrv := grpc.NewServer()
	h := grpc_server.NewHandler(orderService)
	orderv1.RegisterOrderServiceServer(grpcSrv, h)
	reflection.Register(grpcSrv)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	hHandler := http_server.NewOrderHandler(orderService)
	hHandler.RegisterRoutes(r.Group("/api"))

	// 9. Start
	g, ctx := errgroup.WithContext(context.Background())

	// Outbox Processor
	g.Go(func() error {
		outboxProcessor.Start()
		<-ctx.Done()
		outboxProcessor.Stop()
		return nil
	})

	// Search Syncer
	g.Go(func() error {
		searchHandler.Start(ctx)
		return nil
	})

	grpcPort := viper.GetString("server.grpc_port")
	g.Go(func() error {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
		if err != nil {
			return err
		}
		slog.Info("Starting gRPC server", "port", grpcPort)
		return grpcSrv.Serve(lis)
	})

	httpPort := viper.GetString("server.http_port")
	if httpPort == "" {
		httpPort = "8081"
	}
	g.Go(func() error {
		addr := fmt.Sprintf(":%s", httpPort)
		server := &http.Server{Addr: addr, Handler: r}
		slog.Info("HTTP server starting", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})

	// 10. Graceful Shutdown
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
