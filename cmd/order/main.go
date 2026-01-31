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

	"net/http/pprof"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	orderv1 "github.com/wyfcoding/financialtrading/go-api/order/v1"
	"github.com/wyfcoding/financialtrading/internal/order/application"
	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"github.com/wyfcoding/financialtrading/internal/order/infrastructure/persistence/mysql"
	"github.com/wyfcoding/financialtrading/internal/order/infrastructure/search"
	"github.com/wyfcoding/financialtrading/internal/order/interfaces/events"
	grpc_server "github.com/wyfcoding/financialtrading/internal/order/interfaces/grpc"
	http_server "github.com/wyfcoding/financialtrading/internal/order/interfaces/http"
	configpkg "github.com/wyfcoding/pkg/config"
	search_pkg "github.com/wyfcoding/pkg/search"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/order/config.toml", "path to config file")
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
	if err := db.AutoMigrate(&domain.Order{}); err != nil {
		panic(fmt.Sprintf("migrate db failed: %v", err))
	}

	// 4. Infrastructure & Domain
	repo := mysql.NewOrderRepository(db)

	// ES Initialization
	esCfg := &search_pkg.Config{
		ServiceName: "order-service",
		ElasticsearchConfig: configpkg.ElasticsearchConfig{
			Addresses: viper.GetStringSlice("elasticsearch.addresses"),
		},
	}
	esClient, err := search_pkg.NewClient(esCfg, nil, nil)
	if err != nil {
		slog.Error("failed to connect elasticsearch", "error", err)
	}
	searchRepo := search.NewOrderSearchRepository(esClient)

	// 5. Application
	// Inject dependencies
	orderService, err := application.NewOrderService(repo, searchRepo, db)
	if err != nil {
		panic(fmt.Sprintf("failed to init order service: %v", err))
	}

	// Configure DTM/Remote Services if needed (from config)
	if dtmServer := viper.GetString("dtm.server"); dtmServer != "" {
		orderService.SetDTMServer(dtmServer)
	}
	// TODO: SetRiskClient, SetAccountClient, SetPositionClient using gRPC connection pools

	// 7. Event Handlers
	searchHandler := events.NewOrderSearchHandler(searchRepo, repo)
	searchHandler.Subscribe(context.Background(), nil)

	// 6. Interfaces
	// gRPC
	grpcSrv := grpc.NewServer()
	handler := grpc_server.NewHandler(orderService)
	orderv1.RegisterOrderServiceServer(grpcSrv, handler)
	reflection.Register(grpcSrv)

	// HTTP
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	hHandler := http_server.NewOrderHandler(orderService)
	hHandler.RegisterRoutes(r.Group("/api"))

	// System Endpoints (Health, Metrics, Pprof)
	sys := r.Group("/sys")
	{
		sys.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "UP"}) })
		sys.GET("/ready", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "READY"}) })
	}

	// Metrics
	r.GET("/metrics", func(c *gin.Context) {
		// In a real app, this should expose Prometheus stats
		c.String(http.StatusOK, "# HELP order_service_running Status of order service\n# TYPE order_service_running gauge\norder_service_running 1")
	})

	// Pprof
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
		httpPort = "8081" // Default for order?
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
