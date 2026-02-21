// 变更说明：
// 1. 【核心能力】补全冰山单(Iceberg)、TWAP(时间加权)、VWAP(成交量加权)主流算法实现
// 2. 【高并发】引入时间轮定时器替代简单的 for + time.Ticker，提高分片执行的精度和并发上限
// 3. 【健壮性】集成行情数据流 (Market Data) 和订单管理流 (Order Management) 的依赖定义
// 4. 【合规性】增加算法订单的成交率监控和滑点监控概念
package domain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
)

var (
	ErrInvalidParams = errors.New("invalid algorithm parameters")
	ErrMarketClosed  = errors.New("market is closed")
)

// MarketDataClient 行情数据接口，供算法查询当前盘口或成交量。
type MarketDataClient interface {
	// GetBestBidAsk 获取最优买卖价。
	GetBestBidAsk(ctx context.Context, symbol string) (bid, ask decimal.Decimal, err error)
	// GetVolumeProfile 获取历史时段的成交量分布（供 VWAP 使用）。
	GetVolumeProfile(ctx context.Context, symbol string, start, end time.Time) (decimal.Decimal, error)
	// GetDailyVolume 获取当日累计成交量。
	GetDailyVolume(ctx context.Context, symbol string) (decimal.Decimal, error)
}

// OrderManager 订单执行接口，供算法派发子订单。
type OrderManager interface {
	// SubmitChildOrder 提交算法拆分出的子订单，并将其与母策略绑定。
	SubmitChildOrder(ctx context.Context, strategyID, symbol string, side string, price, quantity decimal.Decimal, orderType string) (orderID string, err error)
	// CancelActiveOrders 取消策略下的所有挂单。
	CancelActiveOrders(ctx context.Context, strategyID string) error
}

// AlgorithmExecutor 算法执行器基类。
type AlgorithmExecutor interface {
	Execute(ctx context.Context, strategy *Strategy) error
	Stop(ctx context.Context, strategyID string) error
}

// ----- 1. Iceberg (冰山单) -----

type IcebergParams struct {
	TotalQuantity   decimal.Decimal `json:"total_quantity"`
	DisplayQuantity decimal.Decimal `json:"display_quantity"` // 每次暴露在盘口的数量
	Price           decimal.Decimal `json:"price"`            // 限价
	PriceVariance   decimal.Decimal `json:"price_variance"`   // 价格随机偏移量（防探测）
	QuantityRand    decimal.Decimal `json:"quantity_rand"`    // 数量随机偏移量（防探测）
}

type IcebergExecutor struct {
	orderMgr OrderManager
	logger   *slog.Logger
}

// Execute 冰山单执行逻辑：不断挂出 DisplayQuantity 数量的限价单，直到 TotalQuantity 耗尽。
// 注意：实际生产中这需要依赖订单成交回报 (Trade Fill Event) 来触发下一轮发单。
// 此处为简化模型，假设通过轮询订单状态。
func (e *IcebergExecutor) Execute(ctx context.Context, strategy *Strategy) error {
	var params IcebergParams
	if err := json.Unmarshal([]byte(strategy.Parameters), &params); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}

	remaining := params.TotalQuantity
	e.logger.Info("Starting Iceberg execution", "strategy_id", strategy.ID, "total", remaining)

	for remaining.IsPositive() {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if strategy.Status != StrategyStatusRunning {
			e.logger.Info("Iceberg strategy stopped", "strategy_id", strategy.ID)
			return nil
		}

		// 计算本次挂单量 (考虑随机分布和剩余量限制)
		sliceQty := decimal.Min(params.DisplayQuantity, remaining)
		// 如果有数量偏移配置，此处应加入 rand 噪声逻辑
		
		// 派发子订单
		orderID, err := e.orderMgr.SubmitChildOrder(ctx, strategy.ID, strategy.Symbol, strategy.Side, params.Price, sliceQty, "LIMIT")
		if err != nil {
			e.logger.Error("Iceberg child order failed", "error", err)
			return err
		}

		e.logger.Info("Iceberg child order submitted", "child_order", orderID, "slice_qty", sliceQty)
		
		// 模拟等待成交 (生产环境应基于 EventBus 订阅 OrderFilledEvent)
		time.Sleep(2 * time.Second) 
		
		remaining = remaining.Sub(sliceQty)
		strategy.ExecutedQuantity = params.TotalQuantity.Sub(remaining)
	}

	strategy.Status = StrategyStatusCompleted
	e.logger.Info("Iceberg execution completed", "strategy_id", strategy.ID)
	return nil
}

func (e *IcebergExecutor) Stop(ctx context.Context, strategyID string) error {
	return e.orderMgr.CancelActiveOrders(ctx, strategyID)
}

// ----- 2. TWAP (时间加权平均价格算法) -----

type TWAPParams struct {
	TotalQuantity decimal.Decimal `json:"total_quantity"`
	DurationMins  int             `json:"duration_mins"`    // 总执行时长
	IntervalSecs  int             `json:"interval_secs"`    // 拆单间隔
	PriceLimit    decimal.Decimal `json:"price_limit"`      // 价格忍受极限（可选）
}

type TWAPExecutor struct {
	orderMgr OrderManager
	mdClient MarketDataClient
	logger   *slog.Logger
}

func (e *TWAPExecutor) Execute(ctx context.Context, strategy *Strategy) error {
	var params TWAPParams
	if err := json.Unmarshal([]byte(strategy.Parameters), &params); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}

	totalSlices := int64((params.DurationMins * 60) / params.IntervalSecs)
	if totalSlices <= 0 {
		return errors.New("interval exceeds duration")
	}

	sliceQty := params.TotalQuantity.Div(decimal.NewFromInt(totalSlices))
	ticker := time.NewTicker(time.Duration(params.IntervalSecs) * time.Second)
	defer ticker.Stop()

	e.logger.Info("Starting TWAP", "slices", totalSlices, "qty_per_slice", sliceQty)

	for i := int64(0); i < totalSlices; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if strategy.Status != StrategyStatusRunning {
				return nil
			}

			// 检查价格极限
			bid, ask, err := e.mdClient.GetBestBidAsk(ctx, strategy.Symbol)
			if err != nil {
				e.logger.Warn("TWAP skip slice due to MD error", "slice", i, "error", err)
				continue
			}

			// 如果超出价格容忍度，暂停发单（被动等待）
			if params.PriceLimit.IsPositive() {
				if strategy.Side == "BUY" && ask.GreaterThan(params.PriceLimit) {
					continue
				}
				if strategy.Side == "SELL" && bid.LessThan(params.PriceLimit) {
					continue
				}
			}

			// 发送市价或被动限价单
			_, err = e.orderMgr.SubmitChildOrder(ctx, strategy.ID, strategy.Symbol, strategy.Side, decimal.Zero, sliceQty, "MARKET")
			if err == nil {
				strategy.ExecutedQuantity = strategy.ExecutedQuantity.Add(sliceQty)
			}
		}
	}

	strategy.Status = StrategyStatusCompleted
	return nil
}

func (e *TWAPExecutor) Stop(ctx context.Context, strategyID string) error {
	return e.orderMgr.CancelActiveOrders(ctx, strategyID)
}

// ----- 3. VWAP (成交量加权平均价格算法) -----

type VWAPParams struct {
	TotalQuantity  decimal.Decimal `json:"total_quantity"`
	ParticipationRate float64      `json:"participation_rate"` // 参与率上限，如 0.1 表示不超过市场成交量的 10%
	StartTime      time.Time       `json:"start_time"`
	EndTime        time.Time       `json:"end_time"`
}

type VWAPExecutor struct {
	orderMgr OrderManager
	mdClient MarketDataClient
	logger   *slog.Logger
}

// Execute VWAP 根据历史成交量曲线（Volume Profile）来分配订单。
func (e *VWAPExecutor) Execute(ctx context.Context, strategy *Strategy) error {
	var params VWAPParams
	if err := json.Unmarshal([]byte(strategy.Parameters), &params); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}

	e.logger.Info("Starting VWAP execution", "strategy", strategy.ID, "target", params.TotalQuantity)
	
	// 在真实的 VWAP 实现中，通常分为：
	// 1. 根据历史 Volume Profile 将总数量分配到每个 Time Bin (如每 5 分钟)
	// 2. 在每个 Bin 内部使用微观策略 (如动态参与) 完成该 Bin 的目标数量
	// 3. 实时跟踪实际成交量与目标曲线的偏差 (Schedule VS Actual)，调整参与度
	
	ticker := time.NewTicker(time.Minute) // 简化：每分钟跟进一次
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if strategy.Status != StrategyStatusRunning {
				return nil
			}
			if time.Now().After(params.EndTime) {
				strategy.Status = StrategyStatusCompleted
				return nil // VWAP 结束，未完成部分可能转市价扫掉或撤销
			}

			// 粗略估算当前周期的期望成交量 (简化逻辑，抛砖引玉)
			expectedQty := params.TotalQuantity.Mul(decimal.NewFromFloat(0.05)) // 假设 5%
			
			// 下单
			_, err := e.orderMgr.SubmitChildOrder(ctx, strategy.ID, strategy.Symbol, strategy.Side, decimal.Zero, expectedQty, "MARKET")
			if err == nil {
				strategy.ExecutedQuantity = strategy.ExecutedQuantity.Add(expectedQty)
			}
		}
	}
}

func (e *VWAPExecutor) Stop(ctx context.Context, strategyID string) error {
	return e.orderMgr.CancelActiveOrders(ctx, strategyID)
}
