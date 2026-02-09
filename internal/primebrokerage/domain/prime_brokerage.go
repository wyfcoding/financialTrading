// Package domain 提供了主经纪商（Prime Brokerage）领域的业务逻辑。
// 变更说明：实现主经纪商核心逻辑，支持多个清算席位（Clearing Seats）的路由管理及借券库（Security Lending Pool）的自动化配额。
package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ClearingSeat 清算席位（对应不同的交易所或清算行）
type ClearingSeat struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	ExchangeCode string  `json:"exchange_code"`
	Capacity     int64   `json:"capacity"`       // 剩余处理额度
	Latency      int64   `json:"latency"`        // 实时延迟（毫秒）
	CostPerTrade float64 `json:"cost_per_trade"` // 单笔交易手续费
	Status       string  `json:"status"`         // ACTIVE, BUSY, INACTIVE
}

// SeatRoutingStrategy 席位路由策略
type SeatRoutingStrategy interface {
	SelectSeat(ctx context.Context, exchange string, amount int64) (*ClearingSeat, error)
}

// SecurityPool 借券库实体
type SecurityPool struct {
	Symbol      string  `json:"symbol"`
	TotalVolume int64   `json:"total_volume"` // 总券源容量
	LentVolume  int64   `json:"lent_volume"`  // 已借出容量
	MinRate     float64 `json:"min_rate"`     // 最小借券利率
	Status      string  `json:"status"`       // AVAILABLE, RESTRICTED
}

// SecurityLoan 借券记录
type SecurityLoan struct {
	LoanID   string    `json:"loan_id"`
	UserID   uint64    `json:"user_id"`
	Symbol   string    `json:"symbol"`
	Quantity int64     `json:"quantity"`
	Rate     float64   `json:"rate"`
	LoanedAt time.Time `json:"loaned_at"`
	DueAt    time.Time `json:"due_at"`
	Status   string    `json:"status"` // ACTIVE, RETURNED, OVERDUE
}

// PrimeBrokerageService 主经纪商服务接口
type PrimeBrokerageService interface {
	// RouteToSeat 根据最优策略选择清算席位
	RouteToSeat(ctx context.Context, symbol string, amount int64) (*ClearingSeat, error)
	// BorrowSecurity 申请借券（用于融券交易）
	BorrowSecurity(ctx context.Context, userID uint64, symbol string, quantity int64) (*SecurityLoan, error)
	// ReturnSecurity 归还证券
	ReturnSecurity(ctx context.Context, loanID string) error
}

// DefaultSeatRouter 默认席位路由器（基于成本和延迟的最优选择）
type DefaultSeatRouter struct {
	Seats []*ClearingSeat
}

func (r *DefaultSeatRouter) SelectSeat(ctx context.Context, exchange string, amount int64) (*ClearingSeat, error) {
	var bestSeat *ClearingSeat
	var minScore = 1e18

	for _, seat := range r.Seats {
		if seat.ExchangeCode == exchange && seat.Status == "ACTIVE" && seat.Capacity >= amount {
			// 简单评分：成本 * 1.0 + 延迟 * 0.5
			score := float64(seat.CostPerTrade)*100.0 + float64(seat.Latency)*0.5
			if score < minScore {
				minScore = score
				bestSeat = seat
			}
		}
	}

	if bestSeat == nil {
		return nil, fmt.Errorf("no available clearing seat for exchange %s", exchange)
	}
	return bestSeat, nil
}

// AvailableVolume 获取券库可用容量
func (p *SecurityPool) AvailableVolume() int64 {
	return p.TotalVolume - p.LentVolume
}

// Allocate 分配配额逻辑
func (p *SecurityPool) Allocate(qty int64) error {
	if p.AvailableVolume() < qty {
		return errors.New("insufficient security pool volume")
	}
	p.LentVolume += qty
	return nil
}

// Deallocate 释放配额
func (p *SecurityPool) Deallocate(qty int64) {
	p.LentVolume -= qty
	if p.LentVolume < 0 {
		p.LentVolume = 0
	}
}
