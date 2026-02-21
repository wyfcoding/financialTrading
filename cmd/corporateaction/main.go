package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc"

	pb "github.com/wyfcoding/financialtrading/go-api/corporateaction/v1"
	"github.com/wyfcoding/financialtrading/internal/corporateaction/application"
	camysql "github.com/wyfcoding/financialtrading/internal/corporateaction/infrastructure/persistence/mysql"
	cagrpc "github.com/wyfcoding/financialtrading/internal/corporateaction/interfaces/grpc"

	"github.com/wyfcoding/pkg/app"
	configpkg "github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/database"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/metrics"
)

const BootstrapName = "corporateaction"

type Config struct {
	configpkg.Config `mapstructure:",squash"`
}

type AppContext struct {
	Config  *Config
	Svc     *application.CorporateActionService
	Metrics *metrics.Metrics
}

func main() {
	if err := app.NewBuilder[*Config, *AppContext](BootstrapName).
		WithConfig(&Config{}).
		WithService(initService).
		WithGRPC(registerGRPC).
		// Corporate actions are mostly internal or triggered by admin/cron, so no public HTTP API for now
		Build().
		Run(); err != nil {
		slog.Error("service bootstrap failed", "error", err)
	}
}

func registerGRPC(s *grpc.Server, ctx *AppContext) {
	pb.RegisterCorporateActionServiceServer(s, cagrpc.NewCorporateActionHandler(ctx.Svc, logging.Default().Logger))
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

	// db.AutoMigrate triggers inside repository constructors

	// 2. Repositories
	actionRepo := camysql.NewActionRepository(db.RawDB())
	entRepo := camysql.NewEntitlementRepository(db.RawDB())

	// 3. Application Services & Mocks
	// 实际生产中这里会传入 gRPC Client包装类
	posSvc := &mockPositionService{}
	cashSvc := &mockCashService{}
	secSvc := &mockSecurityService{}

	svc := application.NewCorporateActionService(actionRepo, entRepo, posSvc, cashSvc, secSvc, logger.Logger)

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

// ---------------------------------------------------------
// Mock Implementations for external dependencies to allow compile
// ---------------------------------------------------------

type mockPositionService struct{}

func (m *mockPositionService) GetPosition(ctx context.Context, accountID, symbol string, date time.Time) (decimal.Decimal, error) {
	return decimal.NewFromInt(1000), nil // 模拟每个账户持有 1000 股
}

func (m *mockPositionService) ListHolders(ctx context.Context, symbol string, date time.Time) ([]string, error) {
	return []string{"user-A", "user-B"}, nil // 模拟两个持有用户
}

type mockCashService struct{}

func (m *mockCashService) Deposit(ctx context.Context, accountID string, amount decimal.Decimal, currency string, refID string) error {
	slog.Info("Mock Cash Deposit", "account", accountID, "amount", amount.String(), "currency", currency, "ref", refID)
	return nil
}

type mockSecurityService struct{}

func (m *mockSecurityService) AdjustPosition(ctx context.Context, accountID, symbol string, delta decimal.Decimal, refID string) error {
	slog.Info("Mock Security Adjust", "account", accountID, "symbol", symbol, "delta", delta.String(), "ref", refID)
	return nil
}
