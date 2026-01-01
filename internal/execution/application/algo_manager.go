package application

import (
	"context"
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
				OrderType: "LIMIT",
				Quantity:  sliceQty.String(),
				Price:     "0",
			})
			if err != nil {
				logging.Error(ctx, "TWAP: failed to place child order", "algo_id", algo.AlgoID, "error", err)
				continue
			}

			// 更新执行进度
			algo.ExecutedQuantity = algo.ExecutedQuantity.Add(sliceQty)
			if err := m.repo.SaveAlgoOrder(ctx, algo); err != nil {
				logging.Error(ctx, "AlgoManager: failed to save algo order", "algo_id", algo.AlgoID, "error", err)
			}

			logging.Info(ctx, "TWAP slice executed", "algo_id", algo.AlgoID, "order_id", resp.Order.OrderId, "executed_qty", algo.ExecutedQuantity.String())
		case <-ctx.Done():
			logging.Info(ctx, "TWAP cancelled", "algo_id", algo.AlgoID)
			return
		}
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
	// VWAP 需要根据市场实时成交量和预设的参与率（Participation Rate）动态下单。
	logging.Info(ctx, "VWAP started", "algo_id", algo.AlgoID, "rate", algo.ParticipationRate.String())

	// 定时检测市场成交量并按比例跟随
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if algo.ExecutedQuantity.GreaterThanOrEqual(algo.TotalQuantity) {
				algo.Status = domain.ExecutionStatusCompleted
				if err := m.repo.SaveAlgoOrder(ctx, algo); err != nil {
					logging.Error(ctx, "AlgoManager: failed to save algo order", "algo_id", algo.AlgoID, "error", err)
				}
				return
			}

			// 获取最新市场成交量
			quote, err := m.marketDataCli.GetLatestQuote(ctx, &marketdatav1.GetLatestQuoteRequest{
				Symbol: algo.Symbol,
			})
			if err != nil {
				logging.Error(ctx, "VWAP: failed to get latest quote", "symbol", algo.Symbol, "error", err)
				continue
			}

			// 根据参与率计算本次应下单量：本次下单量 = 市场最近成交量 * 参与率
			marketVol := decimal.NewFromFloat(quote.LastSize)
			if marketVol.IsZero() {
				continue
			}

			sliceQty := marketVol.Mul(algo.ParticipationRate)

			// 确保不超过总剩余量
			remaining := algo.TotalQuantity.Sub(algo.ExecutedQuantity)
			if sliceQty.GreaterThan(remaining) {
				sliceQty = remaining
			}

			if sliceQty.IsZero() {
				continue
			}

			_, err = m.orderCli.CreateOrder(ctx, &orderv1.CreateOrderRequest{
				UserId:    algo.UserID,
				Symbol:    algo.Symbol,
				Side:      string(algo.Side),
				OrderType: "LIMIT",
				Quantity:  sliceQty.String(),
				Price:     "0",
			})

			if err == nil {
				algo.ExecutedQuantity = algo.ExecutedQuantity.Add(sliceQty)
				if err := m.repo.SaveAlgoOrder(ctx, algo); err != nil {
					logging.Error(ctx, "AlgoManager: failed to save algo order", "algo_id", algo.AlgoID, "error", err)
				}
			}
		case <-time.After(time.Until(algo.EndTime)):
			m.failAlgo(ctx, algo, "time window expired")
			return
		case <-ctx.Done():
			logging.Info(ctx, "VWAP cancelled", "algo_id", algo.AlgoID)
			return
		}
	}
}

func (m *AlgoManager) failAlgo(ctx context.Context, algo *domain.AlgoOrder, reason string) {
	algo.Status = domain.ExecutionStatusFailed
	if err := m.repo.SaveAlgoOrder(ctx, algo); err != nil {
		logging.Error(ctx, "AlgoManager: failed to save algo order", "algo_id", algo.AlgoID, "error", err)
	}
	logging.Error(ctx, "Algorithm failed", "algo_id", algo.AlgoID, "reason", reason)
}
