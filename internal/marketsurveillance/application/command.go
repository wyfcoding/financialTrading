// Package application 市场监察服务应用层 - 命令服务
// 生成摘要：
//  1. 处理订单事件流入，调用检测引擎，自动生成告警
//  2. 管理规则 CRUD、告警审核
//  3. 维护用户监察评分
//  4. 通过 Kafka 事件发布检测结果
package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/wyfcoding/financialtrading/internal/marketsurveillance/domain"
	"github.com/wyfcoding/pkg/messagequeue"
)

// CommandService 市场监察命令服务
type CommandService struct {
	alertRepo      domain.AlertRepository
	ruleRepo       domain.RuleRepository
	userScoreRepo  domain.UserScoreRepository
	eventRepo      domain.OrderEventRepository
	engine         *domain.DetectionEngine
	eventPublisher messagequeue.EventPublisher
	logger         *slog.Logger
}

// NewCommandService 创建命令服务
func NewCommandService(
	alertRepo domain.AlertRepository,
	ruleRepo domain.RuleRepository,
	userScoreRepo domain.UserScoreRepository,
	eventRepo domain.OrderEventRepository,
	engine *domain.DetectionEngine,
	publisher messagequeue.EventPublisher,
	logger *slog.Logger,
) *CommandService {
	return &CommandService{
		alertRepo:      alertRepo,
		ruleRepo:       ruleRepo,
		userScoreRepo:  userScoreRepo,
		eventRepo:      eventRepo,
		engine:         engine,
		eventPublisher: publisher,
		logger:         logger,
	}
}

// SubmitOrderEventCmd 提交订单事件命令
type SubmitOrderEventCmd struct {
	OrderID   string
	UserID    string
	Symbol    string
	EventType domain.OrderEventType
	Side      string
	Price     decimal.Decimal
	Quantity  decimal.Decimal
	Timestamp time.Time
	Venue     string
	AccountID string
}

// SubmitOrderEventResult 提交结果
type SubmitOrderEventResult struct {
	AlertTriggered bool
	Alert          *domain.SurveillanceAlert
}

// SubmitOrderEvent 处理订单事件流入
// 流程：存储事件 → 加载启用规则 → 检测引擎分析 → 生成告警 → 更新用户评分
func (s *CommandService) SubmitOrderEvent(
	ctx context.Context,
	cmd SubmitOrderEventCmd,
) (*SubmitOrderEventResult, error) {
	start := time.Now()

	// 1. 存储事件记录
	record := &domain.OrderEventRecord{
		OrderID:   cmd.OrderID,
		UserID:    cmd.UserID,
		Symbol:    cmd.Symbol,
		EventType: cmd.EventType,
		Side:      cmd.Side,
		Price:     cmd.Price.String(),
		Quantity:  cmd.Quantity.String(),
		Venue:     cmd.Venue,
		AccountID: cmd.AccountID,
		EventTime: cmd.Timestamp,
	}
	if err := s.eventRepo.SaveEvent(ctx, record); err != nil {
		s.logger.ErrorContext(ctx, "failed to save order event",
			"order_id", cmd.OrderID, "error", err, "duration", time.Since(start))
		return nil, err
	}

	// 2. 加载启用的规则
	rules, err := s.ruleRepo.ListEnabled(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to load rules", "error", err)
		return nil, err
	}

	// 3. 获取历史事件（检测窗口取最大规则窗口）
	maxWindow := int64(300)
	for _, r := range rules {
		if r.WindowSeconds > maxWindow {
			maxWindow = r.WindowSeconds
		}
	}
	windowStart := cmd.Timestamp.Add(-time.Duration(maxWindow) * time.Second)
	historyRecords, err := s.eventRepo.GetByUserAndSymbol(
		ctx, cmd.UserID, cmd.Symbol, windowStart, cmd.Timestamp,
	)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to load history, using empty",
			"error", err)
		historyRecords = nil
	}
	history := recordsToEvents(historyRecords)

	// 4. 构建当前事件
	currentEvent := domain.OrderEvent{
		OrderID:   cmd.OrderID,
		UserID:    cmd.UserID,
		Symbol:    cmd.Symbol,
		EventType: cmd.EventType,
		Side:      cmd.Side,
		Price:     cmd.Price,
		Quantity:  cmd.Quantity,
		Timestamp: cmd.Timestamp,
		Venue:     cmd.Venue,
		AccountID: cmd.AccountID,
	}

	// 5. 执行检测
	results := s.engine.Detect(currentEvent, history, rules)

	// 6. 处理检测结果
	var highestAlert *domain.SurveillanceAlert
	for _, result := range results {
		orderIDs := collectRelatedOrders(history, cmd.UserID, cmd.Symbol)
		alert := domain.NewAlert(
			fmt.Sprintf("SUR-%d-%s", time.Now().UnixNano(), cmd.UserID[:minLen(8, len(cmd.UserID))]),
			result.Type,
			result.Severity,
			cmd.UserID,
			cmd.Symbol,
			result.Reason,
			"", // ruleID 可在检测结果中携带
			result.Confidence,
			strings.Join(orderIDs, ","),
		)
		if err := s.alertRepo.Save(ctx, alert); err != nil {
			s.logger.ErrorContext(ctx, "failed to save alert",
				"alert_id", alert.AlertID, "error", err)
			continue
		}

		// 发布告警创建事件
		alertEvent := &domain.AlertCreatedEvent{
			AlertID:         alert.AlertID,
			UserID:          alert.UserID,
			Symbol:          alert.Symbol,
			Type:            alert.Type,
			Severity:        alert.Severity,
			ConfidenceScore: alert.ConfidenceScore,
			Timestamp:       time.Now(),
		}
		if err := s.eventPublisher.Publish(ctx, alertEvent.EventName(), alert.AlertID, alertEvent); err != nil {
			s.logger.ErrorContext(ctx, "failed to publish alert event", "error", err)
		}

		// 更新用户评分
		s.updateUserScore(ctx, cmd.UserID, result)

		if highestAlert == nil || alert.ConfidenceScore > highestAlert.ConfidenceScore {
			highestAlert = alert
		}

		s.logger.InfoContext(ctx, "surveillance alert created",
			"alert_id", alert.AlertID,
			"type", result.Type,
			"severity", result.Severity,
			"confidence", result.Confidence,
			"user_id", cmd.UserID,
			"symbol", cmd.Symbol,
			"duration", time.Since(start))
	}

	return &SubmitOrderEventResult{
		AlertTriggered: highestAlert != nil,
		Alert:          highestAlert,
	}, nil
}

// CreateRuleCmd 创建规则命令
type CreateRuleCmd struct {
	Name                    string
	Type                    domain.ManipulationType
	WindowSeconds           int64
	Threshold               decimal.Decimal
	MinCancelRatio          decimal.Decimal
	MaxWashVolumeRatio      decimal.Decimal
	PriceDeviationThreshold decimal.Decimal
	Description             string
}

// CreateRule 创建监察规则
func (s *CommandService) CreateRule(ctx context.Context, cmd CreateRuleCmd) (*domain.SurveillanceRule, error) {
	rule := &domain.SurveillanceRule{
		RuleID:                  fmt.Sprintf("RULE-%d", time.Now().UnixNano()),
		Name:                    cmd.Name,
		Type:                    cmd.Type,
		Enabled:                 true,
		WindowSeconds:           cmd.WindowSeconds,
		Threshold:               cmd.Threshold,
		MinCancelRatio:          cmd.MinCancelRatio,
		MaxWashVolumeRatio:      cmd.MaxWashVolumeRatio,
		PriceDeviationThreshold: cmd.PriceDeviationThreshold,
		Description:             cmd.Description,
	}
	if err := s.ruleRepo.Save(ctx, rule); err != nil {
		return nil, err
	}
	s.logger.InfoContext(ctx, "surveillance rule created",
		"rule_id", rule.RuleID, "name", rule.Name, "type", rule.Type)
	return rule, nil
}

// UpdateRuleCmd 更新规则命令
type UpdateRuleCmd struct {
	RuleID                  string
	Name                    string
	Enabled                 bool
	WindowSeconds           int64
	Threshold               decimal.Decimal
	MinCancelRatio          decimal.Decimal
	MaxWashVolumeRatio      decimal.Decimal
	PriceDeviationThreshold decimal.Decimal
	Description             string
}

// UpdateRule 更新规则
func (s *CommandService) UpdateRule(ctx context.Context, cmd UpdateRuleCmd) (*domain.SurveillanceRule, error) {
	rule, err := s.ruleRepo.GetByID(ctx, cmd.RuleID)
	if err != nil {
		return nil, fmt.Errorf("rule %s not found: %w", cmd.RuleID, err)
	}
	rule.Name = cmd.Name
	rule.Enabled = cmd.Enabled
	rule.WindowSeconds = cmd.WindowSeconds
	rule.Threshold = cmd.Threshold
	rule.MinCancelRatio = cmd.MinCancelRatio
	rule.MaxWashVolumeRatio = cmd.MaxWashVolumeRatio
	rule.PriceDeviationThreshold = cmd.PriceDeviationThreshold
	rule.Description = cmd.Description
	if err := s.ruleRepo.Save(ctx, rule); err != nil {
		return nil, err
	}
	return rule, nil
}

// ReviewAlertCmd 审核告警命令
type ReviewAlertCmd struct {
	AlertID    string
	ReviewerID string
	Confirmed  bool
	Comment    string
}

// ReviewAlert 审核告警
// 流程：开始调查 → 确认/误报关闭 → 事件发布 → 更新用户评分
func (s *CommandService) ReviewAlert(ctx context.Context, cmd ReviewAlertCmd) error {
	alert, err := s.alertRepo.GetByID(ctx, cmd.AlertID)
	if err != nil {
		return fmt.Errorf("alert %s not found: %w", cmd.AlertID, err)
	}

	// 如果是 Open 状态先进入调查
	if alert.Status == domain.AlertOpen {
		if err := alert.StartInvestigation(cmd.ReviewerID); err != nil {
			return err
		}
	}

	// 关闭告警
	if cmd.Confirmed {
		if err := alert.CloseConfirmed(cmd.ReviewerID, cmd.Comment); err != nil {
			return err
		}
	} else {
		if err := alert.CloseFalsePositive(cmd.ReviewerID, cmd.Comment); err != nil {
			return err
		}
	}

	if err := s.alertRepo.Save(ctx, alert); err != nil {
		return err
	}

	// 发布事件
	for _, ev := range alert.GetDomainEvents() {
		if err := s.eventPublisher.Publish(ctx, ev.EventName(), cmd.AlertID, ev); err != nil {
			s.logger.ErrorContext(ctx, "failed to publish event",
				"event", ev.EventName(), "error", err)
		}
	}
	alert.ClearDomainEvents()

	s.logger.InfoContext(ctx, "surveillance alert reviewed",
		"alert_id", cmd.AlertID,
		"confirmed", cmd.Confirmed,
		"reviewer_id", cmd.ReviewerID)
	return nil
}

// updateUserScore 更新用户监察评分
func (s *CommandService) updateUserScore(
	ctx context.Context,
	userID string,
	result domain.DetectionResult,
) {
	score, err := s.userScoreRepo.GetByUserID(ctx, userID)
	if err != nil {
		score = &domain.UserSurveillanceScore{UserID: userID}
	}

	score.TotalAlerts++

	switch result.Type {
	case domain.ManipulationSpoofing:
		score.SpoofingScore = ewma(score.SpoofingScore, result.Confidence, 0.3)
	case domain.ManipulationWashTrading:
		score.WashTradingScore = ewma(score.WashTradingScore, result.Confidence, 0.3)
	case domain.ManipulationLayering:
		score.LayeringScore = ewma(score.LayeringScore, result.Confidence, 0.3)
	}

	// 综合评分 = 各维度加权平均
	score.OverallScore = score.SpoofingScore*0.35 +
		score.WashTradingScore*0.30 +
		score.LayeringScore*0.20 +
		float64(score.ConfirmedAlerts)/float64(max(score.TotalAlerts, 1))*0.15

	if score.OverallScore > 1.0 {
		score.OverallScore = 1.0
	}

	if err := s.userScoreRepo.Save(ctx, score); err != nil {
		s.logger.ErrorContext(ctx, "failed to update user score",
			"user_id", userID, "error", err)
	}
}

// ewma 指数加权移动平均
func ewma(oldValue, newValue, alpha float64) float64 {
	return alpha*newValue + (1-alpha)*oldValue
}

// max 取最大值
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// minLen 取最小长度
func minLen(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// collectRelatedOrders 收集关联订单ID
func collectRelatedOrders(events []domain.OrderEvent, userID, symbol string) []string {
	seen := make(map[string]bool)
	var orders []string
	for _, ev := range events {
		if ev.UserID == userID && ev.Symbol == symbol && !seen[ev.OrderID] {
			seen[ev.OrderID] = true
			orders = append(orders, ev.OrderID)
		}
	}
	return orders
}

// recordsToEvents 将持久化记录转换为领域事件
func recordsToEvents(records []*domain.OrderEventRecord) []domain.OrderEvent {
	events := make([]domain.OrderEvent, 0, len(records))
	for _, r := range records {
		price, _ := decimal.NewFromString(r.Price)
		qty, _ := decimal.NewFromString(r.Quantity)
		events = append(events, domain.OrderEvent{
			OrderID:   r.OrderID,
			UserID:    r.UserID,
			Symbol:    r.Symbol,
			EventType: r.EventType,
			Side:      r.Side,
			Price:     price,
			Quantity:  qty,
			Timestamp: r.EventTime,
			Venue:     r.Venue,
			AccountID: r.AccountID,
		})
	}
	return events
}

// 确保 json import 被使用（在序列化告警关联订单时需要）
var _ = json.Marshal
