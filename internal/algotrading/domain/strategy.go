// 变更说明：
// 1. 将所有价格和数量改为 decimal.Decimal
package domain

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

type StrategyStatus string

const (
	StrategyStatusCreated   StrategyStatus = "CREATED"
	StrategyStatusRunning   StrategyStatus = "RUNNING"
	StrategyStatusPaused    StrategyStatus = "PAUSED"
	StrategyStatusStopped   StrategyStatus = "STOPPED"
	StrategyStatusCompleted StrategyStatus = "COMPLETED"
	StrategyStatusFailed    StrategyStatus = "FAILED"
)

type StrategyType int8

const (
	StrategyTypeUnknown StrategyType = 0
	StrategyTypeTWAP    StrategyType = 1
	StrategyTypeVWAP    StrategyType = 2
	StrategyTypeIceberg StrategyType = 3
	StrategyTypeSniper  StrategyType = 4
)

type Strategy struct {
	ID               string          `json:"id" gorm:"primaryKey"`
	StrategyID       string          `json:"strategy_id" gorm:"column:strategy_id;uniqueIndex"`
	UserID           uint64          `json:"user_id"`
	Type             StrategyType    `json:"type" gorm:"column:type"`
	AlgorithmName    string          `json:"algorithm_name"` // TWAP, VWAP, ICEBERG, SNIPER
	Symbol           string          `json:"symbol"`
	Side             string          `json:"side"`       // BUY, SELL
	Parameters       string          `json:"parameters"` // JSON serialized params
	TotalQuantity    decimal.Decimal `json:"total_quantity"`
	ExecutedAmount   int64           `json:"executed_amount"`
	ExecutedQuantity decimal.Decimal `json:"executed_quantity"`
	Status           StrategyStatus  `json:"status"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`

	domainEvents []DomainEvent `json:"-" gorm:"-"`
}

func NewStrategy(strategyID string, userID uint64, strategyType StrategyType, symbol, parameters string) *Strategy {
	now := time.Now()
	return &Strategy{
		ID:           strategyID,
		StrategyID:   strategyID,
		UserID:       userID,
		Type:         strategyType,
		Symbol:       symbol,
		Parameters:   parameters,
		Status:       StrategyStatusCreated,
		CreatedAt:    now,
		UpdatedAt:    now,
		domainEvents: make([]DomainEvent, 0),
	}
}

func (s *Strategy) Start() error {
	if s.Status == StrategyStatusRunning {
		return nil
	}
	if s.Status == StrategyStatusStopped || s.Status == StrategyStatusCompleted {
		return errors.New("strategy cannot be started from current status")
	}
	s.Status = StrategyStatusRunning
	s.UpdatedAt = time.Now()
	s.domainEvents = append(s.domainEvents, &StrategyStartedEvent{
		StrategyID: s.getStrategyID(),
		UserID:     s.UserID,
		Timestamp:  s.UpdatedAt,
	})
	return nil
}

func (s *Strategy) Stop() error {
	if s.Status == StrategyStatusStopped {
		return nil
	}
	if s.Status == StrategyStatusCompleted {
		return errors.New("completed strategy cannot be stopped")
	}
	s.Status = StrategyStatusStopped
	s.UpdatedAt = time.Now()
	s.domainEvents = append(s.domainEvents, &StrategyStoppedEvent{
		StrategyID: s.getStrategyID(),
		UserID:     s.UserID,
		Timestamp:  s.UpdatedAt,
	})
	return nil
}

func (s *Strategy) GetDomainEvents() []DomainEvent {
	return s.domainEvents
}

func (s *Strategy) ClearDomainEvents() {
	s.domainEvents = s.domainEvents[:0]
}

func (s *Strategy) getStrategyID() string {
	if s.StrategyID != "" {
		return s.StrategyID
	}
	return s.ID
}
