package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	accountv1 "github.com/wyfcoding/financialtrading/go-api/account/v1"
	positionv1 "github.com/wyfcoding/financialtrading/go-api/position/v1"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
)

// LiquidationEngine 强平引擎，负责定期检查杠杆账户风险并触发强平。
type LiquidationEngine struct {
	accountClient  accountv1.AccountServiceClient
	positionClient positionv1.PositionServiceClient
	publisher      domain.EventPublisher
	logger         *slog.Logger
	checkInterval  time.Duration
	mmThreshold    decimal.Decimal // 维持保证金阈值，例如 110% (1.1)
}

func NewLiquidationEngine(
	accClient accountv1.AccountServiceClient,
	posClient positionv1.PositionServiceClient,
	publisher domain.EventPublisher,
	logger *slog.Logger,
) *LiquidationEngine {
	return &LiquidationEngine{
		accountClient:  accClient,
		positionClient: posClient,
		publisher:      publisher,
		logger:         logger,
		checkInterval:  10 * time.Second,
		mmThreshold:    decimal.NewFromFloat(1.1),
	}
}

// Start 启动强平引擎监控循环。
func (e *LiquidationEngine) Start(ctx context.Context) error {
	ticker := time.NewTicker(e.checkInterval)
	defer ticker.Stop()

	e.logger.Info("Liquidation Engine started", "interval", e.checkInterval, "threshold", e.mmThreshold.String())

	for {
		select {
		case <-ctx.Done():
			e.logger.Info("Liquidation Engine stopping...")
			return nil
		case <-ticker.C:
			if err := e.RunCycle(ctx); err != nil {
				e.logger.Error("Liquidation cycle failed", "error", err)
			}
		}
	}
}

// RunCycle 执行一次扫描。
func (e *LiquidationEngine) RunCycle(ctx context.Context) error {
	pageToken := int32(0)
	for {
		resp, err := e.accountClient.ListAccounts(ctx, &accountv1.ListAccountsRequest{
			AccountType: "MARGIN",
			PageSize:    100,
			PageToken:   pageToken,
		})
		if err != nil {
			return fmt.Errorf("failed to list accounts: %w", err)
		}

		for _, acc := range resp.Accounts {
			if err := e.CheckAccountRisk(ctx, acc); err != nil {
				e.logger.Error("failed to check account risk", "account_id", acc.AccountId, "error", err)
			}
		}

		if resp.NextPageToken == 0 {
			break
		}
		pageToken = resp.NextPageToken
	}
	return nil
}

// CheckAccountRisk 检查单个账户的强平风险。
func (e *LiquidationEngine) CheckAccountRisk(ctx context.Context, acc *accountv1.AccountResponse) error {
	posResp, err := e.positionClient.GetPositions(ctx, &positionv1.GetPositionsRequest{
		UserId: acc.UserId,
	})
	if err != nil {
		return fmt.Errorf("failed to get positions for user %s: %w", acc.UserId, err)
	}

	totalUsedMargin := decimal.Zero
	totalUnrealizedPnL := decimal.Zero

	balance, _ := decimal.NewFromString(acc.Balance)

	for _, pos := range posResp.Positions {
		marginReq, _ := decimal.NewFromString(pos.MarginRequirement)
		unrealizedPnL, _ := decimal.NewFromString(pos.UnrealizedPnl)

		totalUsedMargin = totalUsedMargin.Add(marginReq)
		totalUnrealizedPnL = totalUnrealizedPnL.Add(unrealizedPnL)
	}

	if totalUsedMargin.IsZero() {
		return nil
	}

	equity := balance.Add(totalUnrealizedPnL)
	marginLevel := equity.Div(totalUsedMargin)

	e.logger.Debug("Account risk check",
		"account_id", acc.AccountId,
		"equity", equity.String(),
		"used_margin", totalUsedMargin.String(),
		"margin_level", marginLevel.String())

	if marginLevel.LessThan(e.mmThreshold) {
		e.logger.Warn("LIQUIDATION TRIGGERED",
			"account_id", acc.AccountId,
			"user_id", acc.UserId,
			"margin_level", marginLevel.String())

		for _, pos := range posResp.Positions {
			qty, _ := decimal.NewFromString(pos.Quantity)
			event := domain.PositionLiquidationTriggeredEvent{
				UserID:        acc.UserId,
				AccountID:     acc.AccountId,
				Symbol:        pos.Symbol,
				Side:          pos.Side,
				Quantity:      qty.InexactFloat64(),
				MarginLevel:   marginLevel.InexactFloat64(),
				TriggerReason: "Margin Level below MM threshold",
				TriggeredAt:   time.Now().Unix(),
				OccurredOn:    time.Now(),
			}

			if e.publisher != nil {
				if err := e.publisher.Publish(ctx, domain.PositionLiquidationTriggeredEventType, acc.UserId, event); err != nil {
					e.logger.Error("failed to publish liquidation event", "error", err)
				}
			}
		}
	}

	return nil
}
