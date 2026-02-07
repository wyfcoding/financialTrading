package application

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketdata/history"
)

const (
	defaultHistoryPriceScale = 100
	defaultHistoryMaxPrice   = 1_000_000 // scale 100 -> max price 10,000.00
)

// HistoryService 提供基于主席树的历史价格分布能力。
// 当前实现为单实例统计（不区分 symbol），用于轻量级实时分析。
type HistoryService struct {
	analyzer *history.PriceDistributionAnalyzer
	maxPrice int
	scale    int
}

func NewHistoryService() *HistoryService {
	return NewHistoryServiceWithConfig(defaultHistoryMaxPrice, defaultHistoryPriceScale)
}

func NewHistoryServiceWithConfig(maxPrice, scale int) *HistoryService {
	if maxPrice <= 0 {
		maxPrice = defaultHistoryMaxPrice
	}
	if scale <= 0 {
		scale = defaultHistoryPriceScale
	}
	return &HistoryService{
		analyzer: history.NewPriceDistributionAnalyzer(maxPrice),
		maxPrice: maxPrice,
		scale:    scale,
	}
}

// RecordTrade 记录成交（symbol 预留，当前实现不区分标的）。
func (s *HistoryService) RecordTrade(symbol string, price decimal.Decimal, ts time.Time) {
	if s == nil || s.analyzer == nil {
		return
	}
	pos := s.toPosition(price)
	if pos <= 0 {
		return
	}
	s.analyzer.RecordTrade(pos, ts.Unix())
}

// QueryVolumeAtTime 查询在时间点的价格区间成交活跃度。
// symbol 预留，当前实现不区分标的。
func (s *HistoryService) QueryVolumeAtTime(symbol string, ts time.Time, low, high decimal.Decimal) int {
	if s == nil || s.analyzer == nil {
		return 0
	}
	l := s.toPosition(low)
	r := s.toPosition(high)
	if l == 0 || r == 0 {
		return 0
	}
	if l > r {
		l, r = r, l
	}
	return s.analyzer.QueryVolumeAtTime(ts.Unix(), l, r)
}

func (s *HistoryService) toPosition(price decimal.Decimal) int {
	if price.LessThanOrEqual(decimal.Zero) {
		return 0
	}
	scaled := price.Mul(decimal.NewFromInt(int64(s.scale))).IntPart()
	pos := int(scaled)
	if pos < 1 {
		pos = 1
	}
	if pos > s.maxPrice {
		pos = s.maxPrice
	}
	return pos
}
