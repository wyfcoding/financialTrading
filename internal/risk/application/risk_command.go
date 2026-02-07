package application

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	accountv1 "github.com/wyfcoding/financialtrading/go-api/account/v1"
	positionv1 "github.com/wyfcoding/financialtrading/go-api/position/v1"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
	"github.com/wyfcoding/pkg/contextx"
	"github.com/wyfcoding/pkg/idgen"
)

// RiskCommandService 处理风险相关的命令操作（Commands）。
type RiskCommandService struct {
	repo           domain.RiskRepository
	readRepo       domain.RiskReadRepository
	accountClient  accountv1.AccountServiceClient
	positionClient positionv1.PositionServiceClient
	publisher      domain.EventPublisher
}

// NewRiskCommandService 创建新的 RiskCommandService 实例
func NewRiskCommandService(
	repo domain.RiskRepository,
	readRepo domain.RiskReadRepository,
	accClient accountv1.AccountServiceClient,
	posClient positionv1.PositionServiceClient,
	publisher domain.EventPublisher,
) *RiskCommandService {
	return &RiskCommandService{
		repo:           repo,
		readRepo:       readRepo,
		accountClient:  accClient,
		positionClient: posClient,
		publisher:      publisher,
	}
}

// AssessRisk 风险评估
func (s *RiskCommandService) AssessRisk(ctx context.Context, cmd AssessRiskCommand) (*RiskAssessmentDTO, error) {
	assessmentID := cmd.AssessmentID
	if assessmentID == "" {
		assessmentID = fmt.Sprintf("RA-%d", idgen.GenID())
	}

	riskScore := calculateRiskScore(cmd.Symbol, cmd.Side, cmd.Quantity, cmd.Price)
	riskLevel := determineRiskLevel(riskScore)
	marginRequirement := calculateMarginRequirement(cmd.Symbol, cmd.Quantity, cmd.Price, riskLevel)

	isAllowed := riskLevel != domain.RiskLevelCritical
	reason := ""

	// 1. 检查最大持仓限额 (MAX_POSITION)
	if isAllowed && s.positionClient != nil {
		posLimit, _ := s.repo.GetLimitByUserIDAndType(ctx, cmd.UserID, domain.LimitTypeMaxPosition)
		if posLimit != nil && !posLimit.LimitValue.IsZero() {
			posResp, err := s.positionClient.GetPositions(ctx, &positionv1.GetPositionsRequest{UserId: cmd.UserID})
			if err == nil {
				symbolQty := decimal.Zero
				for _, p := range posResp.Positions {
					if p.Symbol == cmd.Symbol {
						q, _ := decimal.NewFromString(p.Quantity)
						symbolQty = symbolQty.Add(q)
					}
				}
				if symbolQty.Add(decimal.NewFromFloat(cmd.Quantity)).GreaterThan(posLimit.LimitValue) {
					isAllowed = false
					reason = fmt.Sprintf("Exceeds maximum position limit for %s", cmd.Symbol)
				}
			}
		}
	}

	// 2. 检查信用额度 (CREDIT_LIMIT)
	if isAllowed && s.accountClient != nil {
		creditLimit, _ := s.repo.GetLimitByUserIDAndType(ctx, cmd.UserID, domain.LimitTypeCreditLimit)
		if creditLimit != nil && !creditLimit.LimitValue.IsZero() {
			accResp, err := s.accountClient.GetAccount(ctx, &accountv1.GetAccountRequest{UserId: cmd.UserID})
			if err == nil && accResp != nil && accResp.Account != nil {
				borrowed, _ := decimal.NewFromString(accResp.Account.BorrowedAmount)
				if borrowed.Add(marginRequirement).GreaterThan(creditLimit.LimitValue) {
					isAllowed = false
					reason = "Exceeds total credit limit"
				}
			}
		}
	}

	if !isAllowed && reason == "" {
		reason = "Risk level too high or system limit reached"
	}

	assessment := &domain.RiskAssessment{
		ID:                assessmentID,
		UserID:            cmd.UserID,
		Symbol:            cmd.Symbol,
		Side:              cmd.Side,
		Quantity:          decimal.NewFromFloat(cmd.Quantity),
		Price:             decimal.NewFromFloat(cmd.Price),
		RiskLevel:         riskLevel,
		RiskScore:         decimal.NewFromFloat(riskScore),
		MarginRequirement: marginRequirement,
		IsAllowed:         isAllowed,
		Reason:            reason,
	}

	var alertToPublish *domain.RiskAlert

	err := s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.SaveAssessment(txCtx, assessment); err != nil {
			return err
		}

		if s.publisher != nil {
			event := domain.RiskAssessmentCreatedEvent{
				AssessmentID:      assessment.ID,
				UserID:            assessment.UserID,
				Symbol:            assessment.Symbol,
				Side:              assessment.Side,
				Quantity:          assessment.Quantity.InexactFloat64(),
				Price:             assessment.Price.InexactFloat64(),
				RiskLevel:         assessment.RiskLevel,
				RiskScore:         assessment.RiskScore.InexactFloat64(),
				MarginRequirement: assessment.MarginRequirement.InexactFloat64(),
				IsAllowed:         assessment.IsAllowed,
				Reason:            assessment.Reason,
				CreatedAt:         time.Now().Unix(),
				OccurredOn:        time.Now(),
			}
			if err := s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.RiskAssessmentCreatedEventType, assessment.ID, event); err != nil {
				return err
			}
		}

		if riskLevel == domain.RiskLevelHigh || riskLevel == domain.RiskLevelCritical {
			alertToPublish = &domain.RiskAlert{
				ID:        fmt.Sprintf("ALERT-%d", idgen.GenID()),
				UserID:    cmd.UserID,
				AlertType: "RiskAssessment",
				Severity:  string(riskLevel),
				Message:   "High risk assessment for " + cmd.Symbol + ": " + reason,
			}
			if err := s.repo.SaveAlert(txCtx, alertToPublish); err != nil {
				return err
			}
			if s.publisher != nil {
				alertEvent := domain.RiskAlertGeneratedEvent{
					AlertID:     alertToPublish.ID,
					UserID:      alertToPublish.UserID,
					AlertType:   alertToPublish.AlertType,
					Severity:    alertToPublish.Severity,
					Message:     alertToPublish.Message,
					GeneratedAt: time.Now().Unix(),
					OccurredOn:  time.Now(),
				}
				if err := s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.RiskAlertGeneratedEventType, alertToPublish.ID, alertEvent); err != nil {
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return toRiskAssessmentDTO(assessment), nil
}

// UpdateRiskLimit 更新风险限额
func (s *RiskCommandService) UpdateRiskLimit(ctx context.Context, cmd UpdateRiskLimitCommand) (*RiskLimitDTO, error) {
	limitID := cmd.LimitID
	if limitID == "" {
		limitID = fmt.Sprintf("RL-%d", idgen.GenID())
	}

	limit := &domain.RiskLimit{
		ID:           limitID,
		UserID:       cmd.UserID,
		LimitType:    cmd.LimitType,
		LimitValue:   decimal.NewFromFloat(cmd.LimitValue),
		CurrentValue: decimal.NewFromFloat(cmd.CurrentValue),
		IsExceeded:   cmd.CurrentValue > cmd.LimitValue,
	}

	var exceededAlert *domain.RiskAlert

	err := s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.SaveLimit(txCtx, limit); err != nil {
			return err
		}
		if s.publisher != nil {
			updateEvent := domain.RiskLimitUpdatedEvent{
				LimitID:      limit.ID,
				UserID:       limit.UserID,
				LimitType:    limit.LimitType,
				LimitValue:   limit.LimitValue.InexactFloat64(),
				CurrentValue: limit.CurrentValue.InexactFloat64(),
				IsExceeded:   limit.IsExceeded,
				UpdatedAt:    time.Now().Unix(),
				OccurredOn:   time.Now(),
			}
			if err := s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.RiskLimitUpdatedEventType, limit.ID, updateEvent); err != nil {
				return err
			}
		}

		if limit.IsExceeded {
			exceededBy := limit.CurrentValue.Sub(limit.LimitValue)
			exceededEvent := domain.RiskLimitExceededEvent{
				LimitID:      limit.ID,
				UserID:       limit.UserID,
				LimitType:    limit.LimitType,
				LimitValue:   limit.LimitValue.InexactFloat64(),
				CurrentValue: limit.CurrentValue.InexactFloat64(),
				ExceededBy:   exceededBy.InexactFloat64(),
				OccurredAt:   time.Now().Unix(),
				OccurredOn:   time.Now(),
			}
			if s.publisher != nil {
				if err := s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.RiskLimitExceededEventType, limit.ID, exceededEvent); err != nil {
					return err
				}
			}

			exceededAlert = &domain.RiskAlert{
				ID:        fmt.Sprintf("ALERT-%d", idgen.GenID()),
				UserID:    cmd.UserID,
				AlertType: "RiskLimitExceeded",
				Severity:  "HIGH",
				Message:   "Risk limit exceeded for " + cmd.LimitType + ": current value " + floatToString(cmd.CurrentValue) + " exceeds limit " + floatToString(cmd.LimitValue),
			}
			if err := s.repo.SaveAlert(txCtx, exceededAlert); err != nil {
				return err
			}
			if s.publisher != nil {
				alertEvent := domain.RiskAlertGeneratedEvent{
					AlertID:     exceededAlert.ID,
					UserID:      exceededAlert.UserID,
					AlertType:   exceededAlert.AlertType,
					Severity:    exceededAlert.Severity,
					Message:     exceededAlert.Message,
					GeneratedAt: time.Now().Unix(),
					OccurredOn:  time.Now(),
				}
				if err := s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.RiskAlertGeneratedEventType, exceededAlert.ID, alertEvent); err != nil {
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if s.readRepo != nil {
		_ = s.readRepo.SaveLimit(ctx, limit.UserID, limit)
	}

	return toRiskLimitDTO(limit), nil
}

// TriggerCircuitBreaker 触发熔断
func (s *RiskCommandService) TriggerCircuitBreaker(ctx context.Context, cmd TriggerCircuitBreakerCommand) (*CircuitBreakerDTO, error) {
	now := time.Now()
	autoResetAt := now.Add(time.Duration(cmd.AutoResetAfter) * time.Second)
	cb := &domain.CircuitBreaker{
		UserID:        cmd.UserID,
		IsFired:       true,
		TriggerReason: cmd.TriggerReason,
		FiredAt:       &now,
		AutoResetAt:   &autoResetAt,
	}

	alert := &domain.RiskAlert{
		ID:        fmt.Sprintf("ALERT-%d", idgen.GenID()),
		UserID:    cmd.UserID,
		AlertType: "CircuitBreakerFired",
		Severity:  "CRITICAL",
		Message:   "Circuit breaker fired: " + cmd.TriggerReason + ", auto-reset at " + autoResetAt.Format("2006-01-02 15:04:05"),
	}

	err := s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.SaveCircuitBreaker(txCtx, cb); err != nil {
			return err
		}
		if err := s.repo.SaveAlert(txCtx, alert); err != nil {
			return err
		}
		if s.publisher == nil {
			return nil
		}

		firedEvent := domain.CircuitBreakerFiredEvent{
			UserID:        cb.UserID,
			TriggerReason: cb.TriggerReason,
			FiredAt:       now.Unix(),
			AutoResetAt:   autoResetAt.Unix(),
			OccurredOn:    time.Now(),
		}
		if err := s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.CircuitBreakerFiredEventType, cb.UserID, firedEvent); err != nil {
			return err
		}

		alertEvent := domain.RiskAlertGeneratedEvent{
			AlertID:     alert.ID,
			UserID:      alert.UserID,
			AlertType:   alert.AlertType,
			Severity:    alert.Severity,
			Message:     alert.Message,
			GeneratedAt: time.Now().Unix(),
			OccurredOn:  time.Now(),
		}
		return s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.RiskAlertGeneratedEventType, alert.ID, alertEvent)
	})
	if err != nil {
		return nil, err
	}

	if s.readRepo != nil {
		_ = s.readRepo.SaveCircuitBreaker(ctx, cb.UserID, cb)
	}

	return toCircuitBreakerDTO(cb), nil
}

// ResetCircuitBreaker 重置熔断
func (s *RiskCommandService) ResetCircuitBreaker(ctx context.Context, cmd ResetCircuitBreakerCommand) (*CircuitBreakerDTO, error) {
	now := time.Now()
	alert := &domain.RiskAlert{
		ID:        fmt.Sprintf("ALERT-%d", idgen.GenID()),
		UserID:    cmd.UserID,
		AlertType: "CircuitBreakerReset",
		Severity:  "INFO",
		Message:   "Circuit breaker reset: " + cmd.ResetReason,
	}

	err := s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.SaveAlert(txCtx, alert); err != nil {
			return err
		}
		if s.publisher == nil {
			return nil
		}
		resetEvent := domain.CircuitBreakerResetEvent{
			UserID:      cmd.UserID,
			ResetReason: cmd.ResetReason,
			ResetAt:     now.Unix(),
			OccurredOn:  time.Now(),
		}
		if err := s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.CircuitBreakerResetEventType, cmd.UserID, resetEvent); err != nil {
			return err
		}

		alertEvent := domain.RiskAlertGeneratedEvent{
			AlertID:     alert.ID,
			UserID:      alert.UserID,
			AlertType:   alert.AlertType,
			Severity:    alert.Severity,
			Message:     alert.Message,
			GeneratedAt: time.Now().Unix(),
			OccurredOn:  time.Now(),
		}
		return s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.RiskAlertGeneratedEventType, alert.ID, alertEvent)
	})
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// UpdateRiskMetrics 更新风险指标
func (s *RiskCommandService) UpdateRiskMetrics(ctx context.Context, cmd UpdateRiskMetricsCommand) (*RiskMetricsDTO, error) {
	metrics := &domain.RiskMetrics{
		UserID:      cmd.UserID,
		VaR95:       decimal.NewFromFloat(cmd.VaR95),
		VaR99:       decimal.NewFromFloat(cmd.VaR99),
		MaxDrawdown: decimal.NewFromFloat(cmd.MaxDrawdown),
		SharpeRatio: decimal.NewFromFloat(cmd.SharpeRatio),
		Correlation: decimal.NewFromFloat(cmd.Correlation),
	}

	err := s.repo.WithTx(ctx, func(txCtx context.Context) error {
		oldMetrics, _ := s.repo.GetMetrics(txCtx, cmd.UserID)
		if err := s.repo.SaveMetrics(txCtx, metrics); err != nil {
			return err
		}
		if s.publisher == nil {
			return nil
		}

		oldVar95 := 0.0
		oldMaxDrawdown := 0.0
		oldSharpe := 0.0
		if oldMetrics != nil {
			oldVar95 = oldMetrics.VaR95.InexactFloat64()
			oldMaxDrawdown = oldMetrics.MaxDrawdown.InexactFloat64()
			oldSharpe = oldMetrics.SharpeRatio.InexactFloat64()
		}

		updateEvent := domain.RiskMetricsUpdatedEvent{
			UserID:         metrics.UserID,
			OldVaR95:       oldVar95,
			NewVaR95:       metrics.VaR95.InexactFloat64(),
			OldMaxDrawdown: oldMaxDrawdown,
			NewMaxDrawdown: metrics.MaxDrawdown.InexactFloat64(),
			OldSharpeRatio: oldSharpe,
			NewSharpeRatio: metrics.SharpeRatio.InexactFloat64(),
			UpdatedAt:      time.Now().Unix(),
			OccurredOn:     time.Now(),
		}
		return s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.RiskMetricsUpdatedEventType, metrics.UserID, updateEvent)
	})
	if err != nil {
		return nil, err
	}

	if s.readRepo != nil {
		_ = s.readRepo.SaveMetrics(ctx, metrics.UserID, metrics)
	}

	return toRiskMetricsDTO(metrics), nil
}

// GenerateRiskAlert 生成风险告警
func (s *RiskCommandService) GenerateRiskAlert(ctx context.Context, cmd GenerateRiskAlertCommand) (*RiskAlertDTO, error) {
	alertID := cmd.AlertID
	if alertID == "" {
		alertID = fmt.Sprintf("ALERT-%d", idgen.GenID())
	}

	alert := &domain.RiskAlert{
		ID:        alertID,
		UserID:    cmd.UserID,
		AlertType: cmd.AlertType,
		Severity:  cmd.Severity,
		Message:   cmd.Message,
	}

	err := s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.SaveAlert(txCtx, alert); err != nil {
			return err
		}
		if s.publisher == nil {
			return nil
		}
		alertEvent := domain.RiskAlertGeneratedEvent{
			AlertID:     alert.ID,
			UserID:      alert.UserID,
			AlertType:   alert.AlertType,
			Severity:    alert.Severity,
			Message:     alert.Message,
			GeneratedAt: time.Now().Unix(),
			OccurredOn:  time.Now(),
		}
		return s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.RiskAlertGeneratedEventType, alert.ID, alertEvent)
	})
	if err != nil {
		return nil, err
	}

	return toRiskAlertDTO(alert), nil
}

// 辅助函数：计算风险分数
func calculateRiskScore(symbol, side string, quantity, price float64) float64 {
	value := quantity * price
	riskScore := value / 10000

	if side == "sell" {
		riskScore *= 1.2
	}

	if symbol == "BTC/USD" || symbol == "ETH/USD" {
		riskScore *= 1.5
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
func calculateMarginRequirement(symbol string, quantity, price float64, riskLevel domain.RiskLevel) decimal.Decimal {
	value := decimal.NewFromFloat(quantity).Mul(decimal.NewFromFloat(price))
	marginRate := decimal.NewFromFloat(0.1)

	switch riskLevel {
	case domain.RiskLevelLow:
		marginRate = decimal.NewFromFloat(0.05)
	case domain.RiskLevelMedium:
		marginRate = decimal.NewFromFloat(0.1)
	case domain.RiskLevelHigh:
		marginRate = decimal.NewFromFloat(0.2)
	case domain.RiskLevelCritical:
		marginRate = decimal.NewFromFloat(0.5)
	}

	if symbol == "BTC/USD" || symbol == "ETH/USD" {
		marginRate = marginRate.Mul(decimal.NewFromFloat(1.2))
	}

	return value.Mul(marginRate)
}

// 辅助函数：将 float64 转换为字符串
func floatToString(value float64) string {
	return fmt.Sprintf("%.2f", value)
}
