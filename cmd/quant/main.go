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
	"github.com/spf13/viper"
	quant_pb "github.com/wyfcoding/financialtrading/go-api/quant/v1"
	"github.com/wyfcoding/financialtrading/internal/quant/application"
	"github.com/wyfcoding/financialtrading/internal/quant/domain"
	"github.com/wyfcoding/financialtrading/internal/quant/infrastructure/client"
	"github.com/wyfcoding/financialtrading/internal/quant/infrastructure/persistence/mysql"
	grpc_server "github.com/wyfcoding/financialtrading/internal/quant/interfaces/grpc"
	http_server "github.com/wyfcoding/financialtrading/internal/quant/interfaces/http"
	"github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/metrics"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/quant/config.toml", "path to config file")
	flag.Parse()

	// 1. Config
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("read config failed: %v", err))
	}

	// 2. Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 3. Database
	dsn := viper.GetString("database.source")
	db, err := gorm.Open(gorm_mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("connect db failed: %v", err))
	}

	// Auto Migrate
	if err := db.AutoMigrate(&domain.Strategy{}, &domain.BacktestResult{}, &domain.Signal{}); err != nil {
		panic(fmt.Sprintf("migrate db failed: %v", err))
	}

	// 4. Infrastructure & Domain
	signalRepo := mysql.NewSignalRepository(db)
	strategyRepo := mysql.NewStrategyRepository(db)
	backtestRepo := mysql.NewBacktestResultRepository(db)

	// Metrics
	metricsImpl := metrics.NewMetrics("quant")

	// Clients
	marketAddr := viper.GetString("services.marketdata.addr")
	if marketAddr == "" {
		marketAddr = "localhost:9090" // Default fallback
	}
	cbCfg := config.CircuitBreakerConfig{
		Enabled:     true,
		Timeout:     5000,
		MaxRequests: 10,
	}
	marketCli, err := client.NewMarketDataClient(marketAddr, metricsImpl, cbCfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create market data client: %v", err))
	}

	// 5. Application
	appService, err := application.NewQuantService(strategyRepo, backtestRepo, signalRepo, marketCli, db)
	if err != nil {
		panic(fmt.Sprintf("failed to init quant service: %v", err))
	}

	// 6. Interfaces
	// gRPC
	grpcSrv := grpc.NewServer()
	quantHandler := grpc_server.NewHandler(appService)
	quant_pb.RegisterQuantServiceServer(grpcSrv, quantHandler)
	reflection.Register(grpcSrv)

	// HTTP
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	hHandler := http_server.NewQuantHandler(appService)
	hHandler.RegisterRoutes(r.Group("/api"))

	// 7. Start
	g, ctx := errgroup.WithContext(context.Background())

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
		httpPort = "8083" // Default for quant?
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

	// 8. Graceful Shutdown
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
