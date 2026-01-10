package history

import (
	"sync"

	algorithm "github.com/wyfcoding/pkg/algorithm/graph"
)

// PriceDistributionAnalyzer 基于主席树的价格分布历史分析器
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
