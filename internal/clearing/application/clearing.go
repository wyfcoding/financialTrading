package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
)

// ClearingService 清算门面服务，整合 Manager 和 Query。
type ClearingService struct {
	manager *ClearingManager
	query   *ClearingQuery
}

// NewClearingService 构造函数。
func NewClearingService(settlementRepo domain.SettlementRepository, eodRepo domain.EODClearingRepository) *ClearingService {
	return &ClearingService{
		manager: NewClearingManager(settlementRepo, eodRepo),
		query:   NewClearingQuery(settlementRepo, eodRepo),
	}
}

// --- Manager (Writes) ---

func (s *ClearingService) SettleTrade(ctx context.Context, req *SettleTradeRequest) (string, error) {
	return s.manager.SettleTrade(ctx, req)
}

func (s *ClearingService) ExecuteEODClearing(ctx context.Context, clearingDate string) (string, error) {
	return s.manager.ExecuteEODClearing(ctx, clearingDate)
}

// --- Query (Reads) ---

func (s *ClearingService) GetClearingStatus(ctx context.Context, clearingID string) (*domain.EODClearing, error) {
	return s.query.GetClearingStatus(ctx, clearingID)
}

func (s *ClearingService) GetSettlementHistory(ctx context.Context, userID string, limit, offset int) ([]*domain.Settlement, int64, error) {
	return s.query.GetSettlementHistory(ctx, userID, limit, offset)
}

// --- Legacy Compatibility Types ---

// SettleTradeRequest 是清算交易请求 DTO
type SettleTradeRequest struct {
	TradeID    string // 交易 ID
	BuyUserID  string // 买方用户 ID
	SellUserID string // 卖方用户 ID
	Symbol     string // 交易对符号
	Quantity   string // 数量
	Price      string // 价格
}
