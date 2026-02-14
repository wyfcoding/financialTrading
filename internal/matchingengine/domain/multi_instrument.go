package domain

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/wyfcoding/pkg/algorithm/types"

	"github.com/shopspring/decimal"
)

type InstrumentType string

const (
	InstrumentTypeStock  InstrumentType = "STOCK"
	InstrumentTypeFuture InstrumentType = "FUTURE"
	InstrumentTypeOption InstrumentType = "OPTION"
	InstrumentTypeETF    InstrumentType = "ETF"
	InstrumentTypeBond   InstrumentType = "BOND"
	InstrumentTypeForex  InstrumentType = "FOREX"
	InstrumentTypeCrypto InstrumentType = "CRYPTO"
)

type Instrument struct {
	Symbol            string            `json:"symbol"`
	Name              string            `json:"name"`
	Type              InstrumentType    `json:"type"`
	BaseCurrency      string            `json:"base_currency"`
	QuoteCurrency     string            `json:"quote_currency"`
	TickSize          decimal.Decimal   `json:"tick_size"`
	LotSize           decimal.Decimal   `json:"lot_size"`
	MinOrderQty       decimal.Decimal   `json:"min_order_qty"`
	MaxOrderQty       decimal.Decimal   `json:"max_order_qty"`
	PriceMultiplier   decimal.Decimal   `json:"price_multiplier"`
	ContractSize      decimal.Decimal   `json:"contract_size"`
	Underlying        string            `json:"underlying,omitempty"`
	StrikePrice       decimal.Decimal   `json:"strike_price"`
	ExpiryDate        *time.Time        `json:"expiry_date,omitempty"`
	OptionType        string            `json:"option_type,omitempty"`
	SettlementType    string            `json:"settlement_type"`
	TradingHours      *TradingHours     `json:"trading_hours,omitempty"`
	MarginRequirement *MarginConfig     `json:"margin_requirement,omitempty"`
	PriceLimits       *PriceLimitConfig `json:"price_limits,omitempty"`
	Status            string            `json:"status"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

type TradingHours struct {
	RegularHours []TimeRange `json:"regular_hours"`
	PreMarket    []TimeRange `json:"pre_market,omitempty"`
	AfterHours   []TimeRange `json:"after_hours,omitempty"`
	Timezone     string      `json:"timezone"`
}

type TimeRange struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type MarginConfig struct {
	InitialMargin     decimal.Decimal `json:"initial_margin"`
	MaintenanceMargin decimal.Decimal `json:"maintenance_margin"`
	LongMarginRate    decimal.Decimal `json:"long_margin_rate"`
	ShortMarginRate   decimal.Decimal `json:"short_margin_rate"`
}

type PriceLimitConfig struct {
	UpLimitRate    decimal.Decimal `json:"up_limit_rate"`
	DownLimitRate  decimal.Decimal `json:"down_limit_rate"`
	ReferencePrice decimal.Decimal `json:"reference_price"`
}

type MultiInstrumentMatchingEngine struct {
	engines     map[string]*DisruptionEngine
	instruments map[string]*Instrument
	mu          sync.RWMutex
	logger      any
}

func NewMultiInstrumentMatchingEngine() *MultiInstrumentMatchingEngine {
	return &MultiInstrumentMatchingEngine{
		engines:     make(map[string]*DisruptionEngine),
		instruments: make(map[string]*Instrument),
	}
}

func (m *MultiInstrumentMatchingEngine) AddInstrument(instrument *Instrument, capacity uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.instruments[instrument.Symbol]; exists {
		return fmt.Errorf("instrument %s already exists", instrument.Symbol)
	}

	engine, err := NewDisruptionEngine(instrument.Symbol, capacity, nil)
	if err != nil {
		return fmt.Errorf("failed to create engine for %s: %w", instrument.Symbol, err)
	}

	m.engines[instrument.Symbol] = engine
	m.instruments[instrument.Symbol] = instrument

	return engine.Start()
}

func (m *MultiInstrumentMatchingEngine) RemoveInstrument(symbol string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	engine, exists := m.engines[symbol]
	if !exists {
		return fmt.Errorf("instrument %s not found", symbol)
	}

	engine.Shutdown()
	delete(m.engines, symbol)
	delete(m.instruments, symbol)

	return nil
}

func (m *MultiInstrumentMatchingEngine) GetEngine(symbol string) (*DisruptionEngine, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	engine, exists := m.engines[symbol]
	if !exists {
		return nil, fmt.Errorf("engine for %s not found", symbol)
	}
	return engine, nil
}

func (m *MultiInstrumentMatchingEngine) GetInstrument(symbol string) (*Instrument, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instrument, exists := m.instruments[symbol]
	if !exists {
		return nil, fmt.Errorf("instrument %s not found", symbol)
	}
	return instrument, nil
}

func (m *MultiInstrumentMatchingEngine) SubmitOrder(order *types.Order) (*MatchingResult, error) {
	engine, err := m.GetEngine(order.Symbol)
	if err != nil {
		return nil, err
	}

	instrument, err := m.GetInstrument(order.Symbol)
	if err != nil {
		return nil, err
	}

	if err := m.validateOrder(order, instrument); err != nil {
		return nil, err
	}

	adjustedOrder := m.adjustOrderForInstrument(order, instrument)
	return engine.SubmitOrder(adjustedOrder)
}

func (m *MultiInstrumentMatchingEngine) validateOrder(order *types.Order, instrument *Instrument) error {
	if order.Quantity.LessThan(instrument.MinOrderQty) {
		return fmt.Errorf("order quantity %s less than minimum %s", order.Quantity, instrument.MinOrderQty)
	}

	if instrument.MaxOrderQty.GreaterThan(decimal.Zero) && order.Quantity.GreaterThan(instrument.MaxOrderQty) {
		return fmt.Errorf("order quantity %s greater than maximum %s", order.Quantity, instrument.MaxOrderQty)
	}

	if !m.isValidPrice(order.Price, instrument.TickSize) {
		return fmt.Errorf("price %s is not valid for tick size %s", order.Price, instrument.TickSize)
	}

	return nil
}

func (m *MultiInstrumentMatchingEngine) isValidPrice(price, tickSize decimal.Decimal) bool {
	if tickSize.IsZero() {
		return true
	}
	remainder := price.Mod(tickSize)
	return remainder.IsZero()
}

func (m *MultiInstrumentMatchingEngine) adjustOrderForInstrument(order *types.Order, instrument *Instrument) *types.Order {
	adjusted := *order

	if instrument.Type == InstrumentTypeFuture {
		adjusted.Quantity = adjusted.Quantity.Mul(instrument.ContractSize)
	}

	return &adjusted
}

func (m *MultiInstrumentMatchingEngine) GetAllSymbols() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	symbols := make([]string, 0, len(m.engines))
	for symbol := range m.engines {
		symbols = append(symbols, symbol)
	}
	return symbols
}

func (m *MultiInstrumentMatchingEngine) GetAllSnapshots(depth int) map[string]*OrderBookSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshots := make(map[string]*OrderBookSnapshot)
	for symbol, engine := range m.engines {
		snapshots[symbol] = engine.GetOrderBookSnapshot(depth)
	}
	return snapshots
}

type OptionMatchingEngine struct {
	*DisruptionEngine
	instrument       *Instrument
	underlyingEngine *DisruptionEngine
	greeksCalculator *GreeksCalculator
}

func NewOptionMatchingEngine(instrument *Instrument, capacity uint64, underlyingEngine *DisruptionEngine) (*OptionMatchingEngine, error) {
	engine, err := NewDisruptionEngine(instrument.Symbol, capacity, nil)
	if err != nil {
		return nil, err
	}

	return &OptionMatchingEngine{
		DisruptionEngine: engine,
		instrument:       instrument,
		underlyingEngine: underlyingEngine,
		greeksCalculator: NewGreeksCalculator(),
	}, nil
}

func (e *OptionMatchingEngine) CalculateGreeks(spotPrice, riskFreeRate, volatility decimal.Decimal) *Greeks {
	return e.greeksCalculator.Calculate(
		spotPrice,
		e.instrument.StrikePrice,
		e.getTimeToExpiry(),
		riskFreeRate,
		volatility,
		e.instrument.OptionType,
	)
}

func (e *OptionMatchingEngine) getTimeToExpiry() decimal.Decimal {
	if e.instrument.ExpiryDate == nil {
		return decimal.Zero
	}

	now := time.Now()
	years := e.instrument.ExpiryDate.Sub(now).Hours() / (365.0 * 24.0)
	return decimal.NewFromFloat(years)
}

type Greeks struct {
	Delta decimal.Decimal `json:"delta"`
	Gamma decimal.Decimal `json:"gamma"`
	Theta decimal.Decimal `json:"theta"`
	Vega  decimal.Decimal `json:"vega"`
	Rho   decimal.Decimal `json:"rho"`
	IV    decimal.Decimal `json:"iv"`
}

type GreeksCalculator struct{}

func NewGreeksCalculator() *GreeksCalculator {
	return &GreeksCalculator{}
}

func (c *GreeksCalculator) Calculate(spotPrice, strikePrice, timeToExpiry, riskFreeRate, volatility decimal.Decimal, optionType string) *Greeks {
	greeks := &Greeks{}

	if timeToExpiry.IsZero() || volatility.IsZero() {
		return greeks
	}

	d1 := c.calculateD1(spotPrice, strikePrice, timeToExpiry, riskFreeRate, volatility)

	if optionType == "CALL" {
		greeks.Delta = c.normalCDF(d1)
	} else {
		greeks.Delta = c.normalCDF(d1).Sub(decimal.NewFromInt(1))
	}

	sqrtT := decimal.NewFromFloat(math.Sqrt(timeToExpiry.InexactFloat64()))
	greeks.Gamma = c.normalPDF(d1).Div(spotPrice.Mul(volatility).Mul(sqrtT))

	return greeks
}

func (c *GreeksCalculator) calculateD1(spotPrice, strikePrice, timeToExpiry, riskFreeRate, volatility decimal.Decimal) decimal.Decimal {
	logMoneyness, _ := spotPrice.Div(strikePrice).Ln(16)
	sqrtT := decimal.NewFromFloat(math.Sqrt(timeToExpiry.InexactFloat64()))

	numerator := logMoneyness.Add(
		riskFreeRate.Add(volatility.Mul(volatility).Div(decimal.NewFromInt(2))).Mul(timeToExpiry),
	)
	denominator := volatility.Mul(sqrtT)
	return numerator.Div(denominator)
}

func (c *GreeksCalculator) calculateD2(d1, timeToExpiry, volatility decimal.Decimal) decimal.Decimal {
	sqrtT := decimal.NewFromFloat(math.Sqrt(timeToExpiry.InexactFloat64()))
	return d1.Sub(volatility.Mul(sqrtT))
}

func (c *GreeksCalculator) normalCDF(x decimal.Decimal) decimal.Decimal {
	return decimal.NewFromFloat(0.5).Add(
		decimal.NewFromFloat(0.5).Mul(c.erf(x.Div(decimal.NewFromFloat(1.41421356237)))),
	)
}

func (c *GreeksCalculator) normalPDF(x decimal.Decimal) decimal.Decimal {
	xFloat := x.InexactFloat64()
	exp := math.Exp(-0.5 * xFloat * xFloat)
	return decimal.NewFromFloat(exp / math.Sqrt(2*math.Pi))
}

func (c *GreeksCalculator) erf(x decimal.Decimal) decimal.Decimal {
	return decimal.NewFromFloat(0.5)
}

type FutureMatchingEngine struct {
	*DisruptionEngine
	instrument *Instrument
	settlement *FutureSettlement
}

func NewFutureMatchingEngine(instrument *Instrument, capacity uint64) (*FutureMatchingEngine, error) {
	engine, err := NewDisruptionEngine(instrument.Symbol, capacity, nil)
	if err != nil {
		return nil, err
	}

	return &FutureMatchingEngine{
		DisruptionEngine: engine,
		instrument:       instrument,
		settlement:       NewFutureSettlement(instrument),
	}, nil
}

func (e *FutureMatchingEngine) CalculateMargin(position *Position, price decimal.Decimal) *MarginRequirement {
	return e.settlement.CalculateMargin(position, price)
}

func (e *FutureMatchingEngine) MarkToMarket(position *Position, settlementPrice decimal.Decimal) *MTMResult {
	return e.settlement.MarkToMarket(position, settlementPrice)
}

type FutureSettlement struct {
	instrument *Instrument
}

func NewFutureSettlement(instrument *Instrument) *FutureSettlement {
	return &FutureSettlement{instrument: instrument}
}

func (s *FutureSettlement) CalculateMargin(position *Position, price decimal.Decimal) *MarginRequirement {
	notional := position.Quantity.Abs().Mul(price).Mul(s.instrument.ContractSize)

	initialMargin := notional.Mul(s.instrument.MarginRequirement.InitialMargin)
	maintenanceMargin := notional.Mul(s.instrument.MarginRequirement.MaintenanceMargin)

	return &MarginRequirement{
		InitialMargin:     initialMargin,
		MaintenanceMargin: maintenanceMargin,
	}
}

func (s *FutureSettlement) MarkToMarket(position *Position, settlementPrice decimal.Decimal) *MTMResult {
	quantity := position.Quantity
	if quantity.IsZero() {
		return &MTMResult{}
	}

	pnl := quantity.Mul(settlementPrice.Sub(position.AvgPrice)).Mul(s.instrument.ContractSize)

	return &MTMResult{
		PositionID:      position.ID,
		SettlementPrice: settlementPrice,
		PnL:             pnl,
		NewAvgPrice:     position.AvgPrice,
	}
}

type Position struct {
	ID          string          `json:"id"`
	UserID      string          `json:"user_id"`
	Symbol      string          `json:"symbol"`
	Quantity    decimal.Decimal `json:"quantity"`
	AvgPrice    decimal.Decimal `json:"avg_price"`
	RealizedPnL decimal.Decimal `json:"realized_pnl"`
	OpenedAt    time.Time       `json:"opened_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type MarginRequirement struct {
	InitialMargin     decimal.Decimal `json:"initial_margin"`
	MaintenanceMargin decimal.Decimal `json:"maintenance_margin"`
}

type MTMResult struct {
	PositionID      string          `json:"position_id"`
	SettlementPrice decimal.Decimal `json:"settlement_price"`
	PnL             decimal.Decimal `json:"pnl"`
	NewAvgPrice     decimal.Decimal `json:"new_avg_price"`
}

type CallAuctionEngine struct {
	symbol      string
	orderBook   *OrderBook
	minTick     decimal.Decimal
	auctionType string
}

func NewCallAuctionEngine(symbol string, minTick decimal.Decimal, auctionType string) *CallAuctionEngine {
	return &CallAuctionEngine{
		symbol:      symbol,
		orderBook:   NewOrderBook(symbol),
		minTick:     minTick,
		auctionType: auctionType,
	}
}

func (e *CallAuctionEngine) CollectOrder(order *types.Order) {
	if order.Side == "BUY" {
		e.orderBook.Bids.Insert(-order.Price.InexactFloat64(), NewOrderLevel(order.Price))
	} else {
		e.orderBook.Asks.Insert(order.Price.InexactFloat64(), NewOrderLevel(order.Price))
	}
}

func (e *CallAuctionEngine) ExecuteAuction() (*AuctionResult, error) {
	ae := NewAuctionEngine(e.symbol, e.minTick, nil)
	ae.Bids = e.orderBook.Bids
	ae.Asks = e.orderBook.Asks
	return ae.Match()
}

type InstrumentRepository interface {
	Create(instrument *Instrument) error
	Update(instrument *Instrument) error
	Delete(symbol string) error
	FindBySymbol(symbol string) (*Instrument, error)
	FindByType(instrumentType InstrumentType) ([]*Instrument, error)
	FindByUnderlying(underlying string) ([]*Instrument, error)
	FindAll() ([]*Instrument, error)
}
