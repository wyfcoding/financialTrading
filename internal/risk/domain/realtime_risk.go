package domain

import (
	"context"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

type RiskAction string

const (
	RiskActionAllow      RiskAction = "ALLOW"
	RiskActionWarn       RiskAction = "WARN"
	RiskActionBlock      RiskAction = "BLOCK"
	RiskActionReview     RiskAction = "REVIEW"
	RiskActionChallenge  RiskAction = "CHALLENGE"
)

type RiskCheckType string

const (
	RiskCheckPreTrade    RiskCheckType = "PRE_TRADE"
	RiskCheckPostTrade   RiskCheckType = "POST_TRADE"
	RiskCheckRealTime    RiskCheckType = "REAL_TIME"
	RiskCheckEndOfDay    RiskCheckType = "END_OF_DAY"
)

type RiskRuleCategory string

const (
	RiskCategoryPosition    RiskRuleCategory = "POSITION"
	RiskCategoryExposure    RiskRuleCategory = "EXPOSURE"
	RiskCategoryLeverage    RiskRuleCategory = "LEVERAGE"
	RiskCategoryLiquidity   RiskRuleCategory = "LIQUIDITY"
	RiskCategoryCredit      RiskRuleCategory = "CREDIT"
	RiskCategoryMarket      RiskRuleCategory = "MARKET"
	RiskCategoryOperational RiskRuleCategory = "OPERATIONAL"
	RiskCategoryCompliance  RiskRuleCategory = "COMPLIANCE"
)

type RealTimeRiskEngine struct {
	rules          map[string]*RiskRule
	positionCache  *PositionCache
	exposureCache  *ExposureCache
	alertManager   *RiskAlertManager
	circuitBreaker *CircuitBreakerManager
	mu             sync.RWMutex
}

type RiskRule struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Category        RiskRuleCategory `json:"category"`
	Description     string           `json:"description"`
	Enabled         bool             `json:"enabled"`
	Priority        int              `json:"priority"`
	Condition       RiskCondition    `json:"condition"`
	Threshold       decimal.Decimal  `json:"threshold"`
	Action          RiskAction       `json:"action"`
	CoolDownSeconds int              `json:"cool_down_seconds"`
	NotifyChannels  []string         `json:"notify_channels"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

type RiskCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type PositionCache struct {
	positions map[string]*UserPosition
	mu        sync.RWMutex
}

type UserPosition struct {
	UserID       string                     `json:"user_id"`
	Positions    map[string]*SymbolPosition `json:"positions"`
	TotalValue   decimal.Decimal            `json:"total_value"`
	TotalExposure decimal.Decimal           `json:"total_exposure"`
	UpdatedAt    time.Time                  `json:"updated_at"`
}

type SymbolPosition struct {
	Symbol       string          `json:"symbol"`
	Quantity     decimal.Decimal `json:"quantity"`
	AvgPrice     decimal.Decimal `json:"avg_price"`
	MarketValue  decimal.Decimal `json:"market_value"`
	UnrealizedPnL decimal.Decimal `json:"unrealized_pnl"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type ExposureCache struct {
	exposures map[string]*UserExposure
	mu        sync.RWMutex
}

type UserExposure struct {
	UserID           string          `json:"user_id"`
	GrossExposure    decimal.Decimal `json:"gross_exposure"`
	NetExposure      decimal.Decimal `json:"net_exposure"`
	LongExposure     decimal.Decimal `json:"long_exposure"`
	ShortExposure    decimal.Decimal `json:"short_exposure"`
	DeltaExposure    decimal.Decimal `json:"delta_exposure"`
	GammaExposure    decimal.Decimal `json:"gamma_exposure"`
	VegaExposure     decimal.Decimal `json:"vega_exposure"`
	ThetaExposure    decimal.Decimal `json:"theta_exposure"`
	MaxDrawdown      decimal.Decimal `json:"max_drawdown"`
	ValueAtRisk95    decimal.Decimal `json:"var_95"`
	ValueAtRisk99    decimal.Decimal `json:"var_99"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type RiskAlertManager struct {
	alerts     map[string]*RiskAlert
	alertQueue chan *RiskAlert
	mu         sync.RWMutex
}

type CircuitBreakerManager struct {
	breakers   map[string]*CircuitBreakerState
	mu         sync.RWMutex
}

type CircuitBreakerState struct {
	UserID        string          `json:"user_id"`
	IsTriggered   bool            `json:"is_triggered"`
	TriggerCount  int             `json:"trigger_count"`
	TriggerReason string          `json:"trigger_reason"`
	TriggeredAt   *time.Time      `json:"triggered_at,omitempty"`
	ResetAt       *time.Time      `json:"reset_at,omitempty"`
	Threshold     decimal.Decimal `json:"threshold"`
}

type RiskCheckContext struct {
	UserID       string          `json:"user_id"`
	Symbol       string          `json:"symbol"`
	Side         string          `json:"side"`
	OrderType    string          `json:"order_type"`
	Quantity     decimal.Decimal `json:"quantity"`
	Price        decimal.Decimal `json:"price"`
	OrderValue   decimal.Decimal `json:"order_value"`
	AccountValue decimal.Decimal `json:"account_value"`
	Leverage     decimal.Decimal `json:"leverage"`
	IPAddress    string          `json:"ip_address"`
	DeviceID     string          `json:"device_id"`
	Timestamp    time.Time       `json:"timestamp"`
	Extra        map[string]interface{} `json:"extra,omitempty"`
}

type RiskCheckResult struct {
	CheckID      string          `json:"check_id"`
	UserID       string          `json:"user_id"`
	Passed       bool            `json:"passed"`
	Action       RiskAction      `json:"action"`
	RiskLevel    RiskLevel       `json:"risk_level"`
	RiskScore    decimal.Decimal `json:"risk_score"`
	TriggeredRules []*TriggeredRule `json:"triggered_rules,omitempty"`
	Reason       string          `json:"reason"`
	CheckedAt    time.Time       `json:"checked_at"`
}

type TriggeredRule struct {
	RuleID    string `json:"rule_id"`
	RuleName  string `json:"rule_name"`
	Threshold string `json:"threshold"`
	Actual    string `json:"actual"`
	Message   string `json:"message"`
}

func NewRealTimeRiskEngine() *RealTimeRiskEngine {
	return &RealTimeRiskEngine{
		rules:          make(map[string]*RiskRule),
		positionCache:  NewPositionCache(),
		exposureCache:  NewExposureCache(),
		alertManager:   NewRiskAlertManager(),
		circuitBreaker: NewCircuitBreakerManager(),
	}
}

func NewPositionCache() *PositionCache {
	return &PositionCache{
		positions: make(map[string]*UserPosition),
	}
}

func NewExposureCache() *ExposureCache {
	return &ExposureCache{
		exposures: make(map[string]*UserExposure),
	}
}

func NewRiskAlertManager() *RiskAlertManager {
	return &RiskAlertManager{
		alerts:     make(map[string]*RiskAlert),
		alertQueue: make(chan *RiskAlert, 1000),
	}
}

func NewCircuitBreakerManager() *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreakerState),
	}
}

func (e *RealTimeRiskEngine) AddRule(rule *RiskRule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules[rule.ID] = rule
}

func (e *RealTimeRiskEngine) RemoveRule(ruleID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.rules, ruleID)
}

func (e *RealTimeRiskEngine) CheckPreTrade(ctx context.Context, checkCtx *RiskCheckContext) (*RiskCheckResult, error) {
	result := &RiskCheckResult{
		CheckID:   generateCheckID(),
		UserID:    checkCtx.UserID,
		Passed:    true,
		Action:    RiskActionAllow,
		RiskLevel: RiskLevelLow,
		CheckedAt: time.Now(),
	}

	if e.circuitBreaker.IsTriggered(checkCtx.UserID) {
		result.Passed = false
		result.Action = RiskActionBlock
		result.RiskLevel = RiskLevelCritical
		result.Reason = "Circuit breaker triggered"
		return result, nil
	}

	e.mu.RLock()
	rules := make([]*RiskRule, 0, len(e.rules))
	for _, rule := range e.rules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	e.mu.RUnlock()

	for _, rule := range rules {
		triggered, actual := e.evaluateRule(rule, checkCtx)
		if triggered {
			result.TriggeredRules = append(result.TriggeredRules, &TriggeredRule{
				RuleID:    rule.ID,
				RuleName:  rule.Name,
				Threshold: rule.Threshold.String(),
				Actual:    actual,
				Message:   rule.Description,
			})

			if rule.Action == RiskActionBlock {
				result.Passed = false
				result.Action = RiskActionBlock
				result.RiskLevel = RiskLevelHigh
			} else if rule.Action == RiskActionReview {
				result.Action = RiskActionReview
				result.RiskLevel = RiskLevelMedium
			} else if rule.Action == RiskActionWarn && result.Action == RiskActionAllow {
				result.Action = RiskActionWarn
				result.RiskLevel = RiskLevelMedium
			}
		}
	}

	result.RiskScore = e.calculateRiskScore(result.TriggeredRules)

	if len(result.TriggeredRules) > 0 {
		e.alertManager.SendAlert(&RiskAlert{
			UserID:    checkCtx.UserID,
			AlertType: "PRE_TRADE_CHECK",
			Severity:  string(result.RiskLevel),
			Message:   result.Reason,
		})
	}

	return result, nil
}

func (e *RealTimeRiskEngine) evaluateRule(rule *RiskRule, ctx *RiskCheckContext) (bool, string) {
	switch rule.Category {
	case RiskCategoryPosition:
		return e.evaluatePositionRule(rule, ctx)
	case RiskCategoryExposure:
		return e.evaluateExposureRule(rule, ctx)
	case RiskCategoryLeverage:
		return e.evaluateLeverageRule(rule, ctx)
	case RiskCategoryCredit:
		return e.evaluateCreditRule(rule, ctx)
	default:
		return false, ""
	}
}

func (e *RealTimeRiskEngine) evaluatePositionRule(rule *RiskRule, ctx *RiskCheckContext) (bool, string) {
	position := e.positionCache.GetPosition(ctx.UserID, ctx.Symbol)
	if position == nil {
		return false, "0"
	}

	newQuantity := position.Quantity
	if ctx.Side == "BUY" {
		newQuantity = newQuantity.Add(ctx.Quantity)
	} else {
		newQuantity = newQuantity.Sub(ctx.Quantity)
	}

	absQuantity := newQuantity.Abs()
	return absQuantity.GreaterThan(rule.Threshold), absQuantity.String()
}

func (e *RealTimeRiskEngine) evaluateExposureRule(rule *RiskRule, ctx *RiskCheckContext) (bool, string) {
	exposure := e.exposureCache.GetExposure(ctx.UserID)
	if exposure == nil {
		return false, "0"
	}

	newExposure := exposure.GrossExposure
	if ctx.Side == "BUY" {
		newExposure = newExposure.Add(ctx.OrderValue)
	} else {
		newExposure = newExposure.Sub(ctx.OrderValue)
	}

	absExposure := newExposure.Abs()
	return absExposure.GreaterThan(rule.Threshold), absExposure.String()
}

func (e *RealTimeRiskEngine) evaluateLeverageRule(rule *RiskRule, ctx *RiskCheckContext) (bool, string) {
	if ctx.AccountValue.IsZero() {
		return false, "0"
	}

	exposure := e.exposureCache.GetExposure(ctx.UserID)
	if exposure == nil {
		return false, "0"
	}

	newExposure := exposure.GrossExposure.Add(ctx.OrderValue)
	leverage := newExposure.Div(ctx.AccountValue)
	return leverage.GreaterThan(rule.Threshold), leverage.StringFixed(2)
}

func (e *RealTimeRiskEngine) evaluateCreditRule(rule *RiskRule, ctx *RiskCheckContext) (bool, string) {
	return false, "N/A"
}

func (e *RealTimeRiskEngine) calculateRiskScore(triggeredRules []*TriggeredRule) decimal.Decimal {
	if len(triggeredRules) == 0 {
		return decimal.Zero
	}

	score := decimal.NewFromInt(int64(len(triggeredRules) * 10))
	return score
}

func (e *RealTimeRiskEngine) UpdatePosition(userID, symbol string, quantity, price decimal.Decimal) {
	e.positionCache.UpdatePosition(userID, symbol, quantity, price)
}

func (e *RealTimeRiskEngine) UpdateExposure(userID string, exposure *UserExposure) {
	e.exposureCache.UpdateExposure(userID, exposure)
}

func (pc *PositionCache) GetPosition(userID, symbol string) *SymbolPosition {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	userPos, exists := pc.positions[userID]
	if !exists {
		return nil
	}

	return userPos.Positions[symbol]
}

func (pc *PositionCache) UpdatePosition(userID, symbol string, quantity, price decimal.Decimal) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	userPos, exists := pc.positions[userID]
	if !exists {
		userPos = &UserPosition{
			UserID:    userID,
			Positions: make(map[string]*SymbolPosition),
		}
		pc.positions[userID] = userPos
	}

	symbolPos, exists := userPos.Positions[symbol]
	if !exists {
		symbolPos = &SymbolPosition{
			Symbol:   symbol,
			Quantity: decimal.Zero,
		}
		userPos.Positions[symbol] = symbolPos
	}

	symbolPos.Quantity = symbolPos.Quantity.Add(quantity)
	symbolPos.AvgPrice = price
	symbolPos.MarketValue = symbolPos.Quantity.Mul(price)
	symbolPos.UpdatedAt = time.Now()

	userPos.TotalValue = decimal.Zero
	for _, pos := range userPos.Positions {
		userPos.TotalValue = userPos.TotalValue.Add(pos.MarketValue.Abs())
	}
	userPos.UpdatedAt = time.Now()
}

func (ec *ExposureCache) GetExposure(userID string) *UserExposure {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.exposures[userID]
}

func (ec *ExposureCache) UpdateExposure(userID string, exposure *UserExposure) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	exposure.UpdatedAt = time.Now()
	ec.exposures[userID] = exposure
}

func (am *RiskAlertManager) SendAlert(alert *RiskAlert) {
	am.mu.Lock()
	defer am.mu.Unlock()
	alert.CreatedAt = time.Now()
	alert.UpdatedAt = time.Now()
	am.alerts[alert.ID] = alert
	select {
	case am.alertQueue <- alert:
	default:
	}
}

func (am *RiskAlertManager) GetAlerts(userID string) []*RiskAlert {
	am.mu.RLock()
	defer am.mu.RUnlock()
	alerts := make([]*RiskAlert, 0)
	for _, alert := range am.alerts {
		if alert.UserID == userID {
			alerts = append(alerts, alert)
		}
	}
	return alerts
}

func (cbm *CircuitBreakerManager) IsTriggered(userID string) bool {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()
	if breaker, exists := cbm.breakers[userID]; exists {
		return breaker.IsTriggered
	}
	return false
}

func (cbm *CircuitBreakerManager) Trigger(userID, reason string) {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()
	now := time.Now()
	breaker, exists := cbm.breakers[userID]
	if !exists {
		breaker = &CircuitBreakerState{UserID: userID}
		cbm.breakers[userID] = breaker
	}
	breaker.IsTriggered = true
	breaker.TriggerCount++
	breaker.TriggerReason = reason
	breaker.TriggeredAt = &now
}

func (cbm *CircuitBreakerManager) Reset(userID string) {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()
	now := time.Now()
	if breaker, exists := cbm.breakers[userID]; exists {
		breaker.IsTriggered = false
		breaker.ResetAt = &now
	}
}

func generateCheckID() string {
	return time.Now().Format("20060102150405") + randomSuffix()
}

func randomSuffix() string {
	return "RISK"
}

type RiskRuleRepository interface {
	Create(rule *RiskRule) error
	Update(rule *RiskRule) error
	Delete(ruleID string) error
	FindByID(ruleID string) (*RiskRule, error)
	FindByCategory(category RiskRuleCategory) ([]*RiskRule, error)
	ListEnabled() ([]*RiskRule, error)
}

type RiskCheckResultRepository interface {
	Save(result *RiskCheckResult) error
	FindByCheckID(checkID string) (*RiskCheckResult, error)
	FindByUserID(userID string, startTime, endTime *time.Time, page, pageSize int) ([]*RiskCheckResult, int64, error)
}

type RiskAlertRepository interface {
	Create(alert *RiskAlert) error
	Update(alert *RiskAlert) error
	FindByID(alertID string) (*RiskAlert, error)
	FindByUserID(userID string, startTime, endTime *time.Time) ([]*RiskAlert, error)
	ListUnresolved(limit int) ([]*RiskAlert, error)
}
