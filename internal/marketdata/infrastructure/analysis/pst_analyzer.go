package analysis

import (
	"sync"

	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
	algorithm "github.com/wyfcoding/pkg/algorithm/graph"
)

// PSTHistoryAnalyzer 基于主席树的历史价格分布分析器（技术实现）。
type PSTHistoryAnalyzer struct {
	pst   *algorithm.PersistentSegmentTree
	times []int64 // 记录每个版本对应的时间戳
	mu    sync.RWMutex
}

func NewPSTHistoryAnalyzer(maxPrice int) domain.HistoryAnalyzer {
	return &PSTHistoryAnalyzer{
		pst:   algorithm.NewPersistentSegmentTree(maxPrice, 100000),
		times: make([]int64, 0),
	}
}

// RecordTrade 记录一笔成交
func (a *PSTHistoryAnalyzer) RecordTrade(price int, timestamp int64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.pst.PushVersion(price)
	a.times = append(a.times, timestamp)
}

// QueryVolumeAtTime 查询在特定时间点，价格区间 [low, high] 的成交活跃度
func (a *PSTHistoryAnalyzer) QueryVolumeAtTime(timestamp int64, low, high int) int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	vIdx := a.findVersionByTime(timestamp)
	if vIdx == -1 {
		return 0
	}

	return a.pst.QueryRange(vIdx, low, high)
}

func (a *PSTHistoryAnalyzer) findVersionByTime(t int64) int {
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
