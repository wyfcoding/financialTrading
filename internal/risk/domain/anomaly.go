package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// AnomalyType 定义异常类型
type AnomalyType string

const (
	AnomalyTypeWashTrading AnomalyType = "WASH_TRADING"
	AnomalyTypeSpoofing    AnomalyType = "SPOOFING"
	AnomalyTypeLayering    AnomalyType = "LAYERING"
)

// AnomalyRecord 异常记录
type AnomalyRecord struct {
	Type            AnomalyType     `json:"type"`
	Symbol          string          `json:"symbol"`
	UserID          string          `json:"user_id"`
	CounterpartyID  string          `json:"counterparty_id"`
	Price           decimal.Decimal `json:"price"`
	Quantity        decimal.Decimal `json:"quantity"`
	ConfidenceScore float64         `json:"confidence_score"`
	Reason          string          `json:"reason"`
	Timestamp       time.Time       `json:"timestamp"`
}

// WashTradingDetector 洗钱/自成交检测器
type WashTradingDetector struct {
	// 阈值配置
	MaxSelfTradeRatio float64 // 自成交比例阈值
}

// DetectWashTrading 检测是否存在自成交行为
func (d *WashTradingDetector) DetectWashTrading(trades []*TradeInfo) []*AnomalyRecord {
	anomalies := make([]*AnomalyRecord, 0)

	// 简单实现：检查买卖双方是否为同一 UserID
	for _, t := range trades {
		if t.BuyerID == t.SellerID && t.BuyerID != "" {
			anomalies = append(anomalies, &AnomalyRecord{
				Type:            AnomalyTypeWashTrading,
				Symbol:          t.Symbol,
				UserID:          t.BuyerID,
				Price:           t.Price,
				Quantity:        t.Quantity,
				ConfidenceScore: 1.0,
				Reason:          "Self-matching trade detected (same buyer and seller)",
				Timestamp:       t.Timestamp,
			})
		}
	}

	return anomalies
}

// TradeInfo 内部简化的交易信息，用于分析
type TradeInfo struct {
	Symbol    string
	BuyerID   string
	SellerID  string
	Price     decimal.Decimal
	Quantity  decimal.Decimal
	Timestamp time.Time
}
