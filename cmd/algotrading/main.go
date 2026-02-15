package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"github.com/wyfcoding/financialtrading/internal/algotrading/application"
	"github.com/wyfcoding/financialtrading/internal/algotrading/domain"
	"github.com/wyfcoding/financialtrading/internal/algotrading/infrastructure/persistence/mysql"
	"github.com/wyfcoding/financialtrading/internal/algotrading/interfaces"
	"github.com/wyfcoding/pkg/app"
	"github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/database"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/messagequeue/kafka"
	"github.com/wyfcoding/pkg/messagequeue/outbox"
	"github.com/wyfcoding/pkg/metrics"
	"github.com/wyfcoding/pkg/middleware"
)

// BootstrapName 服务唯一标识
const BootstrapName = "algotrading"

// Config 服务扩展配置
type Config struct {
	config.Config `mapstructure:",squash"`
	AlgoTrading   struct {
		BacktestConcurrency   int    `mapstructure:"backtest_concurrency" toml:"backtest_concurrency"`
		MarketDataServiceAddr string `mapstructure:"market_data_service_addr" toml:"market_data_service_addr"`
	} `mapstructure:"algotrading" toml:"algotrading"`
}

// AppContext 应用上下文
type AppContext struct {
	Config       *Config
	CmdService   *application.CommandService
	QueryService *application.QueryService
	HTTPHandler  *interfaces.HTTPHandler
	Metrics      *metrics.Metrics
}

func main() {
	if err := app.NewBuilder[*Config, *AppContext](BootstrapName).
		WithConfig(&Config{}).
		WithService(initService).
		WithGRPC(registerGRPC).
		WithGin(registerGin).
		WithGinMiddleware(
			middleware.CORS(),
			middleware.TimeoutMiddleware(30*time.Second),
		).
		Build().
		Run(); err != nil {
		slog.Error("service bootstrap failed", "error", err)
	}
}

func registerGRPC(s *grpc.Server, ctx *AppContext) {
	// 注册 gRPC 服务实现 (假设 interfaces 层已有实现)
	// pb.RegisterAlgoTradingServiceServer(s, interfaces.NewGRPCServer(ctx.CmdService, ctx.QueryService))
}

func registerGin(e *gin.Engine, ctx *AppContext) {
	if ctx.Config.Server.Environment == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	api := e.Group("/api/v1")
	{
		ctx.HTTPHandler.RegisterRoutes(api)
	}
}

func initService(cfg *Config, m *metrics.Metrics) (*AppContext, func(), error) {
	bootLog := slog.With("module", "bootstrap")
	logger := logging.Default()

	// 1. 数据库
	dbWrapper, err := database.NewDB(cfg.Data.Database, cfg.CircuitBreaker, logger, m)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init db: %w", err)
	}
	db := dbWrapper.RawDB()

	// 自动迁移
	if err := db.AutoMigrate(&domain.Strategy{}, &domain.Backtest{}, &outbox.Message{}); err != nil {
		return nil, nil, fmt.Errorf("failed to migrate tables: %w", err)
	}

	// 2. 消息队列 & Outbox
	producer := kafka.NewProducer(&cfg.MessageQueue.Kafka, logger, m)
	outboxMgr := outbox.NewManager(db, logger.Logger)
	outboxProc := outbox.NewProcessor(outboxMgr, func(ctx context.Context, topic, key string, payload []byte) error {
		return producer.PublishToTopic(ctx, topic, []byte(key), payload)
	}, 100, 5*time.Second)
	outboxProc.Start()

	// 3. 仓储
	strategyRepo := mysql.NewStrategyRepository(db)
	backtestRepo := mysql.NewBacktestRepository(db)

	// 4. 服务
	publisher := outbox.NewPublisher(outboxMgr)
	cmdService := application.NewCommandService(strategyRepo, backtestRepo, publisher, logger.Logger)
	queryService := application.NewQueryService(strategyRepo, backtestRepo, logger.Logger)

	// 5. Handler
	httpHandler := interfaces.NewHTTPHandler(cmdService, queryService)

	cleanup := func() {
		bootLog.Info("shutting down...")
		outboxProc.Stop()
		if producer != nil {
			producer.Close()
		}
		if sqlDB, err := db.DB(); err == nil && sqlDB != nil {
			sqlDB.Close()
		}
	}

	return &AppContext{
		Config:       cfg,
		CmdService:   cmdService,
		QueryService: queryService,
		HTTPHandler:  httpHandler,
		Metrics:      m,
	}, cleanup, nil
}
