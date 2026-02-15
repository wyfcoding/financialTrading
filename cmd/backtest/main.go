package main

import (
	"fmt"
	"log/slog"

	"google.golang.org/grpc"

	pb "github.com/wyfcoding/financialtrading/go-api/backtest/v1"
	"github.com/wyfcoding/financialtrading/internal/backtest/application"
	"github.com/wyfcoding/financialtrading/internal/backtest/domain"
	"github.com/wyfcoding/financialtrading/internal/backtest/infrastructure/persistence/mysql"
	"github.com/wyfcoding/financialtrading/internal/backtest/interfaces"
	"github.com/wyfcoding/pkg/app"
	"github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/database"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/metrics"
)

// BootstrapName 服务唯一标识
const BootstrapName = "backtest"

// Config 服务扩展配置
type Config struct {
	config.Config `mapstructure:",squash"`
}

// AppContext 应用上下文
type AppContext struct {
	Config     *Config
	AppService *application.BacktestApplicationService
	Metrics    *metrics.Metrics
}

func main() {
	if err := app.NewBuilder[*Config, *AppContext](BootstrapName).
		WithConfig(&Config{}).
		WithService(initService).
		WithGRPC(registerGRPC).
		Build().
		Run(); err != nil {
		slog.Error("service bootstrap failed", "error", err)
	}
}

func registerGRPC(s *grpc.Server, ctx *AppContext) {
	pb.RegisterBacktestServiceServer(s, interfaces.NewBacktestHandler(ctx.AppService))
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
	if err := db.AutoMigrate(&domain.BacktestTask{}, &domain.BacktestReport{}); err != nil {
		return nil, nil, fmt.Errorf("failed to migrate tables: %w", err)
	}

	// 2. 依赖注入
	engine := &domain.BacktestEngine{}
	repo := mysql.NewBacktestRepository(db)
	appService := application.NewBacktestApplicationService(engine, repo, logger.Logger)

	cleanup := func() {
		bootLog.Info("shutting down...")
		if sqlDB, err := db.DB(); err == nil && sqlDB != nil {
			sqlDB.Close()
		}
	}

	return &AppContext{
		Config:     cfg,
		AppService: appService,
		Metrics:    m,
	}, cleanup, nil
}
