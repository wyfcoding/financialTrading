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
	executionv1 "github.com/wyfcoding/financialtrading/go-api/execution/v1"
	marketdatav1 "github.com/wyfcoding/financialtrading/go-api/marketdata/v1"
	orderv1 "github.com/wyfcoding/financialtrading/go-api/order/v1"
	"github.com/wyfcoding/financialtrading/internal/account/infrastructure/messaging"
	"github.com/wyfcoding/financialtrading/internal/execution/application"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
	"github.com/wyfcoding/financialtrading/internal/execution/infrastructure"
	"github.com/wyfcoding/financialtrading/internal/execution/infrastructure/client"
	"github.com/wyfcoding/financialtrading/internal/execution/infrastructure/persistence/mysql"
	grpcserver "github.com/wyfcoding/financialtrading/internal/execution/interfaces/grpc"
	httpserver "github.com/wyfcoding/financialtrading/internal/execution/interfaces/http"
	"github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/database"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/messagequeue/outbox"
	"github.com/wyfcoding/pkg/metrics"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configPath = flag.String("config", "configs/execution/config.toml", "config file path")

func main() {
	flag.Parse()

	// 1. Config
	var cfg config.Config
	if err := config.Load(*configPath, &cfg); err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// 2. Logger
	logCfg := &logging.Config{
		Service:    cfg.Server.Name,
		Module:     "execution",
		Level:      cfg.Log.Level,
		File:       cfg.Log.File,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
	}
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
		if err := db.RawDB().AutoMigrate(&domain.Trade{}, &domain.AlgoOrder{}); err != nil {
			slog.Error("failed to migrate database", "error", err)
		}
	}

	outboxMgr := outbox.NewManager(db.RawDB(), logger.Logger)

	// 5. Clients
	orderAddr := cfg.GetGRPCAddr("order")
	if orderAddr == "" {
		orderAddr = "localhost:50051"
	}
	orderConn, err := grpc.Dial(orderAddr, grpc.WithInsecure())
	if err != nil {
		slog.Error("failed to connect to order service", "error", err)
		os.Exit(1)
	}
	orderCli := orderv1.NewOrderServiceClient(orderConn)

	mdAddr := cfg.GetGRPCAddr("marketdata")
	if mdAddr == "" {
		mdAddr = "localhost:50052"
	}
	mdConn, err := grpc.Dial(mdAddr, grpc.WithInsecure())
	if err != nil {
		slog.Error("failed to connect to marketdata service", "error", err)
	}

	// 5. Application Services
	tradeRepo := mysql.NewTradeRepository(db.RawDB())
	algoRepo := mysql.NewAlgoOrderRepository(db.RawDB())
	outboxPub := messaging.NewOutboxPublisher(outboxMgr)

	mdCli := marketdatav1.NewMarketDataServiceClient(mdConn)
	mdProvider := client.NewGRPCMarketDataProvider(mdCli)
	volumeProvider := infrastructure.NewMockVolumeProfileProvider()

	executionSvc := application.NewExecutionService(
		tradeRepo,
		algoRepo,
		outboxPub,
		orderCli,
		mdProvider,
		volumeProvider,
		metricsImpl,
		db.RawDB(),
	)

	// 6. Interfaces
	grpcSrv := grpc.NewServer()
	executionHandler := grpcserver.NewHandler(executionSvc)
	executionv1.RegisterExecutionServiceServer(grpcSrv, executionHandler)
	reflection.Register(grpcSrv)

	gin.SetMode(gin.ReleaseMode)
	if cfg.Server.Environment == "dev" {
		gin.SetMode(gin.DebugMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())

	httpHandler := httpserver.NewExecutionHandler(executionSvc)
	httpHandler.RegisterRoutes(r.Group("/api"))

	// 7. Start
	g, ctx := errgroup.WithContext(context.Background())

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
		slog.Info("Execution algorithm worker starting")
		executionSvc.StartAlgoWorker(ctx)
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
