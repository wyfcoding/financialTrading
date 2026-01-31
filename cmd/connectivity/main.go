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
	"github.com/wyfcoding/financialtrading/internal/connectivity/application"
	"github.com/wyfcoding/financialtrading/internal/connectivity/infrastructure/client"
	grpc_server "github.com/wyfcoding/financialtrading/internal/connectivity/interfaces/grpc"
	"github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/connectivity/fix"
	"github.com/wyfcoding/pkg/logging"
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
	logCfg := &logging.Config{
		Service: cfg.Server.Name,
		Level:   cfg.Log.Level,
	}
	logger := logging.NewFromConfig(logCfg)
	slog.SetDefault(logger.Logger)

	// 3. Metrics
	// 3. Metrics
	_ = metrics.NewMetrics(cfg.Server.Name)

	// 4. FIX Engine Initialization
	sessionMgr := fix.NewSessionManager()
	// 模拟预置一个会话
	sessionMgr.AddSession(fix.NewSession("INST_001", "TRANS_GATEWAY", "INST_CLIENT"))

	// 5. Application Services
	execAddr := cfg.GetGRPCAddr("execution")
	if execAddr == "" {
		execAddr = "localhost:9081" // Fallback
	}
	execCli, _ := client.NewExecutionClient(execAddr)

	appService := application.NewConnectivityService(sessionMgr, execCli)

	// 6. gRPC Server
	grpcSrv := grpc.NewServer()
	grpc_server.NewHandler(grpcSrv, appService)
	reflection.Register(grpcSrv)

	// 7. HTTP Server (System info)
	r := gin.New()
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "UP"}) })

	// 8. Lifecycle Management
	g, ctx := errgroup.WithContext(context.Background())

	// gRPC
	g.Go(func() error {
		addr := fmt.Sprintf(":%d", cfg.Server.GRPC.Port)
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		slog.Info("gRPC server starting", "addr", addr)
		return grpcSrv.Serve(lis)
	})

	// HTTP
	g.Go(func() error {
		addr := fmt.Sprintf(":%d", cfg.Server.HTTP.Port)
		server := &http.Server{Addr: addr, Handler: r}
		slog.Info("HTTP server starting", "addr", addr)
		return server.ListenAndServe()
	})

	// 模拟 FIX TCP Server (及解析逻辑)
	g.Go(func() error {
		// 这里在实际生产中应该是一个 TCP Server，监听 9800 端口并使用 fix.Parse 解析报文
		slog.Info("FIX Gateway simulator starting", "port", 9800)
		<-ctx.Done()
		return nil
	})

	// Graceful Shutdown
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
