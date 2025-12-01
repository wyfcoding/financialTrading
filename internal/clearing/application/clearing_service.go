// Package application 包含清算服务的用例逻辑
package application

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialTrading/internal/clearing/domain"
	"github.com/wyfcoding/financialTrading/pkg/logger"
	"github.com/wyfcoding/financialTrading/pkg/utils"
)

// SettleTradeRequest 清算交易请求 DTO
// 用于接收清算交易的请求参数
type SettleTradeRequest struct {
	TradeID    string // 交易 ID
	BuyUserID  string // 买方用户 ID
	SellUserID string // 卖方用户 ID
	Symbol     string // 交易对符号
	Quantity   string // 数量
	Price      string // 价格
}

// ClearingApplicationService 清算应用服务
// 负责处理清算相关的业务逻辑，包括实时清算和日终清算
type ClearingApplicationService struct {
	settlementRepo domain.SettlementRepository
	eodRepo        domain.EODClearingRepository
	snowflake      *utils.SnowflakeID
}

// NewClearingApplicationService 创建清算应用服务
func NewClearingApplicationService(
	settlementRepo domain.SettlementRepository,
	eodRepo domain.EODClearingRepository,
) *ClearingApplicationService {
	return &ClearingApplicationService{
		settlementRepo: settlementRepo,
		eodRepo:        eodRepo,
		snowflake:      utils.NewSnowflakeID(7),
	}
}

// SettleTrade 清算交易
func (cas *ClearingApplicationService) SettleTrade(ctx context.Context, req *SettleTradeRequest) error {
	// 验证输入
	if req.TradeID == "" || req.BuyUserID == "" || req.SellUserID == "" {
		return fmt.Errorf("invalid request parameters")
	}

	// 解析数量和价格
	quantity, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		return fmt.Errorf("invalid quantity: %w", err)
	}

	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		return fmt.Errorf("invalid price: %w", err)
	}

	// 生成清算 ID
	settlementID := fmt.Sprintf("SETTLE-%d", cas.snowflake.Generate())

	// 创建清算记录
	settlement := &domain.Settlement{
		SettlementID:   settlementID,
		TradeID:        req.TradeID,
		BuyUserID:      req.BuyUserID,
		SellUserID:     req.SellUserID,
		Symbol:         req.Symbol,
		Quantity:       quantity,
		Price:          price,
		Status:         "COMPLETED",
		SettlementTime: time.Now(),
		CreatedAt:      time.Now(),
	}

	// 保存清算记录
	if err := cas.settlementRepo.Save(ctx, settlement); err != nil {
		logger.WithContext(ctx).Error("Failed to save settlement",
			"settlement_id", settlementID,
			"error", err,
		)
		return fmt.Errorf("failed to save settlement: %w", err)
	}

	logger.WithContext(ctx).Debug("Trade settled successfully",
		"settlement_id", settlementID,
		"trade_id", req.TradeID,
	)

	return nil
}

// ExecuteEODClearing 执行日终清算
func (cas *ClearingApplicationService) ExecuteEODClearing(ctx context.Context, clearingDate string) error {
	// 生成清算 ID
	clearingID := fmt.Sprintf("EOD-%d", cas.snowflake.Generate())

	// 创建日终清算记录
	clearing := &domain.EODClearing{
		ClearingID:    clearingID,
		ClearingDate:  clearingDate,
		Status:        "PROCESSING",
		StartTime:     time.Now(),
		TradesSettled: 0,
		TotalTrades:   0,
	}

	// 保存日终清算记录
	if err := cas.eodRepo.Save(ctx, clearing); err != nil {
		logger.WithContext(ctx).Error("Failed to save EOD clearing",
			"clearing_id", clearingID,
			"error", err,
		)
		return fmt.Errorf("failed to save EOD clearing: %w", err)
	}

	logger.WithContext(ctx).Debug("EOD clearing started",
		"clearing_id", clearingID,
		"clearing_date", clearingDate,
	)

	return nil
}

// GetClearingStatus 获取清算状态
func (cas *ClearingApplicationService) GetClearingStatus(ctx context.Context, clearingID string) (*domain.EODClearing, error) {
	// 获取清算记录
	clearing, err := cas.eodRepo.Get(ctx, clearingID)
	if err != nil {
		logger.WithContext(ctx).Error("Failed to get clearing status",
			"clearing_id", clearingID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get clearing status: %w", err)
	}

	return clearing, nil
}
