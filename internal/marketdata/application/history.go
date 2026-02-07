package application

import (
	"sync"
	"time"

	"github.com/shopspring/decimal"
	algorithm "github.com/wyfcoding/pkg/algorithm/graph"
)

const (
	defaultHistoryPriceScale = 100
	defaultHistoryMaxPrice   = 1_000_000 // scale 100 -> max price 10,000.00
)

// PriceDistributionAnalyzer 基于主席树的价格分布历史分析器
// 迁入 application，避免额外的 history 包。
type PriceDistributionAnalyzer struct {
	pst   *algorithm.PersistentSegmentTree
	times []int64 // 记录每个版本对应的时间戳
	mu    sync.RWMutex
}

// NewPriceDistributionAnalyzer 创建分析器
// maxPrice: 允许的最高价格（用于线段树区间）
func NewPriceDistributionAnalyzer(maxPrice int) *PriceDistributionAnalyzer {
	return &PriceDistributionAnalyzer{
		pst:   algorithm.NewPersistentSegmentTree(maxPrice, 100000),
		times: make([]int64, 0),
	}
}

// RecordTrade 记录一笔成交
func (a *PriceDistributionAnalyzer) RecordTrade(price int, timestamp int64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.pst.PushVersion(price)
	a.times = append(a.times, timestamp)
}

// QueryVolumeAtTime 查询在特定时间点，价格区间 [low, high] 的成交活跃度
func (a *PriceDistributionAnalyzer) QueryVolumeAtTime(timestamp int64, low, high int) int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// 1. 找到该时间戳对应的版本号 (二分查找)
	vIdx := a.findVersionByTime(timestamp)
	if vIdx == -1 {
		return 0
	}

	// 2. 使用主席树进行 $O(\log N)$ 查询
	return a.pst.QueryRange(vIdx, low, high)
}

func (a *PriceDistributionAnalyzer) findVersionByTime(t int64) int {
	l, r := 0, len(a.times)-1
	ans := -1
	for l <= r {
		mid := (l + r) >> 1
		if a.times[mid] <= t {
			ans = mid
			l = mid + 1
		} else {
			r = mid - 1
		}
	}
	return ans
}

// HistoryService 提供基于主席树的历史价格分布能力。
// 当前实现为单实例统计（不区分 symbol），用于轻量级实时分析。
type HistoryService struct {
	analyzer *PriceDistributionAnalyzer
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
		analyzer: NewPriceDistributionAnalyzer(maxPrice),
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
