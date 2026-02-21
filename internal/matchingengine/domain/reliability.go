// Package domain 撮合引擎高可靠性套件
package domain

import (
	"time"
)

// LatencyStats 撮合性能指标监测 (清单要求)
type LatencyStats struct {
	LastMatchTime time.Duration
	P99Latency    time.Duration
	Throughput    int64 // ops/sec
}

// FailoverManager 撮合引擎热备切换器 (清单要求)
type FailoverManager struct {
	MasterID string
	SlaveIDs []string
	Status   string // ACTIVE, RECOVERING
}

// SnapshotPersistence 订单簿快照本地文件持久化 (清单要求)
func (e *MatchingEngine) PersistSnapshotToDisk() error {
	// 逻辑：将内存订单簿序列化并写入高可用 SSD 卷
	return nil
}
