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

	"github.com/gin-gonic/gin"
	accountv1 "github.com/wyfcoding/financialtrading/go-api/account/v1"
	"github.com/wyfcoding/financialtrading/internal/account/application"
	"github.com/wyfcoding/financialtrading/internal/account/domain"
	"github.com/wyfcoding/financialtrading/internal/account/infrastructure/messaging"
	"github.com/wyfcoding/financialtrading/internal/account/infrastructure/persistence"
	"github.com/wyfcoding/financialtrading/internal/account/infrastructure/persistence/mysql"
	"github.com/wyfcoding/financialtrading/internal/account/infrastructure/persistence/redis"
	grpcserver "github.com/wyfcoding/financialtrading/internal/account/interfaces/grpc"
	httpserver "github.com/wyfcoding/financialtrading/internal/account/interfaces/http"
	"github.com/wyfcoding/pkg/cache"
	"github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/database"
	"github.com/wyfcoding/pkg/logging"
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
	// Map config.LogConfig to logging.Config
	logCfg := &logging.Config{
		Service:    cfg.Server.Name,
		Module:     "account",
		Level:      cfg.Log.Level,
		File:       cfg.Log.File,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
	}
	logger := logging.NewFromConfig(logCfg)
	slog.SetDefault(logger.Logger)

	// 3. 初始化指标
	metricsImpl := metrics.NewMetrics(cfg.Server.Name)
	if cfg.Metrics.Enabled {
		// Start metrics server
		go metricsImpl.ExposeHTTP(cfg.Metrics.Port)
	}

	// 4. 初始化基础设施
	// Database
	db, err := database.NewDB(cfg.Data.Database, cfg.CircuitBreaker, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		os.Exit(1)
	}

	// Auto Migrate (仅用于开发方便)
	if cfg.Server.Environment == "dev" {
		if err := db.RawDB().AutoMigrate(&domain.Account{}, &mysql.EventPO{}, &mysql.TransactionPO{}); err != nil {
			slog.Error("failed to migrate database", "error", err)
		}
	}

	// Outbox
	outboxMgr := outbox.NewManager(db.RawDB(), nil)

	// 5. 初始化仓储
	// Redis
	redisCache, err := cache.NewRedisCache(&cfg.Data.Redis, cfg.CircuitBreaker, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to init redis", "error", err)
	}

	mysqlRepo := mysql.NewAccountRepository(db.RawDB())
	redisRepo := redis.NewAccountRedisRepository(redisCache.GetClient())
	accountRepo := persistence.NewCompositeAccountRepository(mysqlRepo, redisRepo)

	eventStore := mysql.NewEventStore(db.RawDB())
	outboxPub := messaging.NewOutboxPublisher(outboxMgr)

	// 6. 初始化应用服务
	commandSvc := application.NewAccountCommandService(accountRepo, eventStore, outboxPub, db.RawDB())
	queryService := application.NewAccountQueryService(accountRepo)
	appService := application.NewAccountService(commandSvc, queryService)

	// 7. 初始化接口层
	// gRPC
	grpcSrv := grpc.NewServer()
	accountSrv := grpcserver.NewHandler(appService)
	accountv1.RegisterAccountServiceServer(grpcSrv, accountSrv)
	reflection.Register(grpcSrv)

	// HTTP
	gin.SetMode(gin.ReleaseMode)
	if cfg.Server.Environment == "dev" {
		gin.SetMode(gin.DebugMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	// Middleware could be added here

	httpHandler := httpserver.NewAccountHandler(appService)
	httpHandler.RegisterRoutes(r.Group("/api"))

	// 8. 启动服务
	g, ctx := errgroup.WithContext(context.Background())

	// gRPC Start
	g.Go(func() error {
		addr := fmt.Sprintf(":%d", cfg.Server.GRPC.Port)
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		slog.Info("gRPC server starting", "addr", addr)
		return grpcSrv.Serve(lis)
	})

	// HTTP Start
	g.Go(func() error {
		addr := fmt.Sprintf(":%d", cfg.Server.HTTP.Port)
		server := &http.Server{
			Addr:    addr,
			Handler: r,
		}
		slog.Info("HTTP server starting", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})

	// 9. 优雅关闭
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
