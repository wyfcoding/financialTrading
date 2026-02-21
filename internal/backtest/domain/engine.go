// 变更说明：
// 1. 【量化工具】历史数据 Tick/分笔回放核心。
// 2. 【仿真环境】提供带滑点与佣金的仿真匹配引擎（撮合器桩）。
// 3. 【绩效归因】自动计算夏普比率、最大回撤、总收益。
//go:build backtest_experimental
// +build backtest_experimental

package domain

import (
	"context"
	"errors"
	"math"
	"sort"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/timeseries"
)

var (
	ErrEngineAlreadyRunning = errors.New("backtest engine is already running")
	ErrNoData               = errors.New("no historical data loaded")
)

// CommissionModel 手续费模型
type CommissionModel interface {
	CalculateFee(price, quantity decimal.Decimal) decimal.Decimal
}

// FixedFeeModel 固定费率模型 (如 0.03%)
type FixedFeeModel struct {
	Rate decimal.Decimal
}

func (m *FixedFeeModel) CalculateFee(price, quantity decimal.Decimal) decimal.Decimal {
	return price.Mul(quantity).Mul(m.Rate)
}

// SlippageModel 滑点模型
type SlippageModel interface {
	ApplySlippage(price decimal.Decimal, isBuy bool) decimal.Decimal
}

// FixedSlippageModel 固定滑点跳数 (Tick Size)
type FixedSlippageModel struct {
	Ticks decimal.Decimal
}

func (m *FixedSlippageModel) ApplySlippage(price decimal.Decimal, isBuy bool) decimal.Decimal {
	if isBuy {
		return price.Add(m.Ticks)
	}
	return price.Sub(m.Ticks)
}

// Position 仿真持仓
type Position struct {
	Symbol   string
	Quantity decimal.Decimal
	AvgCost  decimal.Decimal
}

// Account 仿真账户
type Account struct {
	InitialBalance decimal.Decimal
	Balance        decimal.Decimal
	Positions      map[string]*Position
	Commission     decimal.Decimal
	EquityHistory  []EquityRecord
}

type EquityRecord struct {
	Timestamp int64
	Equity    decimal.Decimal
}

// SimulatedTrade 仿真交易回报记录
type SimulatedTrade struct {
	TradeID   int64
	Symbol    string
	IsBuy     bool
	Price     decimal.Decimal
	Quantity  decimal.Decimal
	Fee       decimal.Decimal
	Timestamp int64
}

// Strategy 用户的回测策略接口
type Strategy interface {
	// Initialize 初始时执行一次
	Initialize()
	// OnBar 有新的 K 线到达时触发（基于所订阅的周期）
	OnBar(bar timeseries.Bar, broker Broker)
	// OnTick 有逐笔数据到达时触发
	OnTick(tick timeseries.Tick, broker Broker)
}

// Broker 被 Strategy 调用以发出虚拟订单和查账户
type Broker interface {
	BuyMarket(symbol string, quantity decimal.Decimal)
	SellMarket(symbol string, quantity decimal.Decimal)
	BuyLimit(symbol string, price, quantity decimal.Decimal)
	SellLimit(symbol string, price, quantity decimal.Decimal)
	GetPositions() map[string]*Position
	GetBalance() decimal.Decimal
}

// EventType 数据流事件的类型
type EventType int

const (
	TypeBar  EventType = 1
	TypeTick EventType = 2
)

// DataEvent 通用的包装事件对象
type DataEvent struct {
	Type      EventType
	Timestamp int64
	Bar       timeseries.Bar
	Tick      timeseries.Tick
}

// BacktestEngine 回测引擎主循环
type BacktestEngine struct {
	account    *Account
	strategy   Strategy
	slippage   SlippageModel
	commission CommissionModel

	events      []DataEvent // 被预加载并按时间排序的历史数据流
	currentTime int64       // 当前仿真时间戳
	running     bool

	trades []SimulatedTrade // 回报列表
}

func NewBacktestEngine(initialCapital decimal.Decimal, st Strategy, slippage SlippageModel, comm CommissionModel) *BacktestEngine {
	return &BacktestEngine{
		account: &Account{
			InitialBalance: initialCapital,
			Balance:        initialCapital,
			Positions:      make(map[string]*Position),
		},
		strategy:   st,
		slippage:   slippage,
		commission: comm,
		events:     make([]DataEvent, 0),
	}
}

// LoadData 装载行情数据并排序
func (e *BacktestEngine) LoadData(events []DataEvent) {
	e.events = append(e.events, events...)
	sort.Slice(e.events, func(i, j int) bool {
		return e.events[i].Timestamp < e.events[j].Timestamp
	})
}

// Run 启动事件循环驱动策略
func (e *BacktestEngine) Run(ctx context.Context) error {
	if e.running {
		return ErrEngineAlreadyRunning
	}
	if len(e.events) == 0 {
		return ErrNoData
	}

	e.running = true
	defer func() { e.running = false }()

	e.strategy.Initialize()

	for _, event := range e.events {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		e.currentTime = event.Timestamp
		e.updateEquityRecord()

		// 分发给用户策略，传入自己做 Broker 实现
		if event.Type == TypeBar {
			e.strategy.OnBar(event.Bar, e)
		} else if event.Type == TypeTick {
			e.strategy.OnTick(event.Tick, e)
		}

		// 真实框架下此处需要遍历未成交的限价单并通过 event 的 High/Low 判断是否被成交触发 (Pending Orders Matching)
		// 此处简化，假设只有市价且立刻成交
	}

	return nil
}

// ----------------- Broker Interface Impl -----------------

// getMarketPrice 获取当前仿真时间的市场价格，回测一般用最新 Bar 的 Close，或者最新 Tick 的 Price (为了严谨)
// 此处我们仅抛砖引玉实现快速撮合
func (e *BacktestEngine) getMarketPrice(symbol string) decimal.Decimal {
	// (需通过缓存上一笔数据的价格，此处用 100 占位)
	return decimal.NewFromInt(100)
}

func (e *BacktestEngine) executeMarketOrder(symbol string, isBuy bool, quantity decimal.Decimal) {
	basePrice := e.getMarketPrice(symbol)

	// 计算滑点
	execPrice := basePrice
	if e.slippage != nil {
		execPrice = e.slippage.ApplySlippage(basePrice, isBuy)
	}

	// 计算手续费
	fee := decimal.Zero
	if e.commission != nil {
		fee = e.commission.CalculateFee(execPrice, quantity)
	}

	totalValue := execPrice.Mul(quantity)

	// 账户扣/加钱 (买扣余额，卖加余额)，忽略保证金控制
	if isBuy {
		e.account.Balance = e.account.Balance.Sub(totalValue).Sub(fee)
	} else {
		e.account.Balance = e.account.Balance.Add(totalValue).Sub(fee)
	}
	e.account.Commission = e.account.Commission.Add(fee)

	// 更新持仓成本
	pos, exists := e.account.Positions[symbol]
	if !exists {
		pos = &Position{Symbol: symbol, Quantity: decimal.Zero, AvgCost: decimal.Zero}
		e.account.Positions[symbol] = pos
	}

	if isBuy {
		// (原有量 * 成本 + 本次量 * 价格) / 总量
		totalCost := pos.AvgCost.Mul(pos.Quantity).Add(execPrice.Mul(quantity))
		pos.Quantity = pos.Quantity.Add(quantity)
		if pos.Quantity.IsPositive() {
			pos.AvgCost = totalCost.Div(pos.Quantity)
		}
	} else { // 卖出 (不改变 AvgCost，仅减去数量，可能转为空头 Short)
		pos.Quantity = pos.Quantity.Sub(quantity)
		if pos.Quantity.IsPositive() {
			// 部分平仓或持有净空头，简化起见不改变平均成本模型
		} else if pos.Quantity.IsNegative() {
			// 直接按新做空均价建立
			pos.AvgCost = execPrice
		} else {
			pos.AvgCost = decimal.Zero
		}
	}

	e.trades = append(e.trades, SimulatedTrade{
		TradeID:   int64(len(e.trades) + 1),
		Symbol:    symbol,
		IsBuy:     isBuy,
		Price:     execPrice,
		Quantity:  quantity,
		Fee:       fee,
		Timestamp: e.currentTime,
	})
}

func (e *BacktestEngine) BuyMarket(symbol string, quantity decimal.Decimal) {
	e.executeMarketOrder(symbol, true, quantity)
}

func (e *BacktestEngine) SellMarket(symbol string, quantity decimal.Decimal) {
	e.executeMarketOrder(symbol, false, quantity)
}

func (e *BacktestEngine) BuyLimit(symbol string, price, quantity decimal.Decimal) { /* TODO 挂单逻辑 */
}
func (e *BacktestEngine) SellLimit(symbol string, price, quantity decimal.Decimal) { /* TODO 挂单逻辑 */
}
func (e *BacktestEngine) GetPositions() map[string]*Position { return e.account.Positions }
func (e *BacktestEngine) GetBalance() decimal.Decimal        { return e.account.Balance }

// ----------------- Report & Analytics -----------------

func (e *BacktestEngine) updateEquityRecord() {
	equity := e.account.Balance
	// 把所有未平仓持仓根据当前价格计入净值 (MtM: Mark to Market)
	for symbol, pos := range e.account.Positions {
		if !pos.Quantity.IsZero() {
			price := e.getMarketPrice(symbol)
			posVal := pos.Quantity.Mul(price)
			equity = equity.Add(posVal)
		}
	}
	e.account.EquityHistory = append(e.account.EquityHistory, EquityRecord{Timestamp: e.currentTime, Equity: equity})
}

// AnalyticsReport 评价报告
type AnalyticsReport struct {
	TotalReturn   float64 // 净值总收益率
	MaxDrawdown   float64 // 最大回撤绝对百分比
	SharpeRatio   float64 // 夏普比率 (针对无通胀假设的均值-方差评价)
	TotalTrades   int
	TotalFeesPaid float64
}

// GenerateReport 基于权益历史曲线生成绩效评价
func (e *BacktestEngine) GenerateReport() *AnalyticsReport {
	if len(e.account.EquityHistory) == 0 {
		return nil
	}

	initialEq, _ := e.account.InitialBalance.Float64()
	finalEq, _ := e.account.EquityHistory[len(e.account.EquityHistory)-1].Equity.Float64()
	totalReturn := (finalEq - initialEq) / initialEq

	maxPeak := initialEq
	maxDD := 0.0

	// 用于算日夏普
	returns := make([]float64, 0)
	lastEq := initialEq

	for _, record := range e.account.EquityHistory {
		eq, _ := record.Equity.Float64()
		if eq > maxPeak {
			maxPeak = eq
		}
		dd := (maxPeak - eq) / maxPeak
		if dd > maxDD {
			maxDD = dd
		}

		ret := (eq - lastEq) / lastEq
		if ret != 0 {
			returns = append(returns, ret)
		}
		lastEq = eq
	}

	var sum, sumSq float64
	for _, r := range returns {
		sum += r
	}
	n := float64(len(returns))
	var avgRet float64
	if n > 0 {
		avgRet = sum / n
	}
	for _, r := range returns {
		diff := r - avgRet
		sumSq += diff * diff
	}
	var stdDev float64
	if n > 0 {
		stdDev = math.Sqrt(sumSq / n)
	}

	sharpe := 0.0
	// 粗略按单笔时间刻度近似转为年化 (假设 252 交易日，每日采样这种)
	if stdDev > 0 {
		sharpe = (avgRet / stdDev) * math.Sqrt(252.0)
	}

	feePaid, _ := e.account.Commission.Float64()

	return &AnalyticsReport{
		TotalReturn:   totalReturn,
		MaxDrawdown:   maxDD,
		SharpeRatio:   sharpe,
		TotalTrades:   len(e.trades),
		TotalFeesPaid: feePaid,
	}
}
