// OrderService 主程序
// 功能：提供订单管理服务，包括创建、取消、修改、查询订单等
// 架构：基于 DDD + 微服务 + gRPC + Kafka
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
	pb "github.com/wyfcoding/financialTrading/go-api/order"
	"github.com/wyfcoding/financialTrading/internal/order/application"
	"github.com/wyfcoding/financialTrading/internal/order/infrastructure/repository"
	grpchandler "github.com/wyfcoding/financialTrading/internal/order/interfaces/grpc"
	httphandler "github.com/wyfcoding/financialTrading/internal/order/interfaces/http"
	"github.com/wyfcoding/financialTrading/pkg/cache"
	"github.com/wyfcoding/financialTrading/pkg/config"
	"github.com/wyfcoding/financialTrading/pkg/db"
	"github.com/wyfcoding/financialTrading/pkg/logger"
	"github.com/wyfcoding/financialTrading/pkg/metrics"
	"github.com/wyfcoding/financialTrading/pkg/middleware"
	"github.com/wyfcoding/financialTrading/pkg/ratelimit"
	"github.com/wyfcoding/financialTrading/pkg/trace"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

func main() {
	// 1. 加载配置
	configPath := "configs/order/config.toml"
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
	logger.Info(ctx, "Starting OrderService",
		"service", cfg.ServiceName,
		"version", cfg.Version,
		"environment", cfg.Environment,
	)

	// 3. 初始化追踪
	if cfg.Tracing.Enabled {
		shutdown, err := trace.InitTracer(cfg.ServiceName, cfg.Tracing.CollectorEndpoint)
		if err != nil {
			logger.Error(ctx, "Failed to initialize tracer", "error", err)
		} else {
			defer func() {
				if err := shutdown(context.Background()); err != nil {
					logger.Error(ctx, "Failed to shutdown tracer", "error", err)
				}
			}()
			logger.Info(ctx, "Tracer initialized", "endpoint", cfg.Tracing.CollectorEndpoint)
		}
	}

	// 4. 初始化数据库
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

	// 5. 初始化 Redis
	redisCfg := cache.Config{
		Host:         cfg.Redis.Host,
		Port:         cfg.Redis.Port,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		MaxPoolSize:  cfg.Redis.MaxPoolSize,
		ConnTimeout:  cfg.Redis.ConnTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	}
	redisCache, err := cache.New(redisCfg)
	if err != nil {
		logger.Fatal(ctx, "Failed to initialize Redis", "error", err)
	}
	defer redisCache.Close()

	// 6. 初始化限流器
	rateLimiter := ratelimit.NewRedisRateLimiter(redisCache.GetClient())

	// 7. 初始化仓储
	orderRepo := repository.NewOrderRepository(database)

	// 8. 初始化应用服务
	orderAppService := application.NewOrderApplicationService(orderRepo)

	// 9. 初始化指标
	metricsInstance := metrics.New(cfg.ServiceName)
	if err := metricsInstance.Register(); err != nil {
		logger.Fatal(ctx, "Failed to register metrics", "error", err)
	}
	if err := metrics.StartHTTPServer(cfg.Metrics.Port, cfg.Metrics.Path); err != nil {
		logger.Fatal(ctx, "Failed to start metrics HTTP server", "error", err)
	}

	// 10. 创建 HTTP 服务器
	httpServer := createHTTPServer(cfg, orderAppService, rateLimiter)

	// 11. 创建 gRPC 服务器
	grpcServer := createGRPCServer(cfg, orderAppService)

	// 12. 启动 HTTP 服务器
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port)
		logger.Info(ctx, "Starting HTTP server", "addr", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal(ctx, "HTTP server error", "error", err)
		}
	}()

	// 13. 启动 gRPC 服务器
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

	// 14. 优雅关停
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info(ctx, "Shutting down OrderService")

	// 关闭 HTTP 服务器
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error(ctx, "HTTP server shutdown error", "error", err)
	}

	// 关闭 gRPC 服务器
	grpcServer.GracefulStop()

	logger.Info(ctx, "OrderService stopped")
}

// createHTTPServer 创建 HTTP 服务器
func createHTTPServer(cfg *config.Config, orderAppService *application.OrderApplicationService, rateLimiter ratelimit.RateLimiter) *http.Server {
	router := gin.Default()

	// 添加中间件
	router.Use(otelgin.Middleware(cfg.ServiceName))
	router.Use(middleware.GinLoggingMiddleware())
	router.Use(middleware.GinRecoveryMiddleware())
	router.Use(middleware.GinCORSMiddleware())
	router.Use(middleware.RateLimitMiddleware(rateLimiter, cfg.RateLimit))

	// 注册路由
	httpHandler := httphandler.NewOrderHandler(orderAppService)
	httpHandler.RegisterRoutes(router)

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
func createGRPCServer(cfg *config.Config, orderAppService *application.OrderApplicationService) *grpc.Server {
	// 创建 gRPC 服务器选项
	opts := []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			middleware.GRPCLoggingInterceptor(),
			middleware.GRPCRecoveryInterceptor(),
		),
		grpc.MaxConcurrentStreams(uint32(cfg.GRPC.MaxConcurrentStreams)),
	}

	server := grpc.NewServer(opts...)

	// 注册服务
	handler := grpchandler.NewGRPCHandler(orderAppService)
	pb.RegisterOrderServiceServer(server, handler)

	return server
}
