package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wyfcoding/pkg/database"
	"github.com/wyfcoding/pkg/response"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	clearingv1 "github.com/wyfcoding/financialtrading/goapi/clearing/v1"
	pb "github.com/wyfcoding/financialtrading/goapi/matchingengine/v1"
	orderv1 "github.com/wyfcoding/financialtrading/goapi/order/v1"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/application"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/infrastructure/persistence/mysql"
	matchinggrpc "github.com/wyfcoding/financialtrading/internal/matchingengine/interfaces/grpc"
	matchinghttp "github.com/wyfcoding/financialtrading/internal/matchingengine/interfaces/http"
	"github.com/wyfcoding/pkg/app"
	"github.com/wyfcoding/pkg/cache"
	configpkg "github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/grpcclient"
	"github.com/wyfcoding/pkg/idempotency"
	"github.com/wyfcoding/pkg/limiter"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/messagequeue/kafka"
	"github.com/wyfcoding/pkg/messagequeue/outbox"
	"github.com/wyfcoding/pkg/metrics"
	"github.com/wyfcoding/pkg/middleware"
)

// BootstrapName 服务唯一标识
const BootstrapName = "matchingengine"

// IdempotencyPrefix 幂等性 Redis 键前缀
const IdempotencyPrefix = "matchingengine:idem"

// Config 服务扩展配置
type Config struct {
	configpkg.Config `mapstructure:",squash"`
}

// AppContext 应用上下文 (包含对外服务实例与依赖)
type AppContext struct {
	Config      *Config
	Matching    *application.MatchingEngineService
	Clients     *ServiceClients
	Handler     *matchinghttp.MatchingHandler
	Metrics     *metrics.Metrics
	Limiter     limiter.Limiter
	Idempotency idempotency.Manager
	Outbox      *outbox.Processor
}

// ServiceClients 下游微服务客户端集合
type ServiceClients struct {
	ClearingConn *grpc.ClientConn `service:"clearing"`
	OrderConn    *grpc.ClientConn `service:"order"`

	// 具体的客户端接口
	Clearing clearingv1.ClearingServiceClient
}

func main() {
	// 构建并运行服务
	if err := app.NewBuilder(BootstrapName).
		WithConfig(&Config{}).
		WithService(initService).
		WithGRPC(registerGRPC).
		WithGin(registerGin).
		WithGinMiddleware(
			middleware.CORS(), // 跨域处理
			middleware.TimeoutMiddleware(30*time.Second), // 全局超时
		).
		Build().
		Run(); err != nil {
		slog.Error("service bootstrap failed", "error", err)
	}
}

// registerGRPC 注册 gRPC 服务
func registerGRPC(s *grpc.Server, svc any) {
	ctx := svc.(*AppContext)
	pb.RegisterMatchingEngineServiceServer(s, matchinggrpc.NewGRPCHandler(ctx.Matching))
}

// registerGin 注册 HTTP 路由
func registerGin(e *gin.Engine, svc any) {
	ctx := svc.(*AppContext)

	// 根据环境设置 Gin 模式
	if ctx.Config.Server.Environment == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 系统检查接口
	sys := e.Group("/sys")
	{
		sys.GET("/health", func(c *gin.Context) {
			response.SuccessWithRawData(c, gin.H{
				"status":    "UP",
				"service":   BootstrapName,
				"timestamp": time.Now().Unix(),
			})
		})
		sys.GET("/ready", func(c *gin.Context) {
			response.SuccessWithRawData(c, gin.H{"status": "READY"})
		})
	}

	// 指标暴露
	if ctx.Config.Metrics.Enabled {
		e.GET(ctx.Config.Metrics.Path, gin.WrapH(ctx.Metrics.Handler()))
	}

	// 全局限流中间件
	e.Use(middleware.RateLimitWithLimiter(ctx.Limiter))

	// 业务 API 路由 v1 (与 ecommerce 对齐)
	api := e.Group("/api/v1")
	{
		ctx.Handler.RegisterRoutes(api)
	}
}

// initService 初始化服务依赖 (数据库、缓存、客户端、领域层)
func initService(cfg any, m *metrics.Metrics) (any, func(), error) {
	c := cfg.(*Config)
	bootLog := slog.With("module", "bootstrap")
	logger := logging.Default() // 获取全局 Logger

	// 打印脱敏配置
	configpkg.PrintWithMask(c)

	// 1. 初始化数据库 (MySQL)
	db, err := database.NewDB(c.Data.Database, c.CircuitBreaker, logger, m)
	if err != nil {
		return nil, nil, fmt.Errorf("database init error: %w", err)
	}

	// 2. 初始化缓存 (Redis)
	redisCache, err := cache.NewRedisCache(c.Data.Redis, c.CircuitBreaker, logger, m)
	if err != nil {
		if sqlDB, err := db.RawDB().DB(); err == nil {
			sqlDB.Close()
		}
		return nil, nil, fmt.Errorf("redis init error: %w", err)
	}

	// 3. 初始化治理组件 (限流器、幂等管理器)
	rateLimiter := limiter.NewRedisLimiter(redisCache.GetClient(), c.RateLimit.Rate, time.Second)
	idemManager := idempotency.NewRedisManager(redisCache.GetClient(), IdempotencyPrefix)

	// 4. 初始化消息队列与 Outbox
	producer := kafka.NewProducer(c.MessageQueue.Kafka, logger, m)

	outboxMgr := outbox.NewManager(db.RawDB(), logger.Logger)
	outboxProcessor := outbox.NewProcessor(outboxMgr, func(ctx context.Context, topic, key string, payload []byte) error {
		if producer == nil {
			return fmt.Errorf("kafka producer not initialized")
		}
		return producer.PublishToTopic(ctx, topic, []byte(key), payload)
	}, 100, 5*time.Second)
	outboxProcessor.Start()

	// 5. 初始化下游微服务客户端
	clients := &ServiceClients{}
	clientCleanup, err := grpcclient.InitClients(c.Services, m, c.CircuitBreaker, clients)
	if err != nil {
		redisCache.Close()
		if sqlDB, err := db.RawDB().DB(); err == nil {
			sqlDB.Close()
		}
		return nil, nil, fmt.Errorf("grpc clients init error: %w", err)
	}
	// 显式转换 gRPC 客户端
	if clients.ClearingConn != nil {
		clients.Clearing = clearingv1.NewClearingServiceClient(clients.ClearingConn)
	}

	// 6. DDD 分层装配
	bootLog.Info("assembling services with full dependency injection...")

	// 6.1 Infrastructure (Persistence)
	tradeRepo, orderBookRepo := mysql.NewMatchingRepository(db.RawDB())

	// 6.2 Application (Service)
	// 顶级架构：撮合引擎通常是单实例单交易对，此处可以从配置加载，例如 "BTC/USDT"
	defaultSymbol := "BTC/USDT"
	matchingService, err := application.NewMatchingEngineService(defaultSymbol, tradeRepo, orderBookRepo, db.RawDB(), outboxMgr, logger.Logger)
	if err != nil {
		redisCache.Close()
		if sqlDB, err := db.RawDB().DB(); err == nil {
			if cerr := sqlDB.Close(); cerr != nil {
				bootLog.Error("failed to close sql database", "error", cerr)
			}
		}
		return nil, nil, fmt.Errorf("matching service init error: %w", err)
	}
	if clients.Clearing != nil {
		matchingService.SetClearingClient(clients.Clearing)
	}
	if clients.OrderConn != nil {
		matchingService.SetOrderClient(orderv1.NewOrderServiceClient(clients.OrderConn))
	}

	// 6.3 Interface (HTTP Handlers)
	handler := matchinghttp.NewMatchingHandler(matchingService)

	// 定义资源清理函数
	cleanup := func() {
		bootLog.Info("shutting down, releasing resources...")
		outboxProcessor.Stop()
		clientCleanup()
		if producer != nil {
			producer.Close()
		}
		if redisCache != nil {
			if err := redisCache.Close(); err != nil {
				bootLog.Error("failed to close redis cache", "error", err)
			}
		}
		if sqlDB, err := db.RawDB().DB(); err == nil && sqlDB != nil {
			if err := sqlDB.Close(); err != nil {
				bootLog.Error("failed to close sql database", "error", err)
			}
		}
	}

	// 返回应用上下文与清理函数
	return &AppContext{
		Config:      c,
		Matching:    matchingService,
		Clients:     clients,
		Handler:     handler,
		Metrics:     m,
		Limiter:     rateLimiter,
		Idempotency: idemManager,
		Outbox:      outboxProcessor,
	}, cleanup, nil
}
