// ClearingService 主程序
// 功能：提供清算服务，包括交易清算、日终清算、清算状态管理
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
	pb "github.com/wyfcoding/financialTrading/go-api/clearing"
	"github.com/wyfcoding/financialTrading/internal/clearing/application"
	"github.com/wyfcoding/financialTrading/internal/clearing/infrastructure/repository"
	grpchandler "github.com/wyfcoding/financialTrading/internal/clearing/interfaces/grpc"
	httphandler "github.com/wyfcoding/financialTrading/internal/clearing/interfaces/http"
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

// main 是 ClearingService 服务的入口函数。
// 它遵循与其他微服务类似的结构，负责整个服务的初始化、启动和优雅关停。
//
// 初始化流程包括：
// 1. 加载配置文件。
// 2. 初始化日志、追踪、数据库、缓存等基础设施。
// 3. 依赖注入：创建 repository 和 application service。
// 4. 初始化 Prometheus 指标。
// 5. 创建并以 goroutine 方式启动 HTTP 和 gRPC 服务器。
// 6. 监听操作系统信号，实现平滑、无中断的服务关停。
func main() {
	// 步骤 1: 加载配置
	// 从 `configs/clearing/config.toml` 加载服务配置。
	configPath := "configs/clearing/config.toml"
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 步骤 2: 初始化日志
	// 根据配置初始化全局 slog 日志记录器。
	loggerCfg := logger.Config{
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

	log.InfoContext(ctx, "Starting ClearingService",
		"version", cfg.Version,
		"environment", cfg.Environment,
	)

	// 步骤 3: 初始化追踪
	// 如果配置中启用，则初始化 OpenTelemetry 分布式追踪。
	if cfg.Tracing.Enabled {
		shutdown, err := trace.InitTracer(cfg.ServiceName, cfg.Tracing.CollectorEndpoint)
		if err != nil {
			log.ErrorContext(ctx, "Failed to initialize tracer", "error", err)
		} else {
			// 注册延迟调用，确保在服务退出时刷新并关闭 tracer provider。
			defer func() {
				if err := shutdown(context.Background()); err != nil {
					log.ErrorContext(ctx, "Failed to shutdown tracer", "error", err)
				}
			}()
			log.InfoContext(ctx, "Tracer initialized", "endpoint", cfg.Tracing.CollectorEndpoint)
		}
	}

	// 步骤 4: 初始化数据库
	// 初始化 GORM，用于连接和操作数据库。
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
		log.ErrorContext(ctx, "Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// 步骤 5: 初始化 Redis
	// 初始化 Redis 客户端，用于缓存和限流。
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

	// 步骤 6: 初始化限流器
	// 创建基于 Redis 的分布式限流器实例。
	rateLimiter := ratelimit.NewRedisRateLimiter(redisCache.GetClient())

	// 步骤 7: 初始化仓储 (DDD - Infrastructure)
	// 创建清算相关的仓储实现。
	settlementRepo := repository.NewSettlementRepository(database)
	eodRepo := repository.NewEODClearingRepository(database)

	// 步骤 8: 初始化应用服务 (DDD - Application)
	// 将仓储注入应用服务，封装核心清算逻辑。
	clearingAppService := application.NewClearingApplicationService(settlementRepo, eodRepo)

	// 步骤 9: 初始化指标
	// 初始化并启动 Prometheus 指标服务。
	metricsInstance := metrics.New(cfg.ServiceName)
	if err := metricsInstance.Register(); err != nil {
		log.ErrorContext(ctx, "Failed to register metrics", "error", err)
		os.Exit(1)
	}
	if err := metrics.StartHTTPServer(cfg.Metrics.Port, cfg.Metrics.Path); err != nil {
		log.ErrorContext(ctx, "Failed to start metrics HTTP server", "error", err)
		os.Exit(1)
	}

	// 步骤 10: 创建 HTTP 服务器
	httpServer := createHTTPServer(cfg, clearingAppService, rateLimiter)

	// 步骤 11: 创建 gRPC 服务器
	grpcServer := createGRPCServer(cfg, clearingAppService)

	// 步骤 12: 启动 HTTP 服务器
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port)
		log.InfoContext(ctx, "Starting HTTP server", "addr", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.ErrorContext(ctx, "HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// 步骤 13: 启动 gRPC 服务器
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.GRPC.Host, cfg.GRPC.Port)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			log.ErrorContext(ctx, "Failed to listen on gRPC address", "error", err)
			os.Exit(1)
		}
		log.InfoContext(ctx, "Starting gRPC server", "addr", addr)
		if err := grpcServer.Serve(listener); err != nil {
			log.ErrorContext(ctx, "gRPC server error", "error", err)
			os.Exit(1)
		}
	}()

	// 步骤 14: 优雅关停
	// 监听 SIGINT 和 SIGTERM 信号以触发关停流程。
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.InfoContext(ctx, "Shutting down ClearingService")

	// 设置一个带超时的 context，用于服务器的优雅关闭。
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// 优雅关闭 HTTP 服务器，等待正在处理的请求完成。
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.ErrorContext(ctx, "HTTP server shutdown error", "error", err)
	}

	// 优雅关闭 gRPC 服务器，停止接受新连接并等待现有 RPC 完成。
	grpcServer.GracefulStop()

	log.InfoContext(ctx, "ClearingService stopped")
}

// createHTTPServer 负责创建和配置 Gin HTTP 服务器。
// 它将应用服务和各类中间件组装在一起。
func createHTTPServer(cfg *config.Config, clearingAppService *application.ClearingApplicationService, rateLimiter ratelimit.RateLimiter) *http.Server {
	router := gin.Default()

	// 注册通用中间件，执行顺序与注册顺序一致。
	router.Use(otelgin.Middleware(cfg.ServiceName))                        // OpenTelemetry 追踪
	router.Use(middleware.GinLoggingMiddleware())                          // 结构化日志
	router.Use(middleware.GinRecoveryMiddleware())                         // Panic 恢复
	router.Use(middleware.GinCORSMiddleware())                             // 跨域资源共享
	router.Use(middleware.RateLimitMiddleware(rateLimiter, cfg.RateLimit)) // API 限流

	// 注册该服务的业务路由。
	httpHandler := httphandler.NewClearingHandler(clearingAppService)
	httpHandler.RegisterRoutes(router)

	// 健康检查端点，对 K8s 等编排工具至关重要。
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   cfg.ServiceName,
			"timestamp": time.Now().Unix(),
		})
	})

	// 返回配置好的 http.Server 实例。
	return &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.HTTP.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.HTTP.WriteTimeout) * time.Second,
	}
}

// createGRPCServer 负责创建和配置 gRPC 服务器。
// 它将应用服务和 gRPC 拦截器组装在一起。
func createGRPCServer(cfg *config.Config, clearingAppService *application.ClearingApplicationService) *grpc.Server {
	// 配置 gRPC 服务器选项。
	opts := []grpc.ServerOption{
		// 添加 OpenTelemetry 拦截器，用于追踪。
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		// 链式一元拦截器，按顺序执行。
		grpc.ChainUnaryInterceptor(
			middleware.GRPCLoggingInterceptor(),  // 日志记录
			middleware.GRPCRecoveryInterceptor(), // Panic 恢复
		),
		// 设置服务器端的最大并发流数。
		grpc.MaxConcurrentStreams(uint32(cfg.GRPC.MaxConcurrentStreams)),
	}

	// 创建 gRPC 服务器。
	server := grpc.NewServer(opts...)

	// 创建并注册 gRPC 服务处理器。
	handler := grpchandler.NewGRPCHandler(clearingAppService)
	pb.RegisterClearingServiceServer(server, handler)

	return server
}
