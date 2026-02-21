package main

import (
	"fmt"
	"log/slog"

	pb "github.com/wyfcoding/financialtrading/go-api/feemanagement/v1"
	"github.com/wyfcoding/financialtrading/internal/feemanagement/application"
	feemysql "github.com/wyfcoding/financialtrading/internal/feemanagement/infrastructure/persistence/mysql"
	feegrpc "github.com/wyfcoding/financialtrading/internal/feemanagement/interfaces/grpc"

	"github.com/wyfcoding/pkg/app"
	configpkg "github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/database"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/metrics"

	"google.golang.org/grpc"
)

const BootstrapName = "feemanagement"

type Config struct {
	configpkg.Config `mapstructure:",squash"`
}

type AppContext struct {
	Config  *Config
	Svc     *application.FeeService
	Metrics *metrics.Metrics
}

func main() {
	if err := app.NewBuilder[*Config, *AppContext](BootstrapName).
		WithConfig(&Config{}).
		WithService(initService).
		WithGRPC(registerGRPC).
		// Note: Fee Management does not have a public HTTP API, skip WithGin.
		Build().
		Run(); err != nil {
		slog.Error("service bootstrap failed", "error", err)
	}
}

func registerGRPC(s *grpc.Server, ctx *AppContext) {
	pb.RegisterFeeManagementServiceServer(s, feegrpc.NewFeeHandler(ctx.Svc, logging.Default().Logger))
}

func initService(cfg *Config, m *metrics.Metrics) (*AppContext, func(), error) {
	c := cfg
	bootLog := slog.With("module", "bootstrap")
	logger := logging.Default()

	configpkg.PrintWithMask(c)

	// 1. DB
	db, err := database.NewDB(c.Data.Database, c.CircuitBreaker, logger, m)
	if err != nil {
		return nil, nil, fmt.Errorf("database init error: %w", err)
	}

	// db.AutoMigrate runs inside NewFeeRepository

	// 2. Repositories
	repo := feemysql.NewFeeRepository(db.RawDB())

	// 3. Application Services
	idGenerator, err := idgen.NewSnowflakeGenerator(configpkg.SnowflakeConfig{MachineID: 1})
	if err != nil {
		return nil, nil, fmt.Errorf("idgen init error: %w", err)
	}
	svc := application.NewFeeService(repo, idGenerator, logger.Logger)

	cleanup := func() {
		bootLog.Info("shutting down, releasing resources...")
		if sqlDB, err := db.RawDB().DB(); err == nil && sqlDB != nil {
			if err := sqlDB.Close(); err != nil {
				bootLog.Error("failed to close sql database", "error", err)
			}
		}
	}

	return &AppContext{
		Config:  c,
		Svc:     svc,
		Metrics: m,
	}, cleanup, nil
}
