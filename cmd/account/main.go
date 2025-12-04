// AccountService 主程序
// 功能：提供账户管理服务
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
	pb "github.com/wyfcoding/financialTrading/go-api/account"
	"github.com/wyfcoding/financialTrading/internal/account/application"
	"github.com/wyfcoding/financialTrading/internal/account/infrastructure/repository"
	grpchandler "github.com/wyfcoding/financialTrading/internal/account/interfaces/grpc"
	httphandler "github.com/wyfcoding/financialTrading/internal/account/interfaces/http"
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

// main 是 AccountService 服务的入口函数。
// 它负责整个服务的初始化、启动和优雅关停。
//
// 初始化流程包括：
// 1. 加载配置文件 (TOML 格式)。
// 2. 初始化日志系统 (slog)。
// 3. 初始化分布式追踪系统 (OpenTelemetry)。
// 4. 初始化数据库连接 (GORM)。
// 5. 初始化 Redis 缓存。
// 6. 初始化限流器。
// 7. 依赖注入：创建 repository 和 application service。
// 8. 初始化 Prometheus 指标。
// 9. 创建并启动 HTTP 和 gRPC 服务器。
// 10. 监听系统信号，实现优雅关停。
func main() {
	// 步骤 1: 加载配置
	// 从指定路径加载 TOML 格式的配置文件，并支持环境变量覆盖。
	configPath := "configs/account/config.toml"
	cfg, err := config.Load(configPath)
	if err != nil {
		// 在日志系统初始化前，使用标准错误输出。
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 步骤 2: 初始化日志
	// 配置并初始化全局的结构化日志系统 (slog)。
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
	log := logger.WithModule("main") // 获取带有 "main" 模块标识的 logger

	log.InfoContext(ctx, "Starting AccountService",
		"version", cfg.Version,
		"environment", cfg.Environment,
	)

	// 步骤 3: 初始化追踪
	// 如果在配置中启用了追踪，则初始化 OpenTelemetry tracer。
	if cfg.Tracing.Enabled {
		shutdown, err := trace.InitTracer(cfg.ServiceName, cfg.Tracing.CollectorEndpoint)
		if err != nil {
			log.ErrorContext(ctx, "Failed to initialize tracer", "error", err)
		} else {
			// 注册一个延迟函数，在服务关闭时优雅地关闭 tracer。
			defer func() {
				if err := shutdown(context.Background()); err != nil {
					log.ErrorContext(ctx, "Failed to shutdown tracer", "error", err)
				}
			}()
			log.InfoContext(ctx, "Tracer initialized", "endpoint", cfg.Tracing.CollectorEndpoint)
		}
	}

	// 步骤 4: 初始化数据库
	// 根据配置初始化 GORM 数据库连接池。
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
	defer database.Close() // 确保在 main 函数退出时关闭数据库连接。

	// 步骤 5: 初始化 Redis
	// 根据配置创建 Redis 客户端和连接池。
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
	defer redisCache.Close() // 确保在 main 函数退出时关闭 Redis 连接。

	// 步骤 6: 初始化限流器
	// 创建一个基于 Redis 的分布式限流器。
	rateLimiter := ratelimit.NewRedisRateLimiter(redisCache.GetClient())

	// 步骤 7: 初始化仓储层 (DDD - Infrastructure)
	// 创建仓储实现，它们是数据访问的抽象。
	accountRepo := repository.NewAccountRepository(database)
	transactionRepo := repository.NewTransactionRepository(database)

	// 步骤 8: 初始化应用服务层 (DDD - Application)
	// 将仓储注入到应用服务中，封装核心业务逻辑。
	accountAppService := application.NewAccountApplicationService(accountRepo, transactionRepo)

	// 步骤 9: 初始化指标
	// 创建 Prometheus 指标实例并注册。
	metricsInstance := metrics.New(cfg.ServiceName)
	if err := metricsInstance.Register(); err != nil {
		log.ErrorContext(ctx, "Failed to register metrics", "error", err)
		os.Exit(1)
	}
	// 启动一个独立的 HTTP 服务器用于暴露 /metrics 端点。
	if err := metrics.StartHTTPServer(cfg.Metrics.Port, cfg.Metrics.Path); err != nil {
		log.ErrorContext(ctx, "Failed to start metrics HTTP server", "error", err)
		os.Exit(1)
	}

	// 步骤 10: 创建 HTTP 服务器 (Gin)
	httpServer := createHTTPServer(cfg, accountAppService, rateLimiter)

	// 步骤 11: 创建 gRPC 服务器
	grpcServer := createGRPCServer(cfg, accountAppService)

	// 步骤 12: 在独立的 goroutine 中启动 HTTP 服务器
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port)
		log.InfoContext(ctx, "Starting HTTP server", "addr", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.ErrorContext(ctx, "HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// 步骤 13: 在独立的 goroutine 中启动 gRPC 服务器
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
	// 创建一个 channel 来接收操作系统信号 (SIGINT, SIGTERM)。
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	// 阻塞，直到接收到关闭信号。
	<-sigChan

	log.InfoContext(ctx, "Shutting down AccountService")

	// 创建一个有超时的 context 用于关停服务器。
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 优雅地关闭 HTTP 服务器。
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.ErrorContext(ctx, "HTTP server shutdown error", "error", err)
	}

	// 优雅地关闭 gRPC 服务器。
	grpcServer.GracefulStop()

	log.InfoContext(ctx, "AccountService stopped")
}

// createHTTPServer 负责创建和配置 Gin HTTP 服务器。
// 它整合了所有必要的中间件和路由。
func createHTTPServer(cfg *config.Config, accountAppService *application.AccountApplicationService, rateLimiter ratelimit.RateLimiter) *http.Server {
	// 使用默认配置创建 Gin 引擎。
	router := gin.Default()

	// 添加一系列中间件，注意顺序：
	// 1. OpenTelemetry 中间件：必须放在最前面，以便为每个请求创建 Trace Span，并将 Trace ID 注入 context。
	router.Use(otelgin.Middleware(cfg.ServiceName))
	// 2. 日志中间件：记录每个请求的详细信息，包括 Trace ID。
	router.Use(middleware.GinLoggingMiddleware())
	// 3. 恢复中间件：捕获 panic，防止服务器崩溃，并返回 500 错误。
	router.Use(middleware.GinRecoveryMiddleware())
	// 4. CORS 中间件：处理跨域请求。
	router.Use(middleware.GinCORSMiddleware())
	// 5. 限流中间件：基于 Redis 保护 API 免受过多请求。
	router.Use(middleware.RateLimitMiddleware(rateLimiter, cfg.RateLimit))

	// 注册应用的 HTTP 路由。
	httpHandler := httphandler.NewAccountHandler(accountAppService)
	httpHandler.RegisterRoutes(router)

	// 添加健康检查端点，用于 K8s liveness/readiness probes。
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   cfg.ServiceName,
			"timestamp": time.Now().Unix(),
		})
	})

	// 创建 http.Server 实例，并配置相关超时。
	return &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.HTTP.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.HTTP.WriteTimeout) * time.Second,
	}
}

// createGRPCServer 负责创建和配置 gRPC 服务器。
// 它整合了所有必要的拦截器 (Interceptor) 和服务实现。
func createGRPCServer(cfg *config.Config, accountAppService *application.AccountApplicationService) *grpc.Server {
	// 创建 gRPC 服务器的一系列选项。
	opts := []grpc.ServerOption{
		// 添加 OpenTelemetry StatsHandler，用于自动追踪 gRPC 调用。
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		// 使用链式 Unary 拦截器，请求将按顺序通过它们：
		// 1. 日志拦截器：记录 gRPC 请求信息。
		// 2. 恢复拦截器：捕获 gRPC handler 中的 panic。
		grpc.ChainUnaryInterceptor(
			middleware.GRPCLoggingInterceptor(),
			middleware.GRPCRecoveryInterceptor(),
		),
		// 设置服务器最大并发流数，防止资源耗尽。
		grpc.MaxConcurrentStreams(uint32(cfg.GRPC.MaxConcurrentStreams)),
	}

	// 使用上述选项创建 gRPC 服务器实例。
	server := grpc.NewServer(opts...)

	// 注册 gRPC 服务实现。
	handler := grpchandler.NewGRPCHandler(accountAppService)
	pb.RegisterAccountServiceServer(server, handler)

	return server
}
