package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/shopspring/decimal"
	"github.com/spf13/viper"
	marketdatav1 "github.com/wyfcoding/financialtrading/go-api/marketdata/v1"
	risk_pb "github.com/wyfcoding/financialtrading/go-api/risk/v1"
	"github.com/wyfcoding/financialtrading/internal/risk/application"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
	risk_client "github.com/wyfcoding/financialtrading/internal/risk/infrastructure/client"
	"github.com/wyfcoding/financialtrading/internal/risk/infrastructure/persistence"
	grpc_server "github.com/wyfcoding/financialtrading/internal/risk/interfaces/grpc"
	"github.com/wyfcoding/pkg/cache"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/security/risk"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/risk/config.toml", "path to config file")
	flag.Parse()

	// 1. Config
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("read config failed: %v", err))
	}

	// 2. Logger
	logger := logging.NewLogger("risk", "main", viper.GetString("log.level"))
	slog.SetDefault(logger.Logger)

	// 3. Database
	dsn := viper.GetString("database.source")
	db, err := gorm.Open(gorm_mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("connect db failed: %v", err))
	}

	// Auto Migrate
	if err := db.AutoMigrate(&domain.RiskLimit{}, &domain.RiskAssessment{}, &domain.RiskMetrics{}, &domain.RiskAlert{}, &domain.CircuitBreaker{}); err != nil {
		panic(fmt.Sprintf("migrate db failed: %v", err))
	}

	// 4. Infrastructure & Domain
	repo := persistence.NewRiskRepository(db)

	ruleEngine := risk.NewBaseEvaluator(logger.Logger)
	localCache, _ := cache.NewBigCache(time.Minute, 100, logger)

	// MarketData Client
	mdAddr := viper.GetString("services.marketdata")
	mdConn, err := grpc.Dial(mdAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Sprintf("failed to connect to marketdata: %v", err))
	}
	mdClient := risk_client.NewGRPCMarketDataClient(marketdatav1.NewMarketDataServiceClient(mdConn))

	// Margin Calculator
	marginCalc := domain.NewVolatilityAdjustedMarginCalculator(
		decimal.NewFromFloat(0.05), // 5% Base Margin
		decimal.NewFromFloat(2.0),  // 2x Volatility Multiplier
		mdClient,
	)

	// 5. Application
	appService := application.NewRiskService(repo, ruleEngine, marginCalc, localCache, logger.Logger)

	// 6. Interfaces
	// gRPC
	grpcSrv := grpc.NewServer()
	riskHandler := grpc_server.NewHandler(appService)
	risk_pb.RegisterRiskServiceServer(grpcSrv, riskHandler)
	reflection.Register(grpcSrv)

	// HTTP Support for Probes, Metrics, Pprof
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	sys := r.Group("/sys")
	{
		sys.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "UP"}) })
		sys.GET("/ready", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "READY"}) })
	}
	r.GET("/metrics", func(c *gin.Context) {
		c.String(http.StatusOK, "# HELP risk_service_running Status of risk service\n# TYPE risk_service_running gauge\nrisk_service_running 1")
	})
	pp := r.Group("/debug/pprof")
	{
		pp.GET("/", gin.WrapF(pprof.Index))
		pp.GET("/cmdline", gin.WrapF(pprof.Cmdline))
		pp.GET("/profile", gin.WrapF(pprof.Profile))
		pp.GET("/symbol", gin.WrapF(pprof.Symbol))
		pp.GET("/trace", gin.WrapF(pprof.Trace))
	}

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
		httpPort = "8087"
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
