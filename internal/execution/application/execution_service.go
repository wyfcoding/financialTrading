//go:build execution_experimental
// +build execution_experimental

// 变更说明：
// execution 执行服务融合了 sor (智能订单路由算法)。
// 完全接管金融系统的对外交易单分配。
package application

import (
	"context"
	"log/slog"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
	soralgo "github.com/wyfcoding/financialtrading/internal/execution/domain/sor"
)

type ExecutionApplicationService struct {
	repo         domain.ExecutionRepository
	sorOptimizer *soralgo.SOROptimizer
	logger       *slog.Logger
}

func NewExecutionApplicationService(r domain.ExecutionRepository, logger *slog.Logger) *ExecutionApplicationService {
	return &ExecutionApplicationService{
		repo:         r,
		sorOptimizer: &soralgo.SOROptimizer{LatencyFactor: 0.1},
		logger:       logger,
	}
}

type ExecuteRequest struct {
	OrderID string
	Symbol  string
	Qty     decimal.Decimal
	IsBuy   bool
	MaxCost decimal.Decimal // HFT 的容忍度指标
}

// RouteAndExecute 收到客户端/网关的单后，计算应分几个单给不同交易所
func (s *ExecutionApplicationService) RouteAndExecute(ctx context.Context, req *ExecuteRequest) error {
	// 1. 获取全市场的 Quote Depth
	// 这里会去 MarketData 服务或者直接读共享内存拉取各大交易所价格
	inputs := []soralgo.RouteInput{
		{VenueID: "EXCHANGE_A", Price: decimal.NewFromInt(100), Quantity: decimal.NewFromInt(50), FeeRate: decimal.NewFromFloat(0.001), LatencyMs: 2.5},
		{VenueID: "EXCHANGE_B", Price: decimal.NewFromInt(99), Quantity: decimal.NewFromInt(100), FeeRate: decimal.NewFromFloat(0.002), LatencyMs: 1.0},
		{VenueID: "DARK_POOL", Price: decimal.NewFromInt(98), Quantity: decimal.NewFromInt(1000), FeeRate: decimal.NewFromFloat(0.0), LatencyMs: 8.0},
	}

	// 2. 利用 pkg 的 SOR 算法求得最优解
	routes := s.sorOptimizer.Optimize(req.Qty, inputs, req.IsBuy)

	// 3. 落库或直接派发子订单发往 Fix Gateway 联通其他交易所
	for _, r := range routes {
		childOrder := &domain.ChildOrder{
			ParentOrderID: req.OrderID,
			VenueID:       r.VenueID,
			DispatchPrice: r.Price,
			DispatchQty:   r.Quantity,
			Status:        "ROUTED",
		}
		if err := s.repo.Save(ctx, childOrder); err != nil {
			s.logger.ErrorContext(ctx, "failed to record routed child order", "error", err)
			continue
		}
		// TODO: push to FIX Gateway via Kafka/gRPC
		s.logger.InfoContext(ctx, "child order routed", "venue", r.VenueID, "qty", r.Quantity.String())
	}
	return nil
}
