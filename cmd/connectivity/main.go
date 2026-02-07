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
	connectivityv1 "github.com/wyfcoding/financialtrading/go-api/connectivity/v1"
	"github.com/wyfcoding/financialtrading/internal/connectivity/application"
	"github.com/wyfcoding/financialtrading/internal/connectivity/infrastructure/client"
	connectivityredis "github.com/wyfcoding/financialtrading/internal/connectivity/infrastructure/persistence/redis"
	grpcserver "github.com/wyfcoding/financialtrading/internal/connectivity/interfaces/grpc"
	"github.com/wyfcoding/pkg/cache"
	"github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/connectivity/fix"
	"github.com/wyfcoding/pkg/database"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/messagequeue/kafka"
	"github.com/wyfcoding/pkg/messagequeue/outbox"
	"github.com/wyfcoding/pkg/metrics"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configPath = flag.String("config", "configs/connectivity/config.toml", "config file path")

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

	// 4. Database (for Outbox)
	db, err := database.NewDB(cfg.Data.Database, cfg.CircuitBreaker, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		os.Exit(1)
	}
	if cfg.Server.Environment == "dev" {
		if err := db.RawDB().AutoMigrate(&outbox.Message{}); err != nil {
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
	quoteRepo := connectivityredis.NewQuoteRedisRepository(redisCache.GetClient())

	// 7. FIX Engine
	sessionMgr := fix.NewSessionManager()
	sessionMgr.AddSession(fix.NewSession("INST_001", "TRANS_GATEWAY", "INST_CLIENT"))

	// 8. Downstream Clients
	execAddr := cfg.GetGRPCAddr("execution")
	if execAddr == "" {
		execAddr = "localhost:9081"
	}
	execCli, _ := client.NewExecutionClient(execAddr)

	// 9. Application Services
	publisher := outbox.NewPublisher(outboxMgr)
	commandSvc := application.NewConnectivityCommandService(sessionMgr, execCli, publisher, quoteRepo)
	querySvc := application.NewConnectivityQueryService(sessionMgr, quoteRepo)

	// 10. gRPC Server
	grpcSrv := grpc.NewServer()
	connectivitySrv := grpcserver.NewHandler(commandSvc, querySvc)
	connectivityv1.RegisterConnectivityServiceServer(grpcSrv, connectivitySrv)
	reflection.Register(grpcSrv)

	// 11. HTTP Server
	r := gin.New()
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "UP"}) })

	// 12. Lifecycle
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
		slog.Info("FIX Gateway simulator starting", "port", 9800)
		<-ctx.Done()
		return nil
	})

	g.Go(func() error {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-quit:
		case <-ctx.Done():
		}
		grpcSrv.GracefulStop()
		return nil
	})

	if err := g.Wait(); err != nil {
		slog.Error("Service exited with error", "error", err)
		os.Exit(1)
	}
}
