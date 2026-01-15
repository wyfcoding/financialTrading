package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	marketdatav1 "github.com/wyfcoding/financialtrading/goapi/marketdata/v1"
	orderv1 "github.com/wyfcoding/financialtrading/goapi/order/v1"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
	"github.com/wyfcoding/pkg/logging"
)

// AlgoManager 负责算法交易的策略调度和执行。
type AlgoManager struct {
	repo          domain.ExecutionRepository
	orderCli      orderv1.OrderServiceClient
	marketDataCli marketdatav1.MarketDataServiceClient
	activeAlgos   map[string]context.CancelFunc
	mu            sync.RWMutex
}

// NewAlgoManager 构造函数。
func NewAlgoManager(repo domain.ExecutionRepository, orderCli orderv1.OrderServiceClient, marketDataCli marketdatav1.MarketDataServiceClient) *AlgoManager {
	return &AlgoManager{
		repo:          repo,
		orderCli:      orderCli,
		marketDataCli: marketDataCli,
		activeAlgos:   make(map[string]context.CancelFunc),
	}
}

// Start 处理并开始一个算法订单。
func (m *AlgoManager) Start(ctx context.Context, algo *domain.AlgoOrder) {
	m.mu.Lock()
	if _, exists := m.activeAlgos[algo.AlgoID]; exists {
		m.mu.Unlock()
		logging.Warn(ctx, "AlgoManager: algorithm already running", "algo_id", algo.AlgoID)
		return
	}

	algoCtx, cancel := context.WithCancel(context.Background())
	m.activeAlgos[algo.AlgoID] = cancel
	m.mu.Unlock()

	logging.Info(ctx, "AlgoManager: starting algorithm", "algo_id", algo.AlgoID, "type", algo.AlgoType)

	switch algo.AlgoType {
	case domain.AlgoTypeTWAP:
		go m.runTWAP(algoCtx, algo)
	case domain.AlgoTypeVWAP:
		go m.runVWAP(algoCtx, algo)
	default:
		m.removeAlgo(algo.AlgoID)
		logging.Error(ctx, "AlgoManager: unknown algorithm type", "type", algo.AlgoType)
	}
}

// Stop 停止一个正在运行的算法。
func (m *AlgoManager) Stop(ctx context.Context, algoID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cancel, ok := m.activeAlgos[algoID]; ok {
		cancel()
		delete(m.activeAlgos, algoID)
		logging.Info(ctx, "AlgoManager: algorithm stopped", "algo_id", algoID)
	}
}

func (m *AlgoManager) removeAlgo(algoID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.activeAlgos, algoID)
}

// runTWAP 执行时间加权平均价格算法。
func (m *AlgoManager) runTWAP(ctx context.Context, algo *domain.AlgoOrder) {
	defer m.removeAlgo(algo.AlgoID)

	// 构造母单对象
	parentOrder := &domain.ParentOrder{
		OrderID:       algo.AlgoID,
		Symbol:        algo.Symbol,
		Side:          string(algo.Side),
		TotalQuantity: algo.TotalQuantity,
		ExecutedQty:   algo.ExecutedQuantity,
		StrategyType:  domain.StrategyTWAP,
		StartTime:     algo.StartTime,
		EndTime:       algo.EndTime,
	}

	// 初始化 TWAP 策略
	strategy := &domain.TWAPStrategy{
		MinSliceSize: decimal.NewFromInt(100), // 示例：最小切片 100 单位
		Randomize:    true,                    // 启用随机扰动
	}

	// 生成切片计划
	slices, err := strategy.GenerateSlices(parentOrder)
	if err != nil {
		m.failAlgo(ctx, algo, fmt.Sprintf("failed to generate slices: %v", err))
		return
	}

	logging.Info(ctx, "TWAP plan generated", "algo_id", algo.AlgoID, "slices_count", len(slices))

	for _, slice := range slices {
		// 等待直到目标执行时间
		waitDuration := time.Until(slice.TargetTime)
		if waitDuration > 0 {
			select {
			case <-time.After(waitDuration):
			case <-ctx.Done():
				logging.Info(ctx, "TWAP cancelled", "algo_id", algo.AlgoID)
				return
			}
		}

		// 下达子订单
		resp, err := m.orderCli.CreateOrder(ctx, &orderv1.CreateOrderRequest{
			UserId:    algo.UserID,
			Symbol:    algo.Symbol,
			Side:      string(algo.Side),
			OrderType: "LIMIT",
			Quantity:  slice.Quantity.String(),
			Price:     "0", // 市价单或根据策略定价
		})
		if err != nil {
			logging.Error(ctx, "TWAP: failed to place child order", "algo_id", algo.AlgoID, "slice_id", slice.SliceID, "error", err)
			// 简单重试或跳过，实际生产需更复杂的错误处理
			continue
		}

		// 更新执行进度
		algo.ExecutedQuantity = algo.ExecutedQuantity.Add(slice.Quantity)
		if err := m.repo.SaveAlgoOrder(ctx, algo); err != nil {
			logging.Error(ctx, "AlgoManager: failed to save algo order", "algo_id", algo.AlgoID, "error", err)
		}

		logging.Info(ctx, "TWAP slice executed", "algo_id", algo.AlgoID, "order_id", resp.Order.OrderId, "slice_qty", slice.Quantity.String())
	}

	algo.Status = domain.ExecutionStatusCompleted
	if err := m.repo.SaveAlgoOrder(ctx, algo); err != nil {
		logging.Error(ctx, "AlgoManager: failed to save algo order", "algo_id", algo.AlgoID, "error", err)
	}
	logging.Info(ctx, "TWAP completed", "algo_id", algo.AlgoID)
}

// runVWAP 执行成交量加权平均价格算法。
func (m *AlgoManager) runVWAP(ctx context.Context, algo *domain.AlgoOrder) {
	defer m.removeAlgo(algo.AlgoID)

	// 构造母单对象
	parentOrder := &domain.ParentOrder{
		OrderID:       algo.AlgoID,
		Symbol:        algo.Symbol,
		Side:          string(algo.Side),
		TotalQuantity: algo.TotalQuantity,
		ExecutedQty:   algo.ExecutedQuantity,
		StrategyType:  domain.StrategyVWAP,
		StartTime:     algo.StartTime,
		EndTime:       algo.EndTime,
	}

	// 模拟历史成交量分布 (Volume Profile)
	// 实际生产中应从 MarketData 服务获取该 Symbol 的历史平均分布
	profile := []domain.VolumeProfileItem{
		{TimeSlot: "Start", Ratio: 0.1},
		{TimeSlot: "Early", Ratio: 0.3},
		{TimeSlot: "Mid", Ratio: 0.2},
		{TimeSlot: "Late", Ratio: 0.3},
		{TimeSlot: "Close", Ratio: 0.1},
	}

	strategy := &domain.VWAPStrategy{
		Profile: profile,
	}

	// 生成基于历史分布的切片计划
	slices, err := strategy.GenerateSlices(parentOrder)
	if err != nil {
		m.failAlgo(ctx, algo, fmt.Sprintf("failed to generate VWAP slices: %v", err))
		return
	}

	logging.Info(ctx, "VWAP plan generated", "algo_id", algo.AlgoID, "slices_count", len(slices))

	for _, slice := range slices {
		// 等待直到目标执行时间
		waitDuration := time.Until(slice.TargetTime)
		if waitDuration > 0 {
			select {
			case <-time.After(waitDuration):
			case <-ctx.Done():
				logging.Info(ctx, "VWAP cancelled", "algo_id", algo.AlgoID)
				return
			}
		}

		// 获取实时报价以确认市场深度 (可选，增强鲁棒性)
		_, err := m.marketDataCli.GetLatestQuote(ctx, &marketdatav1.GetLatestQuoteRequest{
			Symbol: algo.Symbol,
		})
		if err != nil {
			logging.Warn(ctx, "VWAP: failed to get latest quote, proceeding blindly", "symbol", algo.Symbol, "error", err)
		}

		// 下达子订单
		resp, err := m.orderCli.CreateOrder(ctx, &orderv1.CreateOrderRequest{
			UserId:    algo.UserID,
			Symbol:    algo.Symbol,
			Side:      string(algo.Side),
			OrderType: "LIMIT",
			Quantity:  slice.Quantity.String(),
			Price:     "0",
		})
		if err != nil {
			logging.Error(ctx, "VWAP: failed to place child order", "algo_id", algo.AlgoID, "slice_id", slice.SliceID, "error", err)
			continue
		}

		// 更新执行进度
		algo.ExecutedQuantity = algo.ExecutedQuantity.Add(slice.Quantity)
		if err := m.repo.SaveAlgoOrder(ctx, algo); err != nil {
			logging.Error(ctx, "AlgoManager: failed to save algo order", "algo_id", algo.AlgoID, "error", err)
		}

		logging.Info(ctx, "VWAP slice executed", "algo_id", algo.AlgoID, "order_id", resp.Order.OrderId, "slice_qty", slice.Quantity.String())
	}

	algo.Status = domain.ExecutionStatusCompleted
	if err := m.repo.SaveAlgoOrder(ctx, algo); err != nil {
		logging.Error(ctx, "AlgoManager: failed to save algo order", "algo_id", algo.AlgoID, "error", err)
	}
	logging.Info(ctx, "VWAP completed", "algo_id", algo.AlgoID)
}

func (m *AlgoManager) failAlgo(ctx context.Context, algo *domain.AlgoOrder, reason string) {
	algo.Status = domain.ExecutionStatusFailed
	if err := m.repo.SaveAlgoOrder(ctx, algo); err != nil {
		logging.Error(ctx, "AlgoManager: failed to save algo order", "algo_id", algo.AlgoID, "error", err)
	}
	logging.Error(ctx, "Algorithm failed", "algo_id", algo.AlgoID, "reason", reason)
}
