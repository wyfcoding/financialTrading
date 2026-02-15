package main

import (
	"fmt"
	"log/slog"

	"google.golang.org/grpc"

	v1 "github.com/wyfcoding/financialtrading/go-api/fixgateway/v1"
	"github.com/wyfcoding/financialtrading/internal/fixgateway/application"
	persistence_mysql "github.com/wyfcoding/financialtrading/internal/fixgateway/infrastructure/persistence/mysql"
	grpc_server "github.com/wyfcoding/financialtrading/internal/fixgateway/interfaces/grpc"
	"github.com/wyfcoding/pkg/app"
	"github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/database"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/metrics"
)

// BootstrapName 服务唯一标识
const BootstrapName = "fixgateway"

// Config 服务扩展配置
type Config struct {
	config.Config `mapstructure:",squash"`
}

// AppContext 应用上下文
type AppContext struct {
	Config     *Config
	AppService *application.FixApplicationService
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
	v1.RegisterFixGatewayServiceServer(s, grpc_server.NewServer(ctx.AppService))
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
	if err := db.AutoMigrate(&persistence_mysql.FixSessionModel{}); err != nil {
		return nil, nil, fmt.Errorf("failed to migrate tables: %w", err)
	}

	// 2. 依赖注入
	repo := persistence_mysql.NewGormFixRepository(db)
	appService := application.NewFixApplicationService(repo, nil, logger.Logger)

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
