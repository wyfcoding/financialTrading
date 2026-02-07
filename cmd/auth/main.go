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
	authv1 "github.com/wyfcoding/financialtrading/go-api/auth/v1"
	"github.com/wyfcoding/financialtrading/internal/auth/application"
	"github.com/wyfcoding/financialtrading/internal/auth/infrastructure/persistence/mysql"
	authredis "github.com/wyfcoding/financialtrading/internal/auth/infrastructure/persistence/redis"
	grpcserver "github.com/wyfcoding/financialtrading/internal/auth/interfaces/grpc"
	httpserver "github.com/wyfcoding/financialtrading/internal/auth/interfaces/http"
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

var configPath = flag.String("config", "configs/auth/config.toml", "config file path")

func main() {
	flag.Parse()

	// 1. 配置
	var cfg config.Config
	if err := config.Load(*configPath, &cfg); err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// 2. 日志
	logCfg := &logging.Config{Service: cfg.Server.Name, Level: cfg.Log.Level}
	logger := logging.NewFromConfig(logCfg)
	slog.SetDefault(logger.Logger)

	// 3. 指标
	metricsImpl := metrics.NewMetrics(cfg.Server.Name)

	// 4. 数据库
	db, err := database.NewDB(cfg.Data.Database, cfg.CircuitBreaker, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		os.Exit(1)
	}
	if cfg.Server.Environment == "dev" {
		if err := db.RawDB().AutoMigrate(&mysql.UserModel{}, &mysql.APIKeyModel{}, &outbox.Message{}); err != nil {
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
	}

	// 7. 仓储
	userRepo := mysql.NewUserRepository(db.RawDB())
	apiKeyRepo := mysql.NewAPIKeyRepository(db.RawDB())
	publisher := outbox.NewPublisher(outboxMgr)

	sessionRepo := authredis.NewSessionRedisRepository(redisCache.GetClient())
	apiKeyRedisRepo := authredis.NewAPIKeyRedisRepository(redisCache.GetClient())

	// 8. 应用服务
	keySvc := application.NewAPIKeyService(apiKeyRepo)
	commandSvc := application.NewAuthCommandService(userRepo, apiKeyRepo, apiKeyRedisRepo, sessionRepo, keySvc, publisher)
	querySvc := application.NewAuthQueryService(userRepo, apiKeyRepo, apiKeyRedisRepo, sessionRepo)

	// 9. 接口层
	grpcSrv := grpc.NewServer()
	authHandler := grpcserver.NewHandler(commandSvc, querySvc)
	authv1.RegisterAuthServiceServer(grpcSrv, authHandler)
	reflection.Register(grpcSrv)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	httpHandler := httpserver.NewHandler(commandSvc, querySvc)
	httpHandler.RegisterRoutes(r.Group("/api"))

	// 10. 启动
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
