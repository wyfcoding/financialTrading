//go:build clearing_legacy_pipeline
// +build clearing_legacy_pipeline

package application

import (
	"context"
	"fmt"
	"log"

	"github.com/shopspring/decimal"
	ledgerApp "github.com/wyfcoding/financialtrading/internal/ledger/application"
	matchDomain "github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
)

type ClearingService struct {
	ledgerService *ledgerApp.LedgerService
}

func NewClearingService(l *ledgerApp.LedgerService) *ClearingService {
	return &ClearingService{ledgerService: l}
}

// ProcessTrade 处理成交记录，生成清算指令
// 这是一个同步阻塞调用，实际生产中应消费 Kafka Topic
func (s *ClearingService) ProcessTrade(ctx context.Context, trade *matchDomain.Trade) error {
	// 1. 计算金额
	// 买方：获得 Coin (Quantity), 支付 Quote (Price * Quantity)
	// 卖方：支付 Coin (Quantity), 获得 Quote (Price * Quantity)

	totalAmount := trade.Price.Mul(trade.Quantity)

	// HARDCODED for demo: 假设 Symbol 格式为 "BTC-USDT"
	// 实际上需要解析 Symbol 或查询 Instrument 服务
	baseCurrency := "BTC"
	quoteCurrency := "USDT"

	// 2. 调用 Ledger 进行双向转账
	// 注意：这里为了演示简单，直接用 "user-{id}-{currency}" 作为 AccountID
	// 实际 Ledger 中 AccountID 通常是 UUID，需要有一张 User->Account 的映射表

	buyerMoneyAcct := fmt.Sprintf("user-%d-%s", trade.BuyerID, quoteCurrency)
	sellerMoneyAcct := fmt.Sprintf("user-%d-%s", trade.SellerID, quoteCurrency)

	buyerAssetAcct := fmt.Sprintf("user-%d-%s", trade.BuyerID, baseCurrency)
	sellerAssetAcct := fmt.Sprintf("user-%d-%s", trade.SellerID, baseCurrency)

	// A. 资金结算 (USDT): 买方 -> 卖方
	// TxID 建议具备幂等性: tradeID + "-money"
	txIDMoney := fmt.Sprintf("trade-%d-money", trade.TradeID)
	err := s.ledgerService.Transfer(ctx,
		txIDMoney,
		buyerMoneyAcct,  // From
		sellerMoneyAcct, // To
		quoteCurrency,
		totalAmount,
		"Trade Settlement - Money",
	)
	if err != nil {
		log.Printf("❌ Clearing Money Failed [Trade %d]: %v", trade.TradeID, err)
		return err
	}

	// B. 资产结算 (BTC): 卖方 -> 买方
	txIDAsset := fmt.Sprintf("trade-%d-asset", trade.TradeID)
	err = s.ledgerService.Transfer(ctx,
		txIDAsset,
		sellerAssetAcct, // From
		buyerAssetAcct,  // To
		baseCurrency,
		trade.Quantity,
		"Trade Settlement - Asset",
	)
	if err != nil {
		log.Printf("❌ Clearing Asset Failed [Trade %d]: %v", trade.TradeID, err)
		// 严重错误：资金已转但资产未转，需要人工介入或 Saga 补偿
		return err
	}

	log.Printf("✅ Trade %d Settled: %s %s transferred, %s %s transferred",
		trade.TradeID, totalAmount, quoteCurrency, trade.Quantity, baseCurrency)
	return nil
}
