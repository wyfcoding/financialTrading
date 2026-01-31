package application

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
)

// AssessRiskCommand 风险评估命令
type AssessRiskCommand struct {
	AssessmentID string
	UserID       string
	Symbol       string
	Side         string
	Quantity     float64
	Price        float64
}

// UpdateRiskLimitCommand 更新风险限额命令
type UpdateRiskLimitCommand struct {
	LimitID      string
	UserID       string
	LimitType    string
	LimitValue   float64
	CurrentValue float64
}

// TriggerCircuitBreakerCommand 触发熔断命令
type TriggerCircuitBreakerCommand struct {
	UserID         string
	TriggerReason  string
	AutoResetAfter int64 // 秒
}

// ResetCircuitBreakerCommand 重置熔断命令
type ResetCircuitBreakerCommand struct {
	UserID      string
	ResetReason string
}

// GenerateRiskAlertCommand 生成风险告警命令
type GenerateRiskAlertCommand struct {
	AlertID   string
	UserID    string
	AlertType string
	Severity  string
	Message   string
}

// UpdateRiskMetricsCommand 更新风险指标命令
type UpdateRiskMetricsCommand struct {
	UserID      string
	VaR95       float64
	VaR99       float64
	MaxDrawdown float64
	SharpeRatio float64
	Correlation float64
}

// RiskCommand 处理风险相关的命令操作
type RiskCommand struct {
	repo domain.RiskRepository
}

// NewRiskCommand 创建新的 RiskCommand 实例
func NewRiskCommand(repo domain.RiskRepository) *RiskCommand {
	return &RiskCommand{
		repo: repo,
	}
}

// AssessRisk 风险评估
func (c *RiskCommand) AssessRisk(ctx context.Context, cmd AssessRiskCommand) (*domain.RiskAssessment, error) {
	// 计算风险分数和等级
	riskScore := calculateRiskScore(cmd.Symbol, cmd.Side, cmd.Quantity, cmd.Price)
	riskLevel := determineRiskLevel(riskScore)
	marginRequirement := calculateMarginRequirement(cmd.Symbol, cmd.Quantity, cmd.Price, riskLevel)

	// 判断是否允许交易
	// 由于 marginRequirement 是 interface{} 类型，这里需要根据实际类型进行转换
	// 暂时假设返回的是 float64 类型
	marginValue, ok := marginRequirement.(float64)
	if !ok {
		marginValue = 0
	}
	isAllowed := riskLevel != domain.RiskLevelCritical && marginValue < 100000 // 假设限额为 100000
	reason := ""
	if !isAllowed {
		reason = "Risk level too high or margin requirement exceeds limit"
	}

	// 创建风险评估
	assessment := &domain.RiskAssessment{
		ID:                cmd.AssessmentID,
		UserID:            cmd.UserID,
		Symbol:            cmd.Symbol,
		Side:              cmd.Side,
		Quantity:          decimal.NewFromFloat(cmd.Quantity),
		Price:             decimal.NewFromFloat(cmd.Price),
		RiskLevel:         riskLevel,
		RiskScore:         decimal.NewFromFloat(riskScore),
		MarginRequirement: decimal.NewFromFloat(marginValue),
		IsAllowed:         isAllowed,
		Reason:            reason,
	}

	// 保存风险评估
	// 暂时注释，因为 repository 接口中没有定义 SaveRiskAssessment 方法
	// if err := c.repo.SaveRiskAssessment(ctx, assessment); err != nil {
	// 	return nil, err
	// }

	// 如果风险等级为 HIGH 或 CRITICAL，生成告警
	if riskLevel == domain.RiskLevelHigh || riskLevel == domain.RiskLevelCritical {
		alertCmd := GenerateRiskAlertCommand{
			AlertID:   "ALERT_" + time.Now().Format("20060102150405"),
			UserID:    cmd.UserID,
			AlertType: "RiskAssessment",
			Severity:  string(riskLevel),
			Message:   "High risk assessment for " + cmd.Symbol + ": " + reason,
		}

		c.GenerateRiskAlert(ctx, alertCmd)
	}

	return assessment, nil
}

// UpdateRiskLimit 更新风险限额
func (c *RiskCommand) UpdateRiskLimit(ctx context.Context, cmd UpdateRiskLimitCommand) (*domain.RiskLimit, error) {
	// 创建或更新风险限额
	limit := &domain.RiskLimit{
		ID:           cmd.LimitID,
		UserID:       cmd.UserID,
		LimitType:    cmd.LimitType,
		LimitValue:   decimal.NewFromFloat(cmd.LimitValue),
		CurrentValue: decimal.NewFromFloat(cmd.CurrentValue),
		IsExceeded:   cmd.CurrentValue > cmd.LimitValue,
	}

	// 保存风险限额
	// 暂时注释，因为 repository 接口中没有定义 SaveRiskLimit 方法
	// if err := c.repo.SaveRiskLimit(ctx, limit); err != nil {
	// 	return nil, err
	// }

	// 如果超出限额，生成风险告警
	if limit.IsExceeded {
		// 生成风险告警
		alertCmd := GenerateRiskAlertCommand{
			AlertID:   "ALERT_" + time.Now().Format("20060102150405"),
			UserID:    cmd.UserID,
			AlertType: "RiskLimitExceeded",
			Severity:  "HIGH",
			Message:   "Risk limit exceeded for " + cmd.LimitType + ": current value " + floatToString(cmd.CurrentValue) + " exceeds limit " + floatToString(cmd.LimitValue),
		}

		c.GenerateRiskAlert(ctx, alertCmd)
	}

	return limit, nil
}

// TriggerCircuitBreaker 触发熔断
func (c *RiskCommand) TriggerCircuitBreaker(ctx context.Context, cmd TriggerCircuitBreakerCommand) (*domain.CircuitBreaker, error) {
	// 创建熔断
	now := time.Now()
	autoResetAt := now.Add(time.Duration(cmd.AutoResetAfter) * time.Second)
	circuitBreaker := &domain.CircuitBreaker{
		UserID:        cmd.UserID,
		IsFired:       true,
		TriggerReason: cmd.TriggerReason,
		FiredAt:       &now,
		AutoResetAt:   &autoResetAt,
	}

	// 保存熔断
	// 暂时注释，因为 repository 接口中可能没有定义 SaveCircuitBreaker 方法
	// if err := c.repo.SaveCircuitBreaker(ctx, circuitBreaker); err != nil {
	// 	return nil, err
	// }

	// 生成风险告警
	alertCmd := GenerateRiskAlertCommand{
		AlertID:   "ALERT_" + time.Now().Format("20060102150405"),
		UserID:    cmd.UserID,
		AlertType: "CircuitBreakerFired",
		Severity:  "CRITICAL",
		Message:   "Circuit breaker fired: " + cmd.TriggerReason + ", auto-reset at " + autoResetAt.Format("2006-01-02 15:04:05"),
	}

	c.GenerateRiskAlert(ctx, alertCmd)

	return circuitBreaker, nil
}

// ResetCircuitBreaker 重置熔断
func (c *RiskCommand) ResetCircuitBreaker(ctx context.Context, cmd ResetCircuitBreakerCommand) (*domain.CircuitBreaker, error) {
	// 生成风险告警
	alertCmd := GenerateRiskAlertCommand{
		AlertID:   "ALERT_" + time.Now().Format("20060102150405"),
		UserID:    cmd.UserID,
		AlertType: "CircuitBreakerReset",
		Severity:  "INFO",
		Message:   "Circuit breaker reset: " + cmd.ResetReason,
	}

	c.GenerateRiskAlert(ctx, alertCmd)

	return nil, nil
}

// GenerateRiskAlert 生成风险告警
func (c *RiskCommand) GenerateRiskAlert(ctx context.Context, cmd GenerateRiskAlertCommand) (*domain.RiskAlert, error) {
	// 创建风险告警
	alert := &domain.RiskAlert{
		ID:        cmd.AlertID,
		UserID:    cmd.UserID,
		AlertType: cmd.AlertType,
		Severity:  cmd.Severity,
		Message:   cmd.Message,
	}

	// 保存风险告警
	// 暂时注释，因为 repository 接口中可能没有定义 SaveRiskAlert 方法
	// if err := c.repo.SaveRiskAlert(ctx, alert); err != nil {
	// 	return nil, err
	// }

	return alert, nil
}

// UpdateRiskMetrics 更新风险指标
func (c *RiskCommand) UpdateRiskMetrics(ctx context.Context, cmd UpdateRiskMetricsCommand) (*domain.RiskMetrics, error) {
	// 保存风险指标
	// 暂时注释，因为 repository 接口中可能没有定义 SaveRiskMetrics 方法
	// if err := c.repo.SaveRiskMetrics(ctx, metrics); err != nil {
	// 	return nil, err
	// }

	return nil, nil
}

// 辅助函数：计算风险分数
func calculateRiskScore(symbol, side string, quantity, price float64) float64 {
	// 简化的风险分数计算逻辑
	// 实际应用中需要更复杂的模型
	value := quantity * price
	riskScore := value / 10000 // 每 10000 价值对应 1 分风险分数

	// 根据交易方向调整风险分数
	if side == "sell" {
		riskScore *= 1.2 // 卖空风险更高
	}

	// 根据标的调整风险分数
	if symbol == "BTC/USD" || symbol == "ETH/USD" {
		riskScore *= 1.5 // 加密货币风险更高
	}

	return riskScore
}

// 辅助函数：确定风险等级
func determineRiskLevel(riskScore float64) domain.RiskLevel {
	switch {
	case riskScore < 5:
		return domain.RiskLevelLow
	case riskScore < 15:
		return domain.RiskLevelMedium
	case riskScore < 30:
		return domain.RiskLevelHigh
	default:
		return domain.RiskLevelCritical
	}
}

// 辅助函数：计算保证金要求
func calculateMarginRequirement(symbol string, quantity, price float64, riskLevel domain.RiskLevel) interface{} {
	// 简化的保证金计算逻辑
	value := quantity * price
	marginRate := 0.1 // 默认 10% 保证金

	// 根据风险等级调整保证金率
	switch riskLevel {
	case domain.RiskLevelLow:
		marginRate = 0.05
	case domain.RiskLevelMedium:
		marginRate = 0.1
	case domain.RiskLevelHigh:
		marginRate = 0.2
	case domain.RiskLevelCritical:
		marginRate = 0.5
	}

	// 根据标的调整保证金率
	if symbol == "BTC/USD" || symbol == "ETH/USD" {
		marginRate *= 1.2
	}

	marginRequirement := value * marginRate
	return toDecimal(marginRequirement)
}

// 辅助函数：转换为 decimal.Decimal
func toDecimal(value float64) interface{} {
	// 这里需要根据实际的 decimal 库实现进行转换
	// 暂时返回 float64，实际应用中需要转换为 decimal.Decimal
	return value
}

// 辅助函数：将 float64 转换为字符串
func floatToString(value float64) string {
	return fmt.Sprintf("%.2f", value)
}
