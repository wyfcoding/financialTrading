package domain

import (
	"log/slog"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// CircuitBreakerState 熔断器状态
type CircuitBreakerState int

const (
	StateClosed   CircuitBreakerState = iota // 正常 Closed (Allow requests)
	StateOpen                                // 熔断 Open (Block requests)
	StateHalfOpen                            // 半开 HalfOpen (Probe)
)

// CircuitBreaker 撮合引擎熔断器
// 基于价格波动率进行熔断保护
type CircuitBreaker struct {
	State            CircuitBreakerState
	ThresholdPercent decimal.Decimal // 触发阈值 (例如 0.10 表示 10%)
	CoolDownDuration time.Duration   // 冷却时间
	OpenUntil        time.Time       // 熔断结束时间
	LastPrice        decimal.Decimal // 上一次成交价
	ReferencePrice   decimal.Decimal // 参考价 (例如开盘价/昨收价)，可选

	mu     sync.RWMutex
	logger *slog.Logger
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(threshold decimal.Decimal, cooldown time.Duration, logger *slog.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		State:            StateClosed,
		ThresholdPercent: threshold,
		CoolDownDuration: cooldown,
		logger:           logger,
	}
}

// CheckPrice 检查即将成交的价格是否触发熔断
// 如果返回 false，表示触发熔断，应停止交易
func (cb *CircuitBreaker) CheckPrice(currentPrice decimal.Decimal) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.State {
	case StateOpen:
		// 如果冷却时间已过，进入半开状态，允许一笔交易尝试
		if now.After(cb.OpenUntil) {
			cb.State = StateHalfOpen
			cb.logger.Info("circuit breaker entering half-open state")
			return true
		}
		return false // 处于熔断状态

	case StateHalfOpen:
		// 半开状态，检查价格是否回归正常
		// 如果价格正常，关闭熔断器；否则重新打开并重置冷却
		if cb.isPriceNormal(currentPrice) {
			cb.State = StateClosed
			cb.LastPrice = currentPrice
			cb.logger.Info("circuit breaker closed (recovered)", "price", currentPrice)
			return true
		}
		// 仍然异常，重新熔断
		cb.State = StateOpen
		cb.OpenUntil = now.Add(cb.CoolDownDuration)
		cb.logger.Warn("circuit breaker reopened (probe failed)", "price", currentPrice)
		return false

	case StateClosed:
		// 正常状态，检查是否异常
		if !cb.isPriceNormal(currentPrice) {
			cb.State = StateOpen
			cb.OpenUntil = now.Add(cb.CoolDownDuration)
			cb.logger.Warn("circuit breaker triggered", "price", currentPrice, "last_price", cb.LastPrice)
			return false
		}
		cb.LastPrice = currentPrice
		return true
	}

	return true
}

// isPriceNormal 检查价格是否在允许范围内
func (cb *CircuitBreaker) isPriceNormal(currentPrice decimal.Decimal) bool {
	if cb.LastPrice.IsZero() {
		return true // 第一笔交易
	}

	// 计算涨跌幅: |current - last| / last
	delta := currentPrice.Sub(cb.LastPrice).Abs()
	ratio := delta.Div(cb.LastPrice)

	return ratio.LessThanOrEqual(cb.ThresholdPercent)
}

// Reset 手动重置熔断器
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.State = StateClosed
	cb.logger.Info("circuit breaker manually reset")
}

// SetReferencePrice 设置参考价
func (cb *CircuitBreaker) SetReferencePrice(price decimal.Decimal) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.ReferencePrice = price
	if cb.LastPrice.IsZero() {
		cb.LastPrice = price
	}
}
