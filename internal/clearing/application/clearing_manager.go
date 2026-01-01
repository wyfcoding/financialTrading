package application

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
	"github.com/wyfcoding/pkg/idgen"
)

// ClearingManager 处理所有清算相关的写入操作（Commands）。
type ClearingManager struct {
	settlementRepo domain.SettlementRepository
	eodRepo        domain.EODClearingRepository
	marginRepo     domain.MarginRequirementRepository
}

// NewClearingManager 构造函数。
func NewClearingManager(settlementRepo domain.SettlementRepository, eodRepo domain.EODClearingRepository, marginRepo domain.MarginRequirementRepository) *ClearingManager {
	return &ClearingManager{
		settlementRepo: settlementRepo,
		eodRepo:        eodRepo,
		marginRepo:     marginRepo,
	}
}

// SettleTrade 清算单笔交易
func (m *ClearingManager) SettleTrade(ctx context.Context, req *SettleTradeRequest) (string, error) {
	if req.TradeID == "" || req.BuyUserID == "" || req.SellUserID == "" {
		return "", fmt.Errorf("invalid request parameters")
	}

	quantity, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		return "", fmt.Errorf("invalid quantity format: %w", err)
	}

	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		return "", fmt.Errorf("invalid price format: %w", err)
	}

	settlementID := fmt.Sprintf("SETTLE-%d", idgen.GenID())

	settlement := &domain.Settlement{
		SettlementID:   settlementID,
		TradeID:        req.TradeID,
		BuyUserID:      req.BuyUserID,
		SellUserID:     req.SellUserID,
		Symbol:         req.Symbol,
		Quantity:       quantity,
		Price:          price,
		Status:         domain.SettlementStatusCompleted,
		SettlementTime: time.Now(),
	}

	if err := m.settlementRepo.Save(ctx, settlement); err != nil {
		return "", err
	}

	return settlementID, nil
}

// ExecuteEODClearing 执行日终清算
func (m *ClearingManager) ExecuteEODClearing(ctx context.Context, clearingDate string) (string, error) {
	clearingID := fmt.Sprintf("EOD-%d", idgen.GenID())

	clearing := &domain.EODClearing{
		ClearingID:    clearingID,
		ClearingDate:  clearingDate,
		Status:        domain.ClearingStatusProcessing,
		StartTime:     time.Now(),
		TradesSettled: 0,
		TotalTrades:   0,
	}

	if err := m.eodRepo.Save(ctx, clearing); err != nil {
		return "", err
	}

	return clearingID, nil
}

// GetMarginRequirement 获取指定品种的实时保证金要求
func (m *ClearingManager) GetMarginRequirement(ctx context.Context, symbol string) (*domain.MarginRequirement, error) {
	margin, err := m.marginRepo.GetBySymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}
	if margin == nil {
		// 默认兜底：10% 基础保证金率，0% 波动率加成
		return &domain.MarginRequirement{
			Symbol:           symbol,
			BaseMarginRate:   decimal.NewFromFloat(0.1),
			VolatilityFactor: decimal.Zero,
		}, nil
	}
	return margin, nil
}
