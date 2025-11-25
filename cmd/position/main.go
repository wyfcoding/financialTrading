// PositionService 主程序
// 功能：提供持仓管理服务
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/wyfcoding/financialTrading/go-api/position"
	"github.com/wyfcoding/financialTrading/internal/position/application"
	"github.com/wyfcoding/financialTrading/internal/position/infrastructure/repository"
	"github.com/wyfcoding/financialTrading/internal/position/interfaces"
	"github.com/wyfcoding/financialTrading/pkg/config"
	"github.com/wyfcoding/financialTrading/pkg/db"
	"github.com/wyfcoding/financialTrading/pkg/logger"
	"github.com/wyfcoding/financialTrading/pkg/metrics"
	"github.com/wyfcoding/financialTrading/pkg/middleware"
	"google.golang.org/grpc"
)

func main() {
	// 1. 加载配置
	configPath := "configs/position/config.toml"
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化日志
	loggerCfg := logger.Config{
		Level:      cfg.Logger.Level,
		Format:     cfg.Logger.Format,
		Output:     cfg.Logger.Output,
		FilePath:   cfg.Logger.FilePath,
		MaxSize:    cfg.Logger.MaxSize,
		MaxBackups: cfg.Logger.MaxBackups,
		MaxAge:     cfg.Logger.MaxAge,
		Compress:   cfg.Logger.Compress,
		WithCaller: cfg.Logger.WithCaller,
	}
	if err := logger.Init(loggerCfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	logger.Info(ctx, "Starting PositionService",
		"service", cfg.ServiceName,
		"version", cfg.Version,
		"environment", cfg.Environment,
	)

	// 3. 初始化数据库
	dbCfg := db.Config{
		Driver:             cfg.Database.Driver,
		DSN:                cfg.Database.DSN,
		MaxOpenConns:       cfg.Database.MaxOpenConns,
		MaxIdleConns:       cfg.Database.MaxIdleConns,
		ConnMaxLifetime:    cfg.Database.ConnMaxLifetime,
		LogEnabled:         cfg.Database.LogEnabled,
		SlowQueryThreshold: cfg.Database.SlowQueryThreshold,
	}
	database, err := db.Init(dbCfg)
	if err != nil {
		logger.Fatal(ctx, "Failed to initialize database", "error", err)
	}
	defer database.Close()

	// 4. 初始化仓储
	positionRepo := repository.NewPositionRepository(database)

	// 5. 初始化应用服务
	positionAppService := application.NewPositionApplicationService(positionRepo)

	// 6. 初始化指标
	metricsInstance := metrics.New(cfg.ServiceName)
	if err := metricsInstance.Register(); err != nil {
		logger.Fatal(ctx, "Failed to register metrics", "error", err)
	}
	if err := metrics.StartHTTPServer(cfg.Metrics.Port, cfg.Metrics.Path); err != nil {
		logger.Fatal(ctx, "Failed to start metrics HTTP server", "error", err)
	}

	// 7. 创建 HTTP 服务器
	httpServer := createHTTPServer(cfg)

	// 8. 创建 gRPC 服务器
	grpcServer := createGRPCServer(cfg, positionAppService)

	// 9. 启动 HTTP 服务器
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port)
		logger.Info(ctx, "Starting HTTP server", "addr", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal(ctx, "HTTP server error", "error", err)
		}
	}()

	// 10. 启动 gRPC 服务器
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.GRPC.Host, cfg.GRPC.Port)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			logger.Fatal(ctx, "Failed to listen on gRPC address", "error", err)
		}
		logger.Info(ctx, "Starting gRPC server", "addr", addr)
		if err := grpcServer.Serve(listener); err != nil {
			logger.Fatal(ctx, "gRPC server error", "error", err)
		}
	}()

	// 11. 优雅关停
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info(ctx, "Shutting down PositionService")

	// 关闭 HTTP 服务器
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error(ctx, "HTTP server shutdown error", "error", err)
	}

	// 关闭 gRPC 服务器
	grpcServer.GracefulStop()

	logger.Info(ctx, "PositionService stopped")
}

// createHTTPServer 创建 HTTP 服务器
func createHTTPServer(cfg *config.Config) *http.Server {
	router := gin.Default()

	// 添加中间件
	router.Use(middleware.GinLoggingMiddleware())
	router.Use(middleware.GinRecoveryMiddleware())
	router.Use(middleware.GinCORSMiddleware())

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   cfg.ServiceName,
			"timestamp": time.Now().Unix(),
		})
	})

	return &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.HTTP.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.HTTP.WriteTimeout) * time.Second,
	}
}

// createGRPCServer 创建 gRPC 服务器
func createGRPCServer(cfg *config.Config, positionAppService *application.PositionApplicationService) *grpc.Server {
	// 创建 gRPC 服务器选项
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(middleware.GRPCLoggingInterceptor()),
		grpc.MaxConcurrentStreams(uint32(cfg.GRPC.MaxConcurrentStreams)),
	}

	server := grpc.NewServer(opts...)

	// 注册服务
	handler := interfaces.NewGRPCHandler(positionAppService)
	pb.RegisterPositionServiceServer(server, handler)

	return server
}
