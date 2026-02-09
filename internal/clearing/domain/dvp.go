// 变更说明：实现 DVP (银货对付) 交收流程，确保券款同步交换，防止本金风险。
// 假设：DVP 流程包含预取券、预锁款及最终交收三个阶段。
package domain

import (
	"fmt"
	"time"
)

// DVPStatus 交收状态
type DVPStatus string

const (
	DVPStatusPending     DVPStatus = "PENDING"
	DVPStatusAssetLocked DVPStatus = "ASSET_LOCKED"
	DVPStatusFundLocked  DVPStatus = "FUND_LOCKED"
	DVPStatusSettled     DVPStatus = "SETTLED"
	DVPStatusFailed      DVPStatus = "FAILED"
)

// DVPSettlement DVP 交收对象
type DVPSettlement struct {
	DVPID        string
	SettlementID string
	Status       DVPStatus
	AssetLocked  bool
	FundLocked   bool
	LastUpdate   time.Time
}

// DVPEngine DVP 交收引擎
type DVPEngine struct{}

func NewDVPEngine() *DVPEngine {
	return &DVPEngine{}
}

// InitiateDVP 发起 DVP 流程
func (e *DVPEngine) InitiateDVP(s *Settlement) *DVPSettlement {
	return &DVPSettlement{
		DVPID:        "DVP-" + s.SettlementID,
		SettlementID: s.SettlementID,
		Status:       DVPStatusPending,
		LastUpdate:   time.Now(),
	}
}

// LockAsset 锁定资产 (券)
func (e *DVPEngine) LockAsset(dvp *DVPSettlement) error {
	if dvp.Status != DVPStatusPending {
		return fmt.Errorf("invalid status for asset locking: %s", dvp.Status)
	}
	dvp.AssetLocked = true
	if dvp.FundLocked {
		dvp.Status = DVPStatusSettled
	} else {
		dvp.Status = DVPStatusAssetLocked
	}
	dvp.LastUpdate = time.Now()
	return nil
}

// LockFund 锁定资金 (款)
func (e *DVPEngine) LockFund(dvp *DVPSettlement) error {
	if dvp.Status != DVPStatusPending && dvp.Status != DVPStatusAssetLocked {
		return fmt.Errorf("invalid status for fund locking: %s", dvp.Status)
	}
	dvp.FundLocked = true
	if dvp.AssetLocked {
		dvp.Status = DVPStatusSettled
	} else {
		dvp.Status = DVPStatusFundLocked
	}
	dvp.LastUpdate = time.Now()
	return nil
}

// Finalize 执行交收
func (e *DVPEngine) Finalize(dvp *DVPSettlement) error {
	if dvp.Status != DVPStatusSettled {
		return fmt.Errorf("cannot finalize: assets or funds not locked")
	}
	dvp.LastUpdate = time.Now()
	return nil
}
