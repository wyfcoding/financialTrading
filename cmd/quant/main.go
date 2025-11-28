package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/wyfcoding/financialTrading/go-api/quant"
	"github.com/wyfcoding/financialTrading/internal/quant/application"
	"github.com/wyfcoding/financialTrading/internal/quant/infrastructure"
	grpchandler "github.com/wyfcoding/financialTrading/internal/quant/interfaces/grpc"
	httphandler "github.com/wyfcoding/financialTrading/internal/quant/interfaces/http"
	"github.com/wyfcoding/financialTrading/pkg/cache"
	"github.com/wyfcoding/financialTrading/pkg/config"
	"github.com/wyfcoding/financialTrading/pkg/db"
	"github.com/wyfcoding/financialTrading/pkg/logger"
	"github.com/wyfcoding/financialTrading/pkg/middleware"
	"github.com/wyfcoding/financialTrading/pkg/ratelimit"
	"github.com/wyfcoding/financialTrading/pkg/trace"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 1. 加载配置
	configPath := "configs/quant/config.toml"
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志
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
	logger.Info(ctx, "Starting QuantService", "version", cfg.Version)

	// 初始化追踪
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

	// 初始化数据库
	dbConfig := db.Config{
		Driver:             cfg.Database.Driver,
		DSN:                cfg.Database.DSN,
		MaxOpenConns:       cfg.Database.MaxOpenConns,
		MaxIdleConns:       cfg.Database.MaxIdleConns,
		ConnMaxLifetime:    cfg.Database.ConnMaxLifetime,
		LogEnabled:         cfg.Database.LogEnabled,
		SlowQueryThreshold: cfg.Database.SlowQueryThreshold,
	}
	gormDB, err := db.Init(dbConfig)
	if err != nil {
		logger.Fatal(ctx, "Failed to connect to database", "error", err)
	}

	// 自动迁移
	if err := gormDB.AutoMigrate(&infrastructure.StrategyModel{}, &infrastructure.BacktestResultModel{}); err != nil {
		logger.Fatal(ctx, "Failed to migrate database", "error", err)
	}

	// 初始化 Redis
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

	// 初始化限流器
	rateLimiter := ratelimit.NewRedisRateLimiter(redisCache.GetClient())

	// 初始化依赖
	marketDataClient := infrastructure.NewMockMarketDataClient()
	strategyRepo := infrastructure.NewStrategyRepository(gormDB.DB)
	backtestRepo := infrastructure.NewBacktestResultRepository(gormDB.DB)
	svc := application.NewQuantService(strategyRepo, backtestRepo, marketDataClient)

	// 6. 创建 HTTP 服务器
	httpServer := createHTTPServer(cfg, svc, rateLimiter)

	// 7. 创建 gRPC 服务器
	grpcServer := createGRPCServer(cfg, svc)

	// 8. 启动 HTTP 服务器
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port)
		logger.Info(ctx, "Starting HTTP server", "addr", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal(ctx, "HTTP server error", "error", err)
		}
	}()

	// 9. 启动 gRPC 服务器
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.GRPC.Host, cfg.GRPC.Port)
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			logger.Fatal(ctx, "Failed to listen on gRPC address", "error", err)
		}
		logger.Info(ctx, "Starting gRPC server", "addr", addr)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal(ctx, "gRPC server error", "error", err)
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info(ctx, "Shutting down QuantService")

	// 关闭 HTTP 服务器
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error(ctx, "HTTP server shutdown error", "error", err)
	}

	// 关闭 gRPC 服务器
	grpcServer.GracefulStop()
	logger.Info(ctx, "Server exited")
}

// createHTTPServer 创建 HTTP 服务器
func createHTTPServer(cfg *config.Config, app *application.QuantService, rateLimiter ratelimit.RateLimiter) *http.Server {
	router := gin.Default()

	// 添加中间件
	router.Use(otelgin.Middleware(cfg.ServiceName))
	router.Use(middleware.GinLoggingMiddleware())
	router.Use(middleware.GinRecoveryMiddleware())
	router.Use(middleware.GinCORSMiddleware())
	router.Use(middleware.RateLimitMiddleware(rateLimiter, cfg.RateLimit))

	// 注册路由
	httpHandler := httphandler.NewQuantHandler(app)
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
func createGRPCServer(cfg *config.Config, app *application.QuantService) *grpc.Server {
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
	handler := grpchandler.NewGRPCHandler(app)
	pb.RegisterQuantServiceServer(server, handler)
	reflection.Register(server)

	return server
}
