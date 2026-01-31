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
	positionv1 "github.com/wyfcoding/financialtrading/go-api/position/v1"
	"github.com/wyfcoding/financialtrading/internal/position/application"
	"github.com/wyfcoding/financialtrading/internal/position/domain"
	"github.com/wyfcoding/financialtrading/internal/position/infrastructure/persistence"
	"github.com/wyfcoding/financialtrading/internal/position/infrastructure/persistence/mysql"
	"github.com/wyfcoding/financialtrading/internal/position/infrastructure/persistence/redis"
	grpc_server "github.com/wyfcoding/financialtrading/internal/position/interfaces/grpc"
	http_server "github.com/wyfcoding/financialtrading/internal/position/interfaces/http"
	"github.com/wyfcoding/pkg/cache"
	configpkg "github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/metrics"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/position/config.toml", "path to config file")
	flag.Parse()

	// 1. Config
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("read config failed: %v", err))
	}

	// 2. Logger
	logger := logging.NewFromConfig(&logging.Config{
		Service: "position-service",
		Level:   "info",
	})
	slog.SetDefault(logger.Logger)

	// 3. Database
	dsn := viper.GetString("database.source")
	db, err := gorm.Open(gorm_mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("connect db failed: %v", err))
	}

	// 4. Metrics
	metricsImpl := metrics.NewMetrics("position-service")

	// Auto Migrate
	if err := db.AutoMigrate(&domain.Position{}); err != nil {
		panic(fmt.Sprintf("migrate db failed: %v", err))
	}

	// 5. Infrastructure & Domain
	// Redis
	redisCfg := &configpkg.RedisConfig{
		Addrs: []string{viper.GetString("redis.addr")},
	}
	cbCfg := configpkg.CircuitBreakerConfig{Enabled: false}
	redisCache, err := cache.NewRedisCache(redisCfg, cbCfg, logger, metricsImpl)
	if err != nil {
		slog.Error("failed to init redis", "error", err)
	}

	mysqlRepo := mysql.NewPositionRepository(db)
	redisRepo := redis.NewPositionRedisRepository(redisCache.GetClient())
	repo := persistence.NewCompositePositionRepository(mysqlRepo, redisRepo)

	// 5. Application
	appService, err := application.NewPositionService(repo, db)
	if err != nil {
		panic(fmt.Sprintf("failed to init position service: %v", err))
	}
	// queryService := application.NewPositionQuery(repo) // Not used independently here yet, handler instantiates/uses it?
	// Handler currently takes PositionService only.
	// Wait, Handler uses h.service.GetPositions which is in PositionQuery?
	// In handler.go: `service *application.PositionService`.
	// But `GetPositions` is in `PositionQuery`.
	// I need to correct Handler to accept both or Merge them.
	// `PositionService` in `service.go` does NOT have `GetPositions`.
	// `PositionQuery` in `query.go` HAS `GetPositions`.
	// So Handler needs BOTH or PositionService needs to embed Query.
	// Let's modify Handler to take Query service too.

	// 6. Interfaces
	// gRPC
	grpcSrv := grpc.NewServer()
	h := grpc_server.NewHandler(appService)
	positionv1.RegisterPositionServiceServer(grpcSrv, h)
	reflection.Register(grpcSrv)

	// HTTP
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	hHandler := http_server.NewPositionHandler(appService)
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
		httpPort = "8082" // Default for position?
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
