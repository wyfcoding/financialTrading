package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/execution/domain"
)

// ExecutionQueryService 处理所有执行相关的查询操作（Queries）。
type ExecutionQueryService struct {
	tradeRepo  domain.TradeRepository
	searchRepo domain.TradeSearchRepository
}

// NewExecutionQueryService 构造函数。
func NewExecutionQueryService(tradeRepo domain.TradeRepository, searchRepo domain.TradeSearchRepository) *ExecutionQueryService {
	return &ExecutionQueryService{
		tradeRepo:  tradeRepo,
		searchRepo: searchRepo,
	}
}

// GetExecutionHistory 获取执行历史 (按订单 ID)
func (q *ExecutionQueryService) GetExecutionHistory(ctx context.Context, orderID string) ([]*ExecutionDTO, error) {
	trade, err := q.tradeRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if trade == nil {
		return []*ExecutionDTO{}, nil
	}

	return []*ExecutionDTO{q.toDTO(trade)}, nil
}

// ListExecutions 获取指定用户的所有执行历史 (优先通过 ES 搜索)
func (q *ExecutionQueryService) ListExecutions(ctx context.Context, userID, symbol string, limit, offset int) ([]*ExecutionDTO, int64, error) {
	trades, total, err := q.searchRepo.Search(ctx, userID, symbol, limit, offset)
	if err != nil {
		// 回退到 MySQL (简单列表，不带复杂筛选)
		mysqlTrades, mysqlErr := q.tradeRepo.List(ctx, userID)
		if mysqlErr != nil {
			return nil, 0, mysqlErr
		}
		trades = mysqlTrades
		total = int64(len(mysqlTrades))
	}

	dtos := make([]*ExecutionDTO, 0, len(trades))
	for _, t := range trades {
		dtos = append(dtos, q.toDTO(t))
	}
	return dtos, total, nil
}

func (q *ExecutionQueryService) toDTO(t *domain.Trade) *ExecutionDTO {
	return &ExecutionDTO{
		ExecutionID: t.TradeID,
		OrderID:     t.OrderID,
		Symbol:      t.Symbol,
		Status:      "FILLED",
		ExecutedQty: t.ExecutedQuantity.String(),
		ExecutedPx:  t.ExecutedPrice.String(),
		Timestamp:   t.ExecutedAt.Unix(),
	}
}
