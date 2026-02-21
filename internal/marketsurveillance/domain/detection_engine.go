// Package domain 市场监察检测引擎 - 领域服务
// 生成摘要：
//  1. 实现 Spoofing / Layering / WashTrading / FrontRunning 四种检测算法
//  2. 基于规则引擎，使用时间窗口内的订单事件流进行实时判定
//  3. 返回检测结果包含置信度分数、告警严重程度
//
// 假设：检测阈值由 SurveillanceRule 配置，可热更新
package domain

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
)

// DetectionResult 检测结果
type DetectionResult struct {
	Triggered  bool
	Type       ManipulationType
	Severity   AlertSeverity
	Confidence float64
	Reason     string
}

// DetectionEngine 检测引擎领域服务
type DetectionEngine struct {
	logger *slog.Logger
}

// NewDetectionEngine 创建检测引擎
func NewDetectionEngine(logger *slog.Logger) *DetectionEngine {
	return &DetectionEngine{logger: logger}
}

// Detect 对订单事件流执行全部启用规则的检测
func (e *DetectionEngine) Detect(
	event OrderEvent,
	history []OrderEvent,
	rules []*SurveillanceRule,
) []DetectionResult {
	var results []DetectionResult
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		window := time.Duration(rule.WindowSeconds) * time.Second
		windowEvents := filterWindow(history, event.Timestamp, window)

		var result DetectionResult
		switch rule.Type {
		case ManipulationSpoofing:
			result = e.detectSpoofing(event, windowEvents, rule)
		case ManipulationLayering:
			result = e.detectLayering(event, windowEvents, rule)
		case ManipulationWashTrading:
			result = e.detectWashTrading(event, windowEvents, rule)
		case ManipulationFrontRun:
			result = e.detectFrontRunning(event, windowEvents, rule)
		case ManipulationClosingPrice:
			result = e.detectClosingPriceManip(event, windowEvents, rule)
		case ManipulationPumpAndDump:
			result = e.detectPumpAndDump(event, windowEvents, rule)
		default:
			continue
		}
		if result.Triggered {
			results = append(results, result)
		}
	}
	return results
}

// detectSpoofing 幌骗检测
// 算法：检测时间窗口内同一用户的下单-快速撤单比率
// 如果撤单率超过阈值且平均订单存活时间极短，判定为幌骗
// 复杂度：O(n)，n 为窗口内事件数
func (e *DetectionEngine) detectSpoofing(
	current OrderEvent,
	history []OrderEvent,
	rule *SurveillanceRule,
) DetectionResult {
	if current.EventType != EventCancel {
		return DetectionResult{}
	}

	var placeCount, cancelCount int
	var totalLifetime time.Duration
	placeMap := make(map[string]time.Time)

	for _, ev := range history {
		if ev.UserID != current.UserID || ev.Symbol != current.Symbol {
			continue
		}
		switch ev.EventType {
		case EventPlace:
			placeCount++
			placeMap[ev.OrderID] = ev.Timestamp
		case EventCancel:
			cancelCount++
			if pt, ok := placeMap[ev.OrderID]; ok {
				totalLifetime += ev.Timestamp.Sub(pt)
			}
		}
	}

	if placeCount < 3 {
		return DetectionResult{}
	}

	cancelRatio := decimal.NewFromInt(int64(cancelCount)).Div(decimal.NewFromInt(int64(placeCount)))
	if cancelRatio.LessThan(rule.MinCancelRatio) {
		return DetectionResult{}
	}

	avgLifetimeMs := float64(0)
	if cancelCount > 0 {
		avgLifetimeMs = float64(totalLifetime.Milliseconds()) / float64(cancelCount)
	}

	confidence := cancelRatio.InexactFloat64() * 0.6
	if avgLifetimeMs < 500 {
		confidence += 0.3
	} else if avgLifetimeMs < 2000 {
		confidence += 0.15
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	severity := e.scoreSeverity(confidence)

	return DetectionResult{
		Triggered:  true,
		Type:       ManipulationSpoofing,
		Severity:   severity,
		Confidence: confidence,
		Reason: fmt.Sprintf(
			"Spoofing detected: cancel_ratio=%.2f, avg_lifetime=%.0fms, orders=%d",
			cancelRatio.InexactFloat64(), avgLifetimeMs, placeCount,
		),
	}
}

// detectLayering 分层检测
// 算法：检测同一用户在同一标的上多个不同价位挂单形成梯度，
// 且大部分在成交前被撤销
func (e *DetectionEngine) detectLayering(
	current OrderEvent,
	history []OrderEvent,
	rule *SurveillanceRule,
) DetectionResult {
	priceSet := make(map[string]int)
	var placeCount, cancelCount int

	for _, ev := range history {
		if ev.UserID != current.UserID || ev.Symbol != current.Symbol {
			continue
		}
		switch ev.EventType {
		case EventPlace:
			placeCount++
			priceSet[ev.Price.String()]++
		case EventCancel:
			cancelCount++
		}
	}

	distinctPrices := len(priceSet)
	if distinctPrices < 3 || placeCount < 5 {
		return DetectionResult{}
	}

	cancelRatio := decimal.NewFromInt(int64(cancelCount)).Div(decimal.NewFromInt(int64(placeCount)))
	if cancelRatio.LessThan(rule.MinCancelRatio) {
		return DetectionResult{}
	}

	confidence := float64(distinctPrices) / 10.0 * 0.4
	confidence += cancelRatio.InexactFloat64() * 0.5
	if confidence > 1.0 {
		confidence = 1.0
	}

	return DetectionResult{
		Triggered:  true,
		Type:       ManipulationLayering,
		Severity:   e.scoreSeverity(confidence),
		Confidence: confidence,
		Reason: fmt.Sprintf(
			"Layering detected: distinct_prices=%d, cancel_ratio=%.2f, orders=%d",
			distinctPrices, cancelRatio.InexactFloat64(), placeCount,
		),
	}
}

// detectWashTrading 对倒/洗售检测
// 算法：检测同一用户在买卖双方的自成交行为
// 标记同一用户在窗口内既有买入成交又有卖出成交的情况
func (e *DetectionEngine) detectWashTrading(
	current OrderEvent,
	history []OrderEvent,
	rule *SurveillanceRule,
) DetectionResult {
	if current.EventType != EventFill {
		return DetectionResult{}
	}

	var buyFills, sellFills int
	var buyVolume, sellVolume decimal.Decimal

	for _, ev := range history {
		if ev.UserID != current.UserID || ev.Symbol != current.Symbol {
			continue
		}
		if ev.EventType != EventFill {
			continue
		}
		switch ev.Side {
		case "BUY":
			buyFills++
			buyVolume = buyVolume.Add(ev.Quantity)
		case "SELL":
			sellFills++
			sellVolume = sellVolume.Add(ev.Quantity)
		}
	}

	if buyFills == 0 || sellFills == 0 {
		return DetectionResult{}
	}

	minVolume := buyVolume
	if sellVolume.LessThan(minVolume) {
		minVolume = sellVolume
	}
	maxVolume := buyVolume
	if sellVolume.GreaterThan(maxVolume) {
		maxVolume = sellVolume
	}

	selfTradeRatio := decimal.Decimal{}
	if !maxVolume.IsZero() {
		selfTradeRatio = minVolume.Div(maxVolume)
	}

	if selfTradeRatio.LessThan(rule.MaxWashVolumeRatio) {
		return DetectionResult{}
	}

	confidence := selfTradeRatio.InexactFloat64() * 0.7
	if buyFills > 3 && sellFills > 3 {
		confidence += 0.2
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	return DetectionResult{
		Triggered:  true,
		Type:       ManipulationWashTrading,
		Severity:   e.scoreSeverity(confidence),
		Confidence: confidence,
		Reason: fmt.Sprintf(
			"Wash trading detected: self_trade_ratio=%.2f, buy_fills=%d, sell_fills=%d",
			selfTradeRatio.InexactFloat64(), buyFills, sellFills,
		),
	}
}

// detectFrontRunning 抢先交易检测
// 算法：检测某用户在大额订单出现前短时间内方向一致的小额下单
func (e *DetectionEngine) detectFrontRunning(
	current OrderEvent,
	history []OrderEvent,
	rule *SurveillanceRule,
) DetectionResult {
	if current.EventType != EventPlace {
		return DetectionResult{}
	}

	largeThreshold := rule.Threshold

	// 检查当前订单是否是大额订单
	if current.Quantity.LessThan(largeThreshold) {
		return DetectionResult{}
	}

	// 查找窗口内其他用户在此订单前的同方向小额下单
	var suspicious []OrderEvent
	for _, ev := range history {
		if ev.UserID == current.UserID || ev.Symbol != current.Symbol {
			continue
		}
		if ev.EventType != EventPlace {
			continue
		}
		if ev.Side != current.Side {
			continue
		}
		if ev.Quantity.GreaterThanOrEqual(largeThreshold) {
			continue
		}
		timeDiff := current.Timestamp.Sub(ev.Timestamp)
		if timeDiff > 0 && timeDiff < 5*time.Second {
			suspicious = append(suspicious, ev)
		}
	}

	if len(suspicious) == 0 {
		return DetectionResult{}
	}

	confidence := float64(len(suspicious)) * 0.25
	if confidence > 1.0 {
		confidence = 1.0
	}

	return DetectionResult{
		Triggered:  true,
		Type:       ManipulationFrontRun,
		Severity:   e.scoreSeverity(confidence),
		Confidence: confidence,
		Reason: fmt.Sprintf(
			"Front running suspected: %d small orders placed before large order (%s)",
			len(suspicious), current.Quantity.String(),
		),
	}
}

// detectClosingPriceManip 尾盘操纵检测
// 算法：检测收盘前最后 N 秒内的集中交易行为
func (e *DetectionEngine) detectClosingPriceManip(
	current OrderEvent,
	history []OrderEvent,
	rule *SurveillanceRule,
) DetectionResult {
	// 简化实现：检测用户在窗口最后 30 秒内的大量下单
	windowEnd := current.Timestamp
	windowStart := windowEnd.Add(-30 * time.Second)

	var recentOrders int
	var totalVolume decimal.Decimal

	for _, ev := range history {
		if ev.UserID != current.UserID || ev.Symbol != current.Symbol {
			continue
		}
		if ev.EventType != EventPlace {
			continue
		}
		if ev.Timestamp.Before(windowStart) || ev.Timestamp.After(windowEnd) {
			continue
		}
		recentOrders++
		totalVolume = totalVolume.Add(ev.Quantity)
	}

	if recentOrders < 5 {
		return DetectionResult{}
	}

	if totalVolume.LessThan(rule.Threshold) {
		return DetectionResult{}
	}

	confidence := float64(recentOrders) / 20.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return DetectionResult{
		Triggered:  true,
		Type:       ManipulationClosingPrice,
		Severity:   e.scoreSeverity(confidence),
		Confidence: confidence,
		Reason: fmt.Sprintf(
			"Closing price manipulation: %d orders, volume=%s in last 30s",
			recentOrders, totalVolume.String(),
		),
	}
}

// detectPumpAndDump 拉高抛售检测
// 算法：检测用户先大量买入再大量卖出的模式
func (e *DetectionEngine) detectPumpAndDump(
	current OrderEvent,
	history []OrderEvent,
	rule *SurveillanceRule,
) DetectionResult {
	if current.EventType != EventFill || current.Side != "SELL" {
		return DetectionResult{}
	}

	var buyVolume, sellVolume decimal.Decimal
	var buyPhaseEnd time.Time

	for _, ev := range history {
		if ev.UserID != current.UserID || ev.Symbol != current.Symbol {
			continue
		}
		if ev.EventType != EventFill {
			continue
		}
		switch ev.Side {
		case "BUY":
			buyVolume = buyVolume.Add(ev.Quantity)
			if ev.Timestamp.After(buyPhaseEnd) {
				buyPhaseEnd = ev.Timestamp
			}
		case "SELL":
			if ev.Timestamp.After(buyPhaseEnd) {
				sellVolume = sellVolume.Add(ev.Quantity)
			}
		}
	}

	if buyVolume.IsZero() || sellVolume.IsZero() {
		return DetectionResult{}
	}

	dumpRatio := sellVolume.Div(buyVolume)
	if dumpRatio.LessThan(decimal.NewFromFloat(0.7)) {
		return DetectionResult{}
	}

	confidence := dumpRatio.InexactFloat64() * 0.6
	if buyVolume.GreaterThan(rule.Threshold) {
		confidence += 0.3
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	return DetectionResult{
		Triggered:  true,
		Type:       ManipulationPumpAndDump,
		Severity:   e.scoreSeverity(confidence),
		Confidence: confidence,
		Reason: fmt.Sprintf(
			"Pump and dump: buy_volume=%s, sell_volume=%s, dump_ratio=%.2f",
			buyVolume.String(), sellVolume.String(), dumpRatio.InexactFloat64(),
		),
	}
}

// scoreSeverity 根据置信度计算严重程度
func (e *DetectionEngine) scoreSeverity(confidence float64) AlertSeverity {
	switch {
	case confidence >= 0.9:
		return SeverityCritical
	case confidence >= 0.7:
		return SeverityHigh
	case confidence >= 0.5:
		return SeverityMedium
	default:
		return SeverityLow
	}
}

// filterWindow 过滤时间窗口内的事件
func filterWindow(events []OrderEvent, now time.Time, window time.Duration) []OrderEvent {
	cutoff := now.Add(-window)
	var result []OrderEvent
	for _, ev := range events {
		if ev.Timestamp.After(cutoff) {
			result = append(result, ev)
		}
	}
	return result
}

// AnalyzeTradingPattern 分析用户交易模式
// 说明：基于历史订单事件统计撤单率、订单存活时间、自成交率等指标
func (e *DetectionEngine) AnalyzeTradingPattern(events []OrderEvent) *TradingPattern {
	if len(events) == 0 {
		return &TradingPattern{RiskLevel: "NONE"}
	}

	var userID, symbol string
	var placeCount, cancelCount, fillCount int64
	var totalLifetime time.Duration
	var buyFills, sellFills int64
	placeMap := make(map[string]time.Time)

	for _, ev := range events {
		if userID == "" {
			userID = ev.UserID
		}
		if symbol == "" {
			symbol = ev.Symbol
		}

		switch ev.EventType {
		case EventPlace:
			placeCount++
			placeMap[ev.OrderID] = ev.Timestamp
		case EventCancel:
			cancelCount++
			if pt, ok := placeMap[ev.OrderID]; ok {
				totalLifetime += ev.Timestamp.Sub(pt)
			}
		case EventFill:
			fillCount++
			switch ev.Side {
			case "BUY":
				buyFills++
			case "SELL":
				sellFills++
			}
		}
	}

	cancelRatio := float64(0)
	if placeCount > 0 {
		cancelRatio = float64(cancelCount) / float64(placeCount)
	}

	avgLifetimeMs := float64(0)
	if cancelCount > 0 {
		avgLifetimeMs = float64(totalLifetime.Milliseconds()) / float64(cancelCount)
	}

	selfTradeRatio := float64(0)
	if buyFills > 0 && sellFills > 0 {
		minFills := buyFills
		if sellFills < minFills {
			minFills = sellFills
		}
		maxFills := buyFills
		if sellFills > maxFills {
			maxFills = sellFills
		}
		selfTradeRatio = float64(minFills) / float64(maxFills)
	}

	riskLevel := "LOW"
	switch {
	case cancelRatio > 0.8 || selfTradeRatio > 0.7:
		riskLevel = "CRITICAL"
	case cancelRatio > 0.6 || selfTradeRatio > 0.5:
		riskLevel = "HIGH"
	case cancelRatio > 0.4 || selfTradeRatio > 0.3:
		riskLevel = "MEDIUM"
	}

	return &TradingPattern{
		UserID:             userID,
		Symbol:             symbol,
		CancelRatio:        cancelRatio,
		AvgOrderLifetimeMs: avgLifetimeMs,
		SelfTradeRatio:     selfTradeRatio,
		TotalOrders:        placeCount,
		TotalCancels:       cancelCount,
		TotalFills:         fillCount,
		RiskLevel:          riskLevel,
	}
}
