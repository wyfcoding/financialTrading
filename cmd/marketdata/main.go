package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/wyfcoding/pkg/database"
	"github.com/wyfcoding/pkg/response"

	"github.com/gin-gonic/gin"
	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/grpc"

	pb "github.com/wyfcoding/financialtrading/goapi/marketdata/v1"
	"github.com/wyfcoding/financialtrading/internal/marketdata/application"
	"github.com/wyfcoding/financialtrading/internal/marketdata/infrastructure/persistence/mysql"
	marketdatagrpc "github.com/wyfcoding/financialtrading/internal/marketdata/interfaces/grpc"
	marketdatahttp "github.com/wyfcoding/financialtrading/internal/marketdata/interfaces/http"
	"github.com/wyfcoding/pkg/app"
	"github.com/wyfcoding/pkg/cache"
	configpkg "github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/grpcclient"
	"github.com/wyfcoding/pkg/idempotency"
	"github.com/wyfcoding/pkg/limiter"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/messagequeue/kafka"
	"github.com/wyfcoding/pkg/metrics"
	"github.com/wyfcoding/pkg/middleware"
	"github.com/wyfcoding/pkg/redis"
	"github.com/wyfcoding/pkg/server"
)

// BootstrapName 服务唯一标识
const BootstrapName = "marketdata"

// IdempotencyPrefix 幂等性 Redis 键前缀
const IdempotencyPrefix = "marketdata:idem"

// Config 服务扩展配置
type Config struct {
	configpkg.Config `mapstructure:",squash"`
}

// AppContext 应用上下文 (包含对外服务实例与依赖)
type AppContext struct {
	Config      *Config
	MarketData  *application.MarketDataService
	Clients     *ServiceClients
	Handler     *marketdatahttp.Handler
	Metrics     *metrics.Metrics
	Limiter     limiter.Limiter
	Idempotency idempotency.Manager
	Consumer    *kafka.Consumer
	WS          *server.WSManager
}

// ServiceClients 下游微服务客户端集合
type ServiceClients struct {
	// 目前 MarketData 服务无下游强依赖
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
	pb.RegisterMarketDataServiceServer(s, marketdatagrpc.NewMarketDataHandler(ctx.MarketData))
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

	// WebSocket 入口
	e.GET("/ws", func(c *gin.Context) {
		ctx.WS.ServeHTTP(c.Writer, c.Request)
	})

	// 指标暴露
	if ctx.Config.Metrics.Enabled {
		e.GET(ctx.Config.Metrics.Path, gin.WrapH(ctx.Metrics.Handler()))
	}

	// 全局限流中间件
	e.Use(middleware.RateLimitWithLimiter(ctx.Limiter))

	// 业务 API 路由 v1
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

	// 3. 初始化 WebSocket 管理器
	wsManager := server.NewWSManager(logger.Logger)
	go wsManager.Run(context.Background())

	// --- 3.1 分布式行情同步中心 (跨节点广播) ---
	distBroadcaster := &distributedBroadcaster{
		local: wsManager,
		redis: redisCache.GetClient(),
	}

	// 订阅 Redis 行情总线，接收来自其他实例的消息
	go func() {
		pubsub := redisCache.GetClient().Subscribe(context.Background(), "marketdata.broadcast")
		defer pubsub.Close()
		for {
			msg, err := pubsub.ReceiveMessage(context.Background())
			if err != nil {
				bootLog.Error("redis cluster sync error", "error", err)
				time.Sleep(time.Second)
				continue
			}
			var envelope struct {
				T string `json:"t"`
				P []byte `json:"p"`
			}
			if err := json.Unmarshal([]byte(msg.Payload), &envelope); err == nil {
				// 仅分发给连接在本机的客户端
				wsManager.BroadcastRaw(envelope.T, envelope.P)
			}
		}
	}()

	// 4. 初始化治理组件
	rateLimiter := limiter.NewRedisLimiter(redisCache.GetClient(), c.RateLimit.Rate, time.Second)
	idemManager := idempotency.NewRedisManager(redisCache.GetClient(), IdempotencyPrefix)

	// 5. 初始化下游微服务客户端
	clients := &ServiceClients{}
	clientCleanup, err := grpcclient.InitClients(c.Services, m, c.CircuitBreaker, clients)
	if err != nil {
		redisCache.Close()
		if sqlDB, err := db.RawDB().DB(); err == nil {
			sqlDB.Close()
		}
		return nil, nil, err
	}

	// 6. DDD 分层装配
	bootLog.Info("assembling services...")
	quoteRepo := mysql.NewQuoteRepository(db.RawDB())
	klineRepo := mysql.NewKlineRepository(db.RawDB())
	tradeRepo := mysql.NewTradeRepository(db.RawDB())
	orderBookRepo := mysql.NewOrderBookRepository(db.RawDB())

	marketDataService := application.NewMarketDataService(quoteRepo, klineRepo, tradeRepo, orderBookRepo, logger.Logger)
	// 注入支持分布式的广播器
	marketDataService.SetBroadcaster(distBroadcaster)

	// 7. 启动成交事件消费
	consumer := kafka.NewConsumer(c.MessageQueue.Kafka, logger, m)
	consumer.Start(context.Background(), 10, func(ctx context.Context, msg kafkago.Message) error {
		if msg.Topic != "trade.executed" {
			return nil
		}
		var event map[string]any
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return err
		}
		return marketDataService.HandleTradeExecuted(ctx, event)
	})

	handler := marketdatahttp.NewHandler(marketDataService)

	cleanup := func() {
		bootLog.Info("shutting down...")
		if consumer != nil {
			consumer.Close()
		}
		clientCleanup()
		if redisCache != nil {
			redisCache.Close()
		}
		if sqlDB, err := db.RawDB().DB(); err == nil {
			sqlDB.Close()
		}
	}

	return &AppContext{
		Config: c, MarketData: marketDataService, Clients: clients, Handler: handler,
		Metrics: m, Limiter: rateLimiter, Idempotency: idemManager, Consumer: consumer, WS: wsManager,
	}, cleanup, nil
}

// distributedBroadcaster 实现跨节点推送
type distributedBroadcaster struct {
	local *server.WSManager
	redis *redis.Client
}

func (b *distributedBroadcaster) Broadcast(topic string, payload any) {
	data, _ := json.Marshal(payload)
	// 1. 本地广播
	b.local.BroadcastRaw(topic, data)
	// 2. 发送到 Redis 总线供其他节点消费
	go func() {
		syncMsg := struct {
			T string `json:"t"`
			P []byte `json:"p"`
		}{T: topic, P: data}
		msgJSON, _ := json.Marshal(syncMsg)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := b.redis.Publish(ctx, "marketdata.broadcast", string(msgJSON)).Err(); err != nil {
			slog.Warn("failed to publish market data to redis", "error", err, "topic", topic)
		}
	}()
}
