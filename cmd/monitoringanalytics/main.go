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

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gin-gonic/gin"
	v1 "github.com/wyfcoding/financialtrading/go-api/monitoringanalytics/v1"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/application"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
	monitor_ck "github.com/wyfcoding/financialtrading/internal/monitoringanalytics/infrastructure/persistence/clickhouse"
	monitor_es "github.com/wyfcoding/financialtrading/internal/monitoringanalytics/infrastructure/persistence/elasticsearch"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/infrastructure/persistence/mysql"
	redisrepo "github.com/wyfcoding/financialtrading/internal/monitoringanalytics/infrastructure/persistence/redis"
	grpc_server "github.com/wyfcoding/financialtrading/internal/monitoringanalytics/interfaces/grpc"
	http_server "github.com/wyfcoding/financialtrading/internal/monitoringanalytics/interfaces/http"
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

var configPath = flag.String("config", "configs/monitoringanalytics/config.toml", "config file path")

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

	// 4. Database (MySQL)
	db, err := database.NewDB(cfg.Data.Database, cfg.CircuitBreaker, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		os.Exit(1)
	}

	if cfg.Server.Environment == "dev" {
		if err := db.RawDB().AutoMigrate(
			&mysql.MetricModel{},
			&mysql.SystemHealthModel{},
			&mysql.AlertModel{},
			&mysql.TradeMetricModel{},
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
	metricRepo := mysql.NewMetricRepository(db.RawDB())
	healthRepo := mysql.NewSystemHealthRepository(db.RawDB())
	alertRepo := mysql.NewAlertRepository(db.RawDB())

	metricReadRepo := redisrepo.NewMetricRedisRepository(redisClient)
	healthReadRepo := redisrepo.NewSystemHealthRedisRepository(redisClient)
	alertReadRepo := redisrepo.NewAlertRedisRepository(redisClient)

	publisher := outbox.NewPublisher(outboxMgr)

	var auditRepo domain.ExecutionAuditRepository
	if cfg.Data.ClickHouse.Addr != "" {
		ckConn, err := clickhouse.Open(&clickhouse.Options{
			Addr: []string{cfg.Data.ClickHouse.Addr},
			Auth: clickhouse.Auth{
				Database: cfg.Data.ClickHouse.Database,
				Username: cfg.Data.ClickHouse.Username,
				Password: cfg.Data.ClickHouse.Password,
			},
		})
		if err != nil {
			slog.Error("failed to init clickhouse", "error", err)
		} else {
			auditRepo = monitor_ck.NewAuditRepository(ckConn)
		}
	}

	var auditESRepo domain.AuditESRepository
	esCfg := &search_pkg.Config{
		ServiceName:         cfg.Server.Name,
		ElasticsearchConfig: cfg.Data.Elasticsearch,
		BreakerConfig:       cfg.CircuitBreaker,
	}
	esClient, err := search_pkg.NewClient(esCfg, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to init elasticsearch", "error", err)
	} else {
		auditESRepo = monitor_es.NewAuditESRepository(esClient)
	}

	// 8. Application Services
	commandSvc := application.NewMonitoringAnalyticsCommandService(
		metricRepo,
		metricReadRepo,
		healthRepo,
		healthReadRepo,
		alertRepo,
		alertReadRepo,
		auditRepo,
		auditESRepo,
		publisher,
	)
	_ = commandSvc
	querySvc := application.NewMonitoringAnalyticsQueryService(
		metricRepo,
		metricReadRepo,
		healthRepo,
		healthReadRepo,
		alertRepo,
		alertReadRepo,
		auditRepo,
		auditESRepo,
	)

	// 9. Interfaces
	grpcSrv := grpc.NewServer()
	monitoringHandler := grpc_server.NewHandler(querySvc)
	v1.RegisterMonitoringAnalyticsServer(grpcSrv, monitoringHandler)
	reflection.Register(grpcSrv)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	hHandler := http_server.NewMonitoringHandler(querySvc)
	hHandler.RegisterRoutes(r.Group("/api"))

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
		return nil
	})

	if err := g.Wait(); err != nil {
		slog.Error("server exited with error", "error", err)
	}
}
