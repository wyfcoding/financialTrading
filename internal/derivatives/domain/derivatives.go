package domain

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/shopspring/decimal"
)

var (
	ErrOptionContractNotFound  = errors.New("option contract not found")
	ErrOptionPositionNotFound  = errors.New("option position not found")
	ErrOptionStrategyNotFound  = errors.New("option strategy not found")
	ErrInvalidOptionType       = errors.New("invalid option type")
	ErrInvalidExerciseStyle    = errors.New("invalid exercise style")
	ErrOptionExpired           = errors.New("option expired")
	ErrExerciseNotAllowed      = errors.New("exercise not allowed")
	ErrInsufficientPosition    = errors.New("insufficient position")
	ErrGreeksCalculationFailed = errors.New("greeks calculation failed")
	ErrVolatilityCalcFailed    = errors.New("volatility calculation failed")
	ErrStrategyValidationFailed = errors.New("strategy validation failed")
)

type OptionType string

const (
	OptionTypeCall OptionType = "CALL"
	OptionTypePut  OptionType = "PUT"
)

type OptionStyle string

const (
	OptionStyleEuropean  OptionStyle = "EUROPEAN"
	OptionStyleAmerican  OptionStyle = "AMERICAN"
	OptionStyleBermudan  OptionStyle = "BERMUDAN"
	OptionStyleAsian     OptionStyle = "ASIAN"
)

type ExerciseType string

const (
	ExerciseTypePhysical ExerciseType = "PHYSICAL"
	ExerciseTypeCash     ExerciseType = "CASH"
)

type OptionStatus string

const (
	OptionStatusActive    OptionStatus = "ACTIVE"
	OptionStatusExercised OptionStatus = "EXERCISED"
	OptionStatusExpired   OptionStatus = "EXPIRED"
	OptionStatusAssigned  OptionStatus = "ASSIGNED"
	OptionStatusCancelled OptionStatus = "CANCELLED"
)

type StrategyType string

const (
	StrategyTypeCoveredCall     StrategyType = "COVERED_CALL"
	StrategyTypeProtectivePut   StrategyType = "PROTECTIVE_PUT"
	StrategyTypeStraddle        StrategyType = "STRADDLE"
	StrategyTypeStrangle        StrategyType = "STRANGLE"
	StrategyTypeBullCallSpread  StrategyType = "BULL_CALL_SPREAD"
	StrategyTypeBearPutSpread   StrategyType = "BEAR_PUT_SPREAD"
	StrategyTypeIronCondor      StrategyType = "IRON_CONDOR"
	StrategyTypeIronButterfly   StrategyType = "IRON_BUTTERFLY"
	StrategyTypeButterfly       StrategyType = "BUTTERFLY"
	StrategyTypeCollar          StrategyType = "COLLAR"
)

type PricingModelType string

const (
	PricingModelTypeBlackScholes     PricingModelType = "BLACK_SCHOLES"
	PricingModelTypeBinomial         PricingModelType = "BINOMIAL"
	PricingModelTypeMonteCarlo       PricingModelType = "MONTE_CARLO"
	PricingModelTypeFiniteDifference PricingModelType = "FINITE_DIFFERENCE"
	PricingModelTypeBAW              PricingModelType = "BAW"
)

type OptionContract struct {
	ID             string        `json:"id"`
	Symbol         string        `json:"symbol"`
	Underlying     string        `json:"underlying"`
	OptionType     OptionType    `json:"option_type"`
	OptionStyle    OptionStyle   `json:"option_style"`
	ExerciseType   ExerciseType  `json:"exercise_type"`
	StrikePrice    decimal.Decimal `json:"strike_price"`
	ExpiryDate     time.Time     `json:"expiry_date"`
	Multiplier     decimal.Decimal `json:"multiplier"`
	ContractSize   decimal.Decimal `json:"contract_size"`
	SettlementType string        `json:"settlement_type"`
	TickSize       decimal.Decimal `json:"tick_size"`
	TickValue      decimal.Decimal `json:"tick_value"`
	Exchange       string        `json:"exchange"`
	Currency       string        `json:"currency"`
	IsStandard     bool          `json:"is_standard"`
	Status         OptionStatus  `json:"status"`
	ListingDate    time.Time     `json:"listing_date"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

func NewOptionContract(symbol, underlying string, optionType OptionType, optionStyle OptionStyle, strikePrice decimal.Decimal, expiryDate time.Time, multiplier decimal.Decimal) *OptionContract {
	return &OptionContract{
		Symbol:       symbol,
		Underlying:   underlying,
		OptionType:   optionType,
		OptionStyle:  optionStyle,
		StrikePrice:  strikePrice,
		ExpiryDate:   expiryDate,
		Multiplier:   multiplier,
		ContractSize: multiplier,
		IsStandard:   true,
		Status:       OptionStatusActive,
		ListingDate:  time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func (oc *OptionContract) IsExpired() bool {
	return time.Now().After(oc.ExpiryDate)
}

func (oc *OptionContract) DaysToExpiry() int {
	duration := time.Until(oc.ExpiryDate)
	return int(duration.Hours() / 24)
}

func (oc *OptionContract) IsInTheMoney(underlyingPrice decimal.Decimal) bool {
	if oc.OptionType == OptionTypeCall {
		return underlyingPrice.GreaterThan(oc.StrikePrice)
	}
	return underlyingPrice.LessThan(oc.StrikePrice)
}

func (oc *OptionContract) IntrinsicValue(underlyingPrice decimal.Decimal) decimal.Decimal {
	if oc.OptionType == OptionTypeCall {
		intrinsic := underlyingPrice.Sub(oc.StrikePrice)
		if intrinsic.IsNegative() {
			return decimal.Zero
		}
		return intrinsic
	}
	intrinsic := oc.StrikePrice.Sub(underlyingPrice)
	if intrinsic.IsNegative() {
		return decimal.Zero
	}
	return intrinsic
}

func (oc *OptionContract) TimeValue(underlyingPrice, optionPrice decimal.Decimal) decimal.Decimal {
	intrinsic := oc.IntrinsicValue(underlyingPrice)
	timeValue := optionPrice.Sub(intrinsic)
	if timeValue.IsNegative() {
		return decimal.Zero
	}
	return timeValue
}

func (oc *OptionContract) Moneyness(underlyingPrice decimal.Decimal) string {
	ratio := underlyingPrice.Div(oc.StrikePrice)
	switch {
	case ratio.LessThan(decimal.NewFromFloat(0.95)):
		return "OTM"
	case ratio.GreaterThan(decimal.NewFromFloat(1.05)):
		return "OTM"
	default:
		return "ATM"
	}
}

func (oc *OptionContract) CanExercise() bool {
	if oc.Status != OptionStatusActive {
		return false
	}
	if oc.IsExpired() {
		return false
	}
	if oc.OptionStyle == OptionStyleEuropean {
		return time.Now().After(oc.ExpiryDate.Add(-time.Minute))
	}
	return true
}

func (oc *OptionContract) MarkExercised() {
	oc.Status = OptionStatusExercised
	oc.UpdatedAt = time.Now()
}

func (oc *OptionContract) MarkExpired() {
	oc.Status = OptionStatusExpired
	oc.UpdatedAt = time.Now()
}

func (oc *OptionContract) MarkAssigned() {
	oc.Status = OptionStatusAssigned
	oc.UpdatedAt = time.Now()
}

type OptionGreeks struct {
	Delta      decimal.Decimal `json:"delta"`
	Gamma      decimal.Decimal `json:"gamma"`
	Theta      decimal.Decimal `json:"theta"`
	Vega       decimal.Decimal `json:"vega"`
	Rho        decimal.Decimal `json:"rho"`
	Vanna      decimal.Decimal `json:"vanna"`
	Charm      decimal.Decimal `json:"charm"`
	Speed      decimal.Decimal `json:"speed"`
	Zomma      decimal.Decimal `json:"zomma"`
	Color      decimal.Decimal `json:"color"`
	DvegaDtime decimal.Decimal `json:"dvega_dtime"`
	DvegaDvol  decimal.Decimal `json:"dvega_dvol"`
}

func NewOptionGreeks() *OptionGreeks {
	return &OptionGreeks{
		Delta:      decimal.Zero,
		Gamma:      decimal.Zero,
		Theta:      decimal.Zero,
		Vega:       decimal.Zero,
		Rho:        decimal.Zero,
		Vanna:      decimal.Zero,
		Charm:      decimal.Zero,
		Speed:      decimal.Zero,
		Zomma:      decimal.Zero,
		Color:      decimal.Zero,
		DvegaDtime: decimal.Zero,
		DvegaDvol:  decimal.Zero,
	}
}

func (g *OptionGreeks) Add(other *OptionGreeks) *OptionGreeks {
	return &OptionGreeks{
		Delta:      g.Delta.Add(other.Delta),
		Gamma:      g.Gamma.Add(other.Gamma),
		Theta:      g.Theta.Add(other.Theta),
		Vega:       g.Vega.Add(other.Vega),
		Rho:        g.Rho.Add(other.Rho),
		Vanna:      g.Vanna.Add(other.Vanna),
		Charm:      g.Charm.Add(other.Charm),
		Speed:      g.Speed.Add(other.Speed),
		Zomma:      g.Zomma.Add(other.Zomma),
		Color:      g.Color.Add(other.Color),
		DvegaDtime: g.DvegaDtime.Add(other.DvegaDtime),
		DvegaDvol:  g.DvegaDvol.Add(other.DvegaDvol),
	}
}

func (g *OptionGreeks) Multiply(factor decimal.Decimal) *OptionGreeks {
	return &OptionGreeks{
		Delta:      g.Delta.Mul(factor),
		Gamma:      g.Gamma.Mul(factor),
		Theta:      g.Theta.Mul(factor),
		Vega:       g.Vega.Mul(factor),
		Rho:        g.Rho.Mul(factor),
		Vanna:      g.Vanna.Mul(factor),
		Charm:      g.Charm.Mul(factor),
		Speed:      g.Speed.Mul(factor),
		Zomma:      g.Zomma.Mul(factor),
		Color:      g.Color.Mul(factor),
		DvegaDtime: g.DvegaDtime.Mul(factor),
		DvegaDvol:  g.DvegaDvol.Mul(factor),
	}
}

type OptionPosition struct {
	ID             string          `json:"id"`
	AccountID      string          `json:"account_id"`
	ContractID     string          `json:"contract_id"`
	Contract       *OptionContract `json:"contract,omitempty"`
	Quantity       decimal.Decimal `json:"quantity"`
	AvgPrice       decimal.Decimal `json:"avg_price"`
	MarketValue    decimal.Decimal `json:"market_value"`
	UnrealizedPnL  decimal.Decimal `json:"unrealized_pnl"`
	RealizedPnL    decimal.Decimal `json:"realized_pnl"`
	Greeks         *OptionGreeks   `json:"greeks"`
	Status         OptionStatus    `json:"status"`
	OpenedAt       time.Time       `json:"opened_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	ClosedAt       *time.Time      `json:"closed_at"`
}

func NewOptionPosition(accountID, contractID string, quantity, price decimal.Decimal) *OptionPosition {
	return &OptionPosition{
		AccountID:     accountID,
		ContractID:    contractID,
		Quantity:      quantity,
		AvgPrice:      price,
		MarketValue:   quantity.Mul(price),
		UnrealizedPnL: decimal.Zero,
		RealizedPnL:   decimal.Zero,
		Greeks:        NewOptionGreeks(),
		Status:        OptionStatusActive,
		OpenedAt:      time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func (op *OptionPosition) AddQuantity(quantity, price decimal.Decimal) {
	totalCost := op.Quantity.Mul(op.AvgPrice).Add(quantity.Mul(price))
	op.Quantity = op.Quantity.Add(quantity)
	op.AvgPrice = totalCost.Div(op.Quantity)
	op.MarketValue = op.Quantity.Mul(price)
	op.UpdatedAt = time.Now()
}

func (op *OptionPosition) ReduceQuantity(quantity, closePrice decimal.Decimal) decimal.Decimal {
	if op.Quantity.LessThanOrEqual(quantity) {
		pnl := op.UnrealizedPnL
		op.RealizedPnL = op.RealizedPnL.Add(pnl)
		op.Quantity = decimal.Zero
		op.MarketValue = decimal.Zero
		op.UnrealizedPnL = decimal.Zero
		now := time.Now()
		op.ClosedAt = &now
		op.Status = OptionStatusExercised
		op.UpdatedAt = now
		return pnl
	}
	
	realizedPnL := quantity.Mul(closePrice.Sub(op.AvgPrice))
	op.Quantity = op.Quantity.Sub(quantity)
	op.MarketValue = op.Quantity.Mul(closePrice)
	op.UnrealizedPnL = op.Quantity.Mul(closePrice.Sub(op.AvgPrice))
	op.RealizedPnL = op.RealizedPnL.Add(realizedPnL)
	op.UpdatedAt = time.Now()
	return realizedPnL
}

func (op *OptionPosition) UpdateMarketValue(currentPrice decimal.Decimal) {
	op.MarketValue = op.Quantity.Mul(currentPrice)
	op.UnrealizedPnL = op.MarketValue.Sub(op.Quantity.Mul(op.AvgPrice))
	op.UpdatedAt = time.Now()
}

func (op *OptionPosition) UpdateGreeks(greeks *OptionGreeks) {
	op.Greeks = greeks.Multiply(op.Quantity)
	op.UpdatedAt = time.Now()
}

func (op *OptionPosition) IsLong() bool {
	return op.Quantity.IsPositive()
}

func (op *OptionPosition) IsShort() bool {
	return op.Quantity.IsNegative()
}

type OptionStrategy struct {
	ID           string          `json:"id"`
	AccountID    string          `json:"account_id"`
	StrategyType StrategyType    `json:"strategy_type"`
	Underlying   string          `json:"underlying"`
	Legs         []StrategyLeg   `json:"legs"`
	NetDebit     decimal.Decimal `json:"net_debit"`
	NetCredit    decimal.Decimal `json:"net_credit"`
	MaxProfit    decimal.Decimal `json:"max_profit"`
	MaxLoss      decimal.Decimal `json:"max_loss"`
	BreakevenPoints []decimal.Decimal `json:"breakeven_points"`
	NetGreeks    *OptionGreeks   `json:"net_greeks"`
	Status       string          `json:"status"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type StrategyLeg struct {
	ID           string          `json:"id"`
	ContractID   string          `json:"contract_id"`
	Contract     *OptionContract `json:"contract,omitempty"`
	Quantity     decimal.Decimal `json:"quantity"`
	Price        decimal.Decimal `json:"price"`
	Side         string          `json:"side"`
	PositionEffect string        `json:"position_effect"`
}

func NewOptionStrategy(accountID string, strategyType StrategyType, underlying string) *OptionStrategy {
	return &OptionStrategy{
		AccountID:      accountID,
		StrategyType:   strategyType,
		Underlying:     underlying,
		Legs:           []StrategyLeg{},
		NetDebit:       decimal.Zero,
		NetCredit:      decimal.Zero,
		MaxProfit:      decimal.Zero,
		MaxLoss:        decimal.Zero,
		BreakevenPoints: []decimal.Decimal{},
		NetGreeks:      NewOptionGreeks(),
		Status:         "OPEN",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func (os *OptionStrategy) AddLeg(contractID string, quantity, price decimal.Decimal, side string) {
	leg := StrategyLeg{
		ContractID: contractID,
		Quantity:   quantity,
		Price:      price,
		Side:       side,
	}
	os.Legs = append(os.Legs, leg)
	
	if side == "BUY" {
		os.NetDebit = os.NetDebit.Add(quantity.Mul(price))
	} else {
		os.NetCredit = os.NetCredit.Add(quantity.Mul(price))
	}
	os.UpdatedAt = time.Now()
}

func (os *OptionStrategy) CalculateNetGreeks(legGreeksMap map[string]*OptionGreeks) {
	netGreeks := NewOptionGreeks()
	for _, leg := range os.Legs {
		if greeks, ok := legGreeksMap[leg.ContractID]; ok {
			legGreeks := greeks.Multiply(leg.Quantity)
			if leg.Side == "SELL" {
				legGreeks = legGreeks.Multiply(decimal.NewFromInt(-1))
			}
			netGreeks = netGreeks.Add(legGreeks)
		}
	}
	os.NetGreeks = netGreeks
	os.UpdatedAt = time.Now()
}

func (os *OptionStrategy) Close() {
	os.Status = "CLOSED"
	os.UpdatedAt = time.Now()
}

type VolatilitySurface struct {
	ID                   string                  `json:"id"`
	Underlying           string                  `json:"underlying"`
	AsOf                 time.Time               `json:"as_of"`
	Points               []VolatilitySurfacePoint `json:"points"`
	InterpolationMethod  string                  `json:"interpolation_method"`
	ExtrapolationMethod  string                  `json:"extrapolation_method"`
	CreatedAt            time.Time               `json:"created_at"`
}

type VolatilitySurfacePoint struct {
	StrikePrice     decimal.Decimal `json:"strike_price"`
	Moneyness       decimal.Decimal `json:"moneyness"`
	DaysToExpiry    int             `json:"days_to_expiry"`
	ImpliedVol      decimal.Decimal `json:"implied_vol"`
	BidIV           decimal.Decimal `json:"bid_iv"`
	AskIV           decimal.Decimal `json:"ask_iv"`
}

func NewVolatilitySurface(underlying string) *VolatilitySurface {
	return &VolatilitySurface{
		Underlying:          underlying,
		AsOf:                time.Now(),
		Points:              []VolatilitySurfacePoint{},
		InterpolationMethod: "CUBIC_SPLINE",
		ExtrapolationMethod: "CONSTANT",
		CreatedAt:           time.Now(),
	}
}

func (vs *VolatilitySurface) AddPoint(strike, moneyness decimal.Decimal, daysToExpiry int, impliedVol, bidIV, askIV decimal.Decimal) {
	point := VolatilitySurfacePoint{
		StrikePrice:  strike,
		Moneyness:    moneyness,
		DaysToExpiry: daysToExpiry,
		ImpliedVol:   impliedVol,
		BidIV:        bidIV,
		AskIV:        askIV,
	}
	vs.Points = append(vs.Points, point)
}

func (vs *VolatilitySurface) GetVolatility(strike decimal.Decimal, daysToExpiry int) decimal.Decimal {
	for _, point := range vs.Points {
		if point.StrikePrice.Equal(strike) && point.DaysToExpiry == daysToExpiry {
			return point.ImpliedVol
		}
	}
	return decimal.Zero
}

type VolatilitySmile struct {
	ID         string                `json:"id"`
	Underlying string                `json:"underlying"`
	ExpiryDate time.Time             `json:"expiry_date"`
	Points     []VolatilitySmilePoint `json:"points"`
	Skew       decimal.Decimal       `json:"skew"`
	Kurtosis   decimal.Decimal       `json:"kurtosis"`
	CreatedAt  time.Time             `json:"created_at"`
}

type VolatilitySmilePoint struct {
	StrikePrice decimal.Decimal `json:"strike_price"`
	Delta       decimal.Decimal `json:"delta"`
	ImpliedVol  decimal.Decimal `json:"implied_vol"`
	Moneyness   decimal.Decimal `json:"moneyness"`
}

func NewVolatilitySmile(underlying string, expiryDate time.Time) *VolatilitySmile {
	return &VolatilitySmile{
		Underlying: underlying,
		ExpiryDate: expiryDate,
		Points:     []VolatilitySmilePoint{},
		Skew:       decimal.Zero,
		Kurtosis:   decimal.Zero,
		CreatedAt:  time.Now(),
	}
}

func (vsm *VolatilitySmile) AddPoint(strike, delta, impliedVol, moneyness decimal.Decimal) {
	point := VolatilitySmilePoint{
		StrikePrice: strike,
		Delta:       delta,
		ImpliedVol:  impliedVol,
		Moneyness:   moneyness,
	}
	vsm.Points = append(vsm.Points, point)
}

type OptionChain struct {
	ID             string            `json:"id"`
	Underlying     string            `json:"underlying"`
	ExpiryDate     time.Time         `json:"expiry_date"`
	Calls          []OptionChainEntry `json:"calls"`
	Puts           []OptionChainEntry `json:"puts"`
	UnderlyingPrice decimal.Decimal  `json:"underlying_price"`
	DaysToExpiry   int               `json:"days_to_expiry"`
	AtmVolatility  decimal.Decimal   `json:"atm_volatility"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

type OptionChainEntry struct {
	ContractID     string          `json:"contract_id"`
	StrikePrice    decimal.Decimal `json:"strike_price"`
	LastPrice      decimal.Decimal `json:"last_price"`
	BidPrice       decimal.Decimal `json:"bid_price"`
	AskPrice       decimal.Decimal `json:"ask_price"`
	BidSize        decimal.Decimal `json:"bid_size"`
	AskSize        decimal.Decimal `json:"ask_size"`
	Volume         decimal.Decimal `json:"volume"`
	OpenInterest   decimal.Decimal `json:"open_interest"`
	ImpliedVol     decimal.Decimal `json:"implied_vol"`
	Greeks         *OptionGreeks   `json:"greeks"`
	InTheMoney     bool            `json:"in_the_money"`
	IntrinsicValue decimal.Decimal `json:"intrinsic_value"`
	TimeValue      decimal.Decimal `json:"time_value"`
}

func NewOptionChain(underlying string, expiryDate time.Time, underlyingPrice decimal.Decimal) *OptionChain {
	daysToExpiry := int(time.Until(expiryDate).Hours() / 24)
	return &OptionChain{
		Underlying:      underlying,
		ExpiryDate:      expiryDate,
		Calls:           []OptionChainEntry{},
		Puts:            []OptionChainEntry{},
		UnderlyingPrice: underlyingPrice,
		DaysToExpiry:    daysToExpiry,
		AtmVolatility:   decimal.Zero,
		UpdatedAt:       time.Now(),
	}
}

func (oc *OptionChain) AddCall(entry OptionChainEntry) {
	oc.Calls = append(oc.Calls, entry)
	oc.UpdatedAt = time.Now()
}

func (oc *OptionChain) AddPut(entry OptionChainEntry) {
	oc.Puts = append(oc.Puts, entry)
	oc.UpdatedAt = time.Now()
}

func (oc *OptionChain) FindATMStrike() decimal.Decimal {
	minDiff := decimal.NewFromFloat(math.MaxFloat64)
	var atmStrike decimal.Decimal
	
	for _, call := range oc.Calls {
		diff := call.StrikePrice.Sub(oc.UnderlyingPrice).Abs()
		if diff.LessThan(minDiff) {
			minDiff = diff
			atmStrike = call.StrikePrice
		}
	}
	return atmStrike
}

type OptionOrder struct {
	ID             string          `json:"id"`
	AccountID      string          `json:"account_id"`
	ContractID     string          `json:"contract_id"`
	Side           string          `json:"side"`
	OrderType      string          `json:"order_type"`
	Quantity       decimal.Decimal `json:"quantity"`
	Price          decimal.Decimal `json:"price"`
	FilledQuantity decimal.Decimal `json:"filled_quantity"`
	AvgFillPrice   decimal.Decimal `json:"avg_fill_price"`
	Status         string          `json:"status"`
	TimeInForce    string          `json:"time_in_force"`
	StrategyID     string          `json:"strategy_id"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	ExpiredAt      *time.Time      `json:"expired_at"`
}

func NewOptionOrder(accountID, contractID, side, orderType string, quantity, price decimal.Decimal) *OptionOrder {
	return &OptionOrder{
		AccountID:      accountID,
		ContractID:     contractID,
		Side:           side,
		OrderType:      orderType,
		Quantity:       quantity,
		Price:          price,
		FilledQuantity: decimal.Zero,
		AvgFillPrice:   decimal.Zero,
		Status:         "PENDING",
		TimeInForce:    "GTC",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func (oo *OptionOrder) Fill(quantity, price decimal.Decimal) {
	totalFilled := oo.FilledQuantity.Add(quantity)
	totalCost := oo.FilledQuantity.Mul(oo.AvgFillPrice).Add(quantity.Mul(price))
	
	oo.FilledQuantity = totalFilled
	oo.AvgFillPrice = totalCost.Div(totalFilled)
	
	if oo.FilledQuantity.Equal(oo.Quantity) {
		oo.Status = "FILLED"
	} else {
		oo.Status = "PARTIALLY_FILLED"
	}
	oo.UpdatedAt = time.Now()
}

func (oo *OptionOrder) Cancel() {
	oo.Status = "CANCELLED"
	oo.UpdatedAt = time.Now()
}

func (oo *OptionOrder) IsFullyFilled() bool {
	return oo.FilledQuantity.Equal(oo.Quantity)
}

type ExerciseRecord struct {
	ID            string          `json:"id"`
	PositionID    string          `json:"position_id"`
	ContractID    string          `json:"contract_id"`
	AccountID     string          `json:"account_id"`
	Quantity      decimal.Decimal `json:"quantity"`
	ExerciseType  ExerciseType    `json:"exercise_type"`
	ExercisePrice decimal.Decimal `json:"exercise_price"`
	SettlementAmt decimal.Decimal `json:"settlement_amt"`
	ExercisedAt   time.Time       `json:"exercised_at"`
	CreatedAt     time.Time       `json:"created_at"`
}

func NewExerciseRecord(positionID, contractID, accountID string, quantity, exercisePrice decimal.Decimal, exerciseType ExerciseType) *ExerciseRecord {
	return &ExerciseRecord{
		PositionID:    positionID,
		ContractID:    contractID,
		AccountID:     accountID,
		Quantity:      quantity,
		ExerciseType:  exerciseType,
		ExercisePrice: exercisePrice,
		ExercisedAt:   time.Now(),
		CreatedAt:     time.Now(),
	}
}

func (er *ExerciseRecord) CalculateSettlement(underlyingPrice decimal.Decimal, optionType OptionType, multiplier decimal.Decimal) {
	if optionType == OptionTypeCall {
		er.SettlementAmt = er.Quantity.Mul(multiplier).Mul(underlyingPrice.Sub(er.ExercisePrice))
	} else {
		er.SettlementAmt = er.Quantity.Mul(multiplier).Mul(er.ExercisePrice.Sub(underlyingPrice))
	}
	if er.SettlementAmt.IsNegative() {
		er.SettlementAmt = decimal.Zero
	}
}

type OptionContractRepository interface {
	Create(ctx context.Context, contract *OptionContract) error
	Update(ctx context.Context, contract *OptionContract) error
	FindByID(ctx context.Context, id string) (*OptionContract, error)
	FindBySymbol(ctx context.Context, symbol string) (*OptionContract, error)
	FindByUnderlying(ctx context.Context, underlying string, limit, offset int) ([]*OptionContract, int64, error)
	FindByExpiry(ctx context.Context, underlying string, expiryDate time.Time) ([]*OptionContract, error)
	FindExpiringSoon(ctx context.Context, days int) ([]*OptionContract, error)
	Delete(ctx context.Context, id string) error
}

type OptionPositionRepository interface {
	Create(ctx context.Context, position *OptionPosition) error
	Update(ctx context.Context, position *OptionPosition) error
	FindByID(ctx context.Context, id string) (*OptionPosition, error)
	FindByAccountID(ctx context.Context, accountID string) ([]*OptionPosition, error)
	FindByContractID(ctx context.Context, accountID, contractID string) (*OptionPosition, error)
	FindActive(ctx context.Context, accountID string) ([]*OptionPosition, error)
	Delete(ctx context.Context, id string) error
}

type OptionStrategyRepository interface {
	Create(ctx context.Context, strategy *OptionStrategy) error
	Update(ctx context.Context, strategy *OptionStrategy) error
	FindByID(ctx context.Context, id string) (*OptionStrategy, error)
	FindByAccountID(ctx context.Context, accountID string) ([]*OptionStrategy, error)
	FindByType(ctx context.Context, accountID string, strategyType StrategyType) ([]*OptionStrategy, error)
	Delete(ctx context.Context, id string) error
}

type VolatilitySurfaceRepository interface {
	Create(ctx context.Context, surface *VolatilitySurface) error
	FindByID(ctx context.Context, id string) (*VolatilitySurface, error)
	FindByUnderlying(ctx context.Context, underlying string) (*VolatilitySurface, error)
	FindLatest(ctx context.Context, underlying string) (*VolatilitySurface, error)
	Delete(ctx context.Context, id string) error
}

type VolatilitySmileRepository interface {
	Create(ctx context.Context, smile *VolatilitySmile) error
	FindByID(ctx context.Context, id string) (*VolatilitySmile, error)
	FindByUnderlyingAndExpiry(ctx context.Context, underlying string, expiryDate time.Time) (*VolatilitySmile, error)
	Delete(ctx context.Context, id string) error
}

type OptionChainRepository interface {
	Create(ctx context.Context, chain *OptionChain) error
	FindByID(ctx context.Context, id string) (*OptionChain, error)
	FindByUnderlyingAndExpiry(ctx context.Context, underlying string, expiryDate time.Time) (*OptionChain, error)
	FindLatest(ctx context.Context, underlying string) (*OptionChain, error)
	Delete(ctx context.Context, id string) error
}

type OptionOrderRepository interface {
	Create(ctx context.Context, order *OptionOrder) error
	Update(ctx context.Context, order *OptionOrder) error
	FindByID(ctx context.Context, id string) (*OptionOrder, error)
	FindByAccountID(ctx context.Context, accountID string, limit, offset int) ([]*OptionOrder, int64, error)
	FindPending(ctx context.Context, accountID string) ([]*OptionOrder, error)
	Delete(ctx context.Context, id string) error
}

type ExerciseRecordRepository interface {
	Create(ctx context.Context, record *ExerciseRecord) error
	FindByID(ctx context.Context, id string) (*ExerciseRecord, error)
	FindByAccountID(ctx context.Context, accountID string, limit, offset int) ([]*ExerciseRecord, int64, error)
	FindByPositionID(ctx context.Context, positionID string) ([]*ExerciseRecord, error)
	Delete(ctx context.Context, id string) error
}
