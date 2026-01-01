package application

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	orderv1 "github.com/wyfcoding/financialtrading/goapi/order/v1"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
	"github.com/wyfcoding/pkg/logging"
)

// AlgoManager 负责算法交易的策略调度和执行。
type AlgoManager struct {
	repo     domain.ExecutionRepository
	orderCli orderv1.OrderServiceClient
}

// NewAlgoManager 构造函数。
func NewAlgoManager(repo domain.ExecutionRepository, orderCli orderv1.OrderServiceClient) *AlgoManager {
	return &AlgoManager{
		repo:     repo,
		orderCli: orderCli,
	}
}

// Start 处理并开始一个算法订单。
func (m *AlgoManager) Start(ctx context.Context, algo *domain.AlgoOrder) {
	logging.Info(ctx, "AlgoManager: starting algorithm", "algo_id", algo.AlgoID, "type", algo.AlgoType)

	switch algo.AlgoType {
	case domain.AlgoTypeTWAP:
		go m.runTWAP(algo)
	case domain.AlgoTypeVWAP:
		go m.runVWAP(algo)
	default:
		logging.Error(ctx, "AlgoManager: unknown algorithm type", "type", algo.AlgoType)
	}
}

// runTWAP 执行时间加权平均价格算法。
func (m *AlgoManager) runTWAP(algo *domain.AlgoOrder) {
	ctx := context.Background()
	duration := algo.EndTime.Sub(algo.StartTime)
	if duration <= 0 {
		m.failAlgo(ctx, algo, "invalid time window")
		return
	}

	// 假设每 1 分钟执行一个切片
	interval := time.Minute
	slices := int(duration / interval)
	if slices <= 0 {
		slices = 1
	}

	sliceQty := algo.TotalQuantity.Div(decimal.NewFromInt(int64(slices)))
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logging.Info(ctx, "TWAP started", "algo_id", algo.AlgoID, "slices", slices, "slice_qty", sliceQty.String())

	for i := 0; i < slices; i++ {
		select {
		case <-ticker.C:
			// 下达子订单
			resp, err := m.orderCli.CreateOrder(ctx, &orderv1.CreateOrderRequest{
				UserId:    algo.UserID,
				Symbol:    algo.Symbol,
				Side:      string(algo.Side),
				OrderType: "LIMIT", // 修正字段名为 OrderType
				Quantity:  sliceQty.String(),
				Price:     "0",
			})
			if err != nil {
				logging.Error(ctx, "TWAP: failed to place child order", "algo_id", algo.AlgoID, "error", err)
				continue
			}

			// 更新执行进度
			algo.ExecutedQuantity = algo.ExecutedQuantity.Add(sliceQty)
			m.repo.SaveAlgoOrder(ctx, algo)

			logging.Info(ctx, "TWAP slice executed", "algo_id", algo.AlgoID, "order_id", resp.Order.OrderId, "executed_qty", algo.ExecutedQuantity.String())
		}
	}

	algo.Status = domain.ExecutionStatusCompleted
	m.repo.SaveAlgoOrder(ctx, algo)
	logging.Info(ctx, "TWAP completed", "algo_id", algo.AlgoID)
}

// runVWAP 执行成交量加权平均价格算法（简化版）。
func (m *AlgoManager) runVWAP(algo *domain.AlgoOrder) {
	// VWAP 通常需要订阅市场实时音量并根据参与率（Participation Rate）动态下单。
	// 这里仅实现核心逻辑框架。
	ctx := context.Background()
	logging.Info(ctx, "VWAP started (Simulated)", "algo_id", algo.AlgoID, "rate", algo.ParticipationRate.String())

	// 实际应用中会订阅 marketdata 实时流，并在成交量波动时按比例下单。
	// 此处模拟为定时检测。
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if algo.ExecutedQuantity.GreaterThanOrEqual(algo.TotalQuantity) {
				algo.Status = domain.ExecutionStatusCompleted
				m.repo.SaveAlgoOrder(ctx, algo)
				return
			}

			// 模拟计算：根据参与率计算本次应下单量
			// 假设全市场成交量为常数 1000
			marketVol := decimal.NewFromInt(1000)
			sliceQty := marketVol.Mul(algo.ParticipationRate)

			// 确保不超过总剩余量
			remaining := algo.TotalQuantity.Sub(algo.ExecutedQuantity)
			if sliceQty.GreaterThan(remaining) {
				sliceQty = remaining
			}

			_, err := m.orderCli.CreateOrder(ctx, &orderv1.CreateOrderRequest{
				UserId:    algo.UserID,
				Symbol:    algo.Symbol,
				Side:      string(algo.Side),
				OrderType: "LIMIT",
				Quantity:  sliceQty.String(),
				Price:     "0",
			})

			if err == nil {
				algo.ExecutedQuantity = algo.ExecutedQuantity.Add(sliceQty)
				m.repo.SaveAlgoOrder(ctx, algo)
			}
		case <-time.After(algo.EndTime.Sub(time.Now())):
			m.failAlgo(ctx, algo, "time window expired")
			return
		}
	}
}

func (m *AlgoManager) failAlgo(ctx context.Context, algo *domain.AlgoOrder, reason string) {
	algo.Status = domain.ExecutionStatusFailed
	m.repo.SaveAlgoOrder(ctx, algo)
	logging.Error(ctx, "Algorithm failed", "algo_id", algo.AlgoID, "reason", reason)
}
