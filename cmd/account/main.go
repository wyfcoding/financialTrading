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
	"github.com/wyfcoding/financialtrading/internal/account/application"
	"github.com/wyfcoding/financialtrading/internal/account/domain"
	"github.com/wyfcoding/financialtrading/internal/account/infrastructure/persistence/mysql"
	"github.com/wyfcoding/financialtrading/internal/account/infrastructure/persistence/redis"
	accountconsumer "github.com/wyfcoding/financialtrading/internal/account/interfaces/consumer"
	grpcserver "github.com/wyfcoding/financialtrading/internal/account/interfaces/grpc"
	httpserver "github.com/wyfcoding/financialtrading/internal/account/interfaces/http"
	"github.com/wyfcoding/pkg/cache"
	"github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/database"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/messagequeue/kafka"
	"github.com/wyfcoding/pkg/messagequeue/outbox"
	"github.com/wyfcoding/pkg/metrics"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configPath = flag.String("config", "configs/account/config.toml", "config file path")

func main() {
	flag.Parse()

	// 1. 初始化配置
	var cfg config.Config
	if err := config.Load(*configPath, &cfg); err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// 2. 初始化日志
	logCfg := &logging.Config{Service: cfg.Server.Name, Level: cfg.Log.Level}
	logger := logging.NewFromConfig(logCfg)
	slog.SetDefault(logger.Logger)

	// 3. 初始化指标
	metricsImpl := metrics.NewMetrics(cfg.Server.Name)

	// 4. 初始化基础设施
	db, err := database.NewDB(cfg.Data.Database, cfg.CircuitBreaker, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		os.Exit(1)
	}

	if cfg.Server.Environment == "dev" {
		if err := db.RawDB().AutoMigrate(&mysql.AccountModel{}, &mysql.EventPO{}, &mysql.TransactionPO{}, &outbox.Message{}); err != nil {
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

	// 6. 初始化仓储
	redisCache, err := cache.NewRedisCache(&cfg.Data.Redis, cfg.CircuitBreaker, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to init redis", "error", err)
	}

	mysqlRepo := mysql.NewAccountRepository(db.RawDB())
	redisRepo := redis.NewAccountRedisRepository(redisCache.GetClient())
	eventStore := mysql.NewEventStore(db.RawDB())
	publisher := outbox.NewPublisher(outboxMgr)

	// 7. 初始化应用服务
	commandSvc := application.NewAccountCommandService(mysqlRepo, eventStore, publisher, logger.Logger)
	queryService := application.NewAccountQueryService(mysqlRepo, redisRepo)
	interestJob := application.NewInterestAccrualJob(commandSvc, queryService, logger.Logger)

	// 7.1 Projection Consumers (Account Events -> Redis)
	projectionSvc := application.NewAccountProjectionService(mysqlRepo, redisRepo, logger.Logger)
	projectionHandler := accountconsumer.NewAccountProjectionHandler(projectionSvc, logger.Logger)
	projectionTopics := []string{
		domain.AccountCreatedEventType,
		domain.AccountDepositedEventType,
		domain.AccountWithdrawnEventType,
		domain.AccountFrozenEventType,
		domain.AccountUnfrozenEventType,
		domain.AccountDeductedEventType,
	}
	projectionConsumers := make([]*kafka.Consumer, 0, len(projectionTopics))
	for _, topic := range projectionTopics {
		consumerCfg := cfg.MessageQueue.Kafka
		consumerCfg.Topic = topic
		if consumerCfg.GroupID == "" {
			consumerCfg.GroupID = "account-projection-group"
		}
		consumer := kafka.NewConsumer(&consumerCfg, logger, metricsImpl)
		consumer.Start(context.Background(), 3, projectionHandler.Handle)
		projectionConsumers = append(projectionConsumers, consumer)
	}

	// 8. 初始化接口层
	grpcSrv := grpc.NewServer()
	accountSrv := grpcserver.NewHandler(commandSvc, queryService)
	accountv1.RegisterAccountServiceServer(grpcSrv, accountSrv)
	reflection.Register(grpcSrv)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	httpHandler := httpserver.NewAccountHandler(commandSvc, queryService)
	httpHandler.RegisterRoutes(r.Group("/api"))

	// 9. 启动服务
	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		outboxProcessor.Start()
		<-ctx.Done()
		outboxProcessor.Stop()
		return nil
	})

	// Interest Accrual Job
	g.Go(func() error {
		interestJob.Start(ctx)
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

	// 10. 优雅关闭
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
				c.Close()
			}
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		slog.Error("server exited with error", "error", err)
	}
}
