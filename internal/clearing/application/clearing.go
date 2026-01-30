package application

import (
	"context"
)

// ClearingService 作为清算操作的门面。
type ClearingService struct {
	Command *ClearingCommandService
	Query   *ClearingQueryService
}

// NewClearingService 创建清算服务门面实例。
func NewClearingService(command *ClearingCommandService, query *ClearingQueryService) *ClearingService {
	return &ClearingService{
		Command: command,
		Query:   query,
	}
}

// --- 写操作（委托给 Command） ---

func (s *ClearingService) SettleTrade(ctx context.Context, req *SettleTradeRequest) (*SettlementDTO, error) {
	return s.Command.SettleTrade(ctx, req)
}

func (s *ClearingService) ExecuteEODClearing(ctx context.Context, clearingDate string) (string, error) {
	return s.Command.ExecuteEODClearing(ctx, clearingDate)
}

func (s *ClearingService) RunLiquidationCheck(ctx context.Context, userID string) error {
	return s.Command.RunLiquidationCheck(ctx, userID)
}

func (s *ClearingService) SagaMarkSettlementCompleted(ctx context.Context, settlementID string) error {
	return s.Command.SagaMarkSettlementCompleted(ctx, settlementID)
}

func (s *ClearingService) SagaMarkSettlementFailed(ctx context.Context, settlementID, reason string) error {
	return s.Command.SagaMarkSettlementFailed(ctx, settlementID, reason)
}

// --- 读操作（委托给 Query） ---

func (s *ClearingService) GetSettlement(ctx context.Context, id string) (*SettlementDTO, error) {
	return s.Query.GetSettlement(ctx, id)
}
