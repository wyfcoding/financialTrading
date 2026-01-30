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

	"github.com/gin-gonic/gin"

	"github.com/spf13/viper"
	clearingv1 "github.com/wyfcoding/financialtrading/go-api/clearing/v1"
	pb "github.com/wyfcoding/financialtrading/go-api/matchingengine/v1"
	orderv1 "github.com/wyfcoding/financialtrading/go-api/order/v1"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/application"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/infrastructure/messaging"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/infrastructure/persistence"
	grpc_server "github.com/wyfcoding/financialtrading/internal/matchingengine/interfaces/grpc"
	httpserver "github.com/wyfcoding/financialtrading/internal/matchingengine/interfaces/http"
	"github.com/wyfcoding/pkg/messagequeue/outbox"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/matchingengine/config.toml", "path to config file")
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

	// 3. Database (MySQL)
	dsn := viper.GetString("database.source")
	db, err := gorm.Open(gorm_mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("connect db failed: %v", err))
	}

	// 4. Infrastructure
	orderBookRepo := persistence.NewOrderBookRepository(db)
	tradeRepo := persistence.NewTradeRepository(db)
	outboxMgr := outbox.NewManager(db, logger)
	outboxPub := messaging.NewOutboxPublisher(outboxMgr)

	// AutoMigrate
	if err := db.AutoMigrate(&domain.OrderBookSnapshot{}, &domain.Trade{}, &domain.Order{}); err != nil {
		slog.Error("failed to migrate database", "error", err)
	}

	// 5. Domain Engine Configuration
	symbol := viper.GetString("matching.symbol")
	if symbol == "" {
		symbol = "BTC-USDT"
	}

	// 6. Application Service
	service, err := application.NewMatchingService(
		symbol,
		tradeRepo,
		orderBookRepo,
		db,
		outboxPub,
		logger,
	)
	if err != nil {
		panic(fmt.Sprintf("failed to init matching engine: %v", err))
	}

	// 7. Inject Downstream Clients & Recover State
	orderAddr := viper.GetString("service.order_addr")
	if orderAddr == "" {
		orderAddr = "localhost:50051"
	}
	orderConn, _ := grpc.NewClient(orderAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	service.SetOrderClient(orderv1.NewOrderServiceClient(orderConn))

	clearingAddr := viper.GetString("service.clearing_addr")
	if clearingAddr == "" {
		clearingAddr = "localhost:50053"
	}
	clearingConn, _ := grpc.NewClient(clearingAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	service.SetClearingClient(clearingv1.NewClearingServiceClient(clearingConn))

	// 8. Interfaces (gRPC)
	grpcSrv := grpc.NewServer()
	handler := grpc_server.NewHandler(service)
	pb.RegisterMatchingEngineServiceServer(grpcSrv, handler)
	reflection.Register(grpcSrv)

	// 9. Interfaces (HTTP)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Domain Routes
	matchingHttp := httpserver.NewMatchingHandler(service)
	matchingHttp.RegisterRoutes(r.Group(""))

	// Probes, Metrics, Pprof
	sys := r.Group("/sys")
	{
		sys.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "UP"}) })
		sys.GET("/ready", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "READY"}) })
	}
	r.GET("/metrics", func(c *gin.Context) {
		c.String(http.StatusOK, "# HELP matching_engine_running Status of matching engine\n# TYPE matching_engine_running gauge\nmatching_engine_running 1")
	})
	pp := r.Group("/debug/pprof")
	{
		pp.GET("/", gin.WrapF(pprof.Index))
		pp.GET("/cmdline", gin.WrapF(pprof.Cmdline))
		pp.GET("/profile", gin.WrapF(pprof.Profile))
		pp.GET("/symbol", gin.WrapF(pprof.Symbol))
		pp.GET("/trace", gin.WrapF(pprof.Trace))
	}

	// 8. Start
	g, ctx := errgroup.WithContext(context.Background())

	port := viper.GetString("server.grpc_port")
	if port == "" {
		port = "50055"
	}

	g.Go(func() error {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
		if err != nil {
			return err
		}
		slog.Info("Starting gRPC server", "port", port)
		return grpcSrv.Serve(lis)
	})

	httpPort := viper.GetString("server.http_port")
	if httpPort == "" {
		httpPort = "8085"
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

	// 9. Graceful Shutdown
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
