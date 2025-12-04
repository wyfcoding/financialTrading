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
	pb "github.com/wyfcoding/financialTrading/go-api/pricing"
	"github.com/wyfcoding/financialTrading/internal/pricing/application"
	"github.com/wyfcoding/financialTrading/internal/pricing/infrastructure/client"
	grpchandler "github.com/wyfcoding/financialTrading/internal/pricing/interfaces/grpc"
	httphandler "github.com/wyfcoding/financialTrading/internal/pricing/interfaces/http"
	"github.com/wyfcoding/financialTrading/pkg/cache"
	"github.com/wyfcoding/financialTrading/pkg/config"
	"github.com/wyfcoding/financialTrading/pkg/grpcclient"
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
	// 加载配置
	configPath := "configs/pricing/config.toml"
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. 初始化日志
	loggerCfg := logger.Config{ // 详细的日志初始化配置
		ServiceName: cfg.ServiceName,
		Level:       cfg.Logger.Level,
		Format:      cfg.Logger.Format,
		Output:      cfg.Logger.Output,
		FilePath:    cfg.Logger.FilePath,
		MaxSize:     cfg.Logger.MaxSize,
		MaxBackups:  cfg.Logger.MaxBackups,
		MaxAge:      cfg.Logger.MaxAge,
		Compress:    cfg.Logger.Compress,
		WithCaller:  cfg.Logger.WithCaller,
	}
	if err := logger.Init(loggerCfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	log := logger.WithModule("main")

	log.InfoContext(ctx, "Starting PricingService", "version", cfg.Version) // 访问 cfg.Version

	// 3. 初始化追踪
	if cfg.Tracing.Enabled {
		shutdown, err := trace.InitTracer(cfg.ServiceName, cfg.Tracing.CollectorEndpoint)
		if err != nil {
			log.ErrorContext(ctx, "Failed to initialize tracer", "error", err)
		} else {
			defer func() {
				if err := shutdown(context.Background()); err != nil {
					log.ErrorContext(ctx, "Failed to shutdown tracer", "error", err)
				}
			}()
			log.InfoContext(ctx, "Tracer initialized", "endpoint", cfg.Tracing.CollectorEndpoint)
		}
	}

	// 4. 初始化 Redis
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
		log.ErrorContext(ctx, "Failed to initialize Redis", "error", err)
		os.Exit(1)
	}
	defer redisCache.Close()

	// 5. 初始化限流器
	rateLimiter := ratelimit.NewRedisRateLimiter(redisCache.GetClient())

	// 6. 初始化层级依赖
	// 基础设施层
	// PricingService 主要是计算服务，可能不需要数据库，或者只需要读取市场数据
	marketDataClientCfg := grpcclient.ClientConfig{
		Target:          cfg.Services["market-data"].Address,
		ConnTimeout:     5,
		RequestTimeout:  5,
		MaxRetries:      3,
		RetryDelay:      100,
		EnableKeepalive: true,
	}
	marketDataClient, err := client.NewMarketDataClient(marketDataClientCfg)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create market data client", "error", err)
		os.Exit(1)
	}

	// 应用层
	svc := application.NewPricingService(marketDataClient) // 将 pricingApp 重命名为 svc

	// 7. 创建 HTTP 服务器
	httpServer := createHTTPServer(cfg, svc, rateLimiter)

	// 8. 创建 gRPC 服务器
	grpcServer := createGRPCServer(cfg, svc)

	// 9. 启动 HTTP 服务器
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port)
		log.InfoContext(ctx, "Starting HTTP server", "addr", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.ErrorContext(ctx, "HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// 10. 启动 gRPC 服务器
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.GRPC.Host, cfg.GRPC.Port)
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			log.ErrorContext(ctx, "Failed to listen on gRPC address", "error", err)
			os.Exit(1)
		}
		log.InfoContext(ctx, "Starting gRPC server", "addr", addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.ErrorContext(ctx, "gRPC server error", "error", err)
			os.Exit(1)
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.InfoContext(ctx, "Shutting down PricingService")

	// 关闭 HTTP 服务器
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.ErrorContext(ctx, "HTTP server shutdown error", "error", err)
	}

	// 关闭 gRPC 服务器
	grpcServer.GracefulStop()
	log.InfoContext(ctx, "Server exited")
}

// createHTTPServer 创建 HTTP 服务器
func createHTTPServer(cfg *config.Config, app *application.PricingService, rateLimiter ratelimit.RateLimiter) *http.Server {
	router := gin.Default()

	// 添加中间件
	router.Use(otelgin.Middleware(cfg.ServiceName))
	router.Use(middleware.GinLoggingMiddleware())
	router.Use(middleware.GinRecoveryMiddleware())
	router.Use(middleware.GinCORSMiddleware())
	router.Use(middleware.RateLimitMiddleware(rateLimiter, cfg.RateLimit))

	// 注册路由
	httpHandler := httphandler.NewPricingHandler(app)
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
func createGRPCServer(cfg *config.Config, app *application.PricingService) *grpc.Server {
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
	pb.RegisterPricingServiceServer(server, handler)
	reflection.Register(server)

	return server
}
