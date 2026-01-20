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
	clearingv1 "github.com/wyfcoding/financialtrading/go-api/clearing/v1"
	"github.com/wyfcoding/financialtrading/internal/clearing/application"
	"github.com/wyfcoding/financialtrading/internal/clearing/infrastructure/persistence/mysql"
	grpcserver "github.com/wyfcoding/financialtrading/internal/clearing/interfaces/grpc"
	httpserver "github.com/wyfcoding/financialtrading/internal/clearing/interfaces/http"
	"github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/database"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/messagequeue/outbox"
	"github.com/wyfcoding/pkg/metrics"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

var configPath = flag.String("config", "configs/clearing/config.toml", "config file path")

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
		Module:     "clearing",
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
		if err := db.RawDB().AutoMigrate(&mysql.SettlementPO{}); err != nil {
			slog.Error("failed to migrate database", "error", err)
		}
	}

	outboxMgr := outbox.NewManager(db.RawDB(), logger.Logger)

	// 5. Downstream Clients
	// Account Service Client
	accountAddr := cfg.GetGRPCAddr("account")
	if accountAddr == "" {
		accountAddr = "localhost:9090"
	}
	accountConn, err := grpc.NewClient(accountAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to connect account service", "error", err)
		os.Exit(1)
	}
	accountClient := accountv1.NewAccountServiceClient(accountConn)

	// 6. Application
	repo := mysql.NewSettlementRepository(db.RawDB())
	appService := application.NewClearingService(repo, outboxMgr, db.RawDB(), accountClient)
	queryService := application.NewClearingQueryService(repo)

	// 7. Interfaces
	grpcSrv := grpc.NewServer()
	clearingSrv := grpcserver.NewHandler(appService, queryService)
	clearingv1.RegisterClearingServiceServer(grpcSrv, clearingSrv)
	reflection.Register(grpcSrv)

	gin.SetMode(gin.ReleaseMode)
	if cfg.Server.Environment == "dev" {
		gin.SetMode(gin.DebugMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())

	httpHandler := httpserver.NewClearingHandler(appService, queryService)
	httpHandler.RegisterRoutes(r.Group("/api"))

	// 8. Start
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
