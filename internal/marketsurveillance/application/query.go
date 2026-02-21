// Package application 市场监察服务应用层 - 查询服务
// 生成摘要：
//  1. 提供告警查询、规则查询、用户评分查询
//  2. 交易模式分析查询
//  3. 纯读操作，不产生副作用
package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/wyfcoding/financialtrading/internal/marketsurveillance/domain"
)

// QueryService 市场监察查询服务
type QueryService struct {
	alertRepo     domain.AlertRepository
	ruleRepo      domain.RuleRepository
	userScoreRepo domain.UserScoreRepository
	eventRepo     domain.OrderEventRepository
	engine        *domain.DetectionEngine
	logger        *slog.Logger
}

// NewQueryService 创建查询服务
func NewQueryService(
	alertRepo domain.AlertRepository,
	ruleRepo domain.RuleRepository,
	userScoreRepo domain.UserScoreRepository,
	eventRepo domain.OrderEventRepository,
	engine *domain.DetectionEngine,
	logger *slog.Logger,
) *QueryService {
	return &QueryService{
		alertRepo:     alertRepo,
		ruleRepo:      ruleRepo,
		userScoreRepo: userScoreRepo,
		eventRepo:     eventRepo,
		engine:        engine,
		logger:        logger,
	}
}

// GetAlert 获取告警详情
func (q *QueryService) GetAlert(ctx context.Context, alertID string) (*domain.SurveillanceAlert, error) {
	return q.alertRepo.GetByID(ctx, alertID)
}

// ListAlertsQuery 告警列表查询参数
type ListAlertsQuery struct {
	Status   *domain.AlertStatus
	Severity *domain.AlertSeverity
	UserID   string
	Symbol   string
	Page     int
	PageSize int
}

// ListAlerts 列表查询告警
func (q *QueryService) ListAlerts(
	ctx context.Context,
	query ListAlertsQuery,
) ([]*domain.SurveillanceAlert, int64, error) {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 20
	}
	filters := domain.AlertFilters{
		Status:   query.Status,
		Severity: query.Severity,
		UserID:   query.UserID,
		Symbol:   query.Symbol,
	}
	return q.alertRepo.ListByFilters(ctx, filters, query.Page, query.PageSize)
}

// GetRule 获取规则详情
func (q *QueryService) GetRule(ctx context.Context, ruleID string) (*domain.SurveillanceRule, error) {
	return q.ruleRepo.GetByID(ctx, ruleID)
}

// ListRulesQuery 规则列表查询参数
type ListRulesQuery struct {
	TypeFilter  *domain.ManipulationType
	EnabledOnly bool
}

// ListRules 列表查询规则
func (q *QueryService) ListRules(
	ctx context.Context,
	query ListRulesQuery,
) ([]*domain.SurveillanceRule, error) {
	if query.EnabledOnly {
		return q.ruleRepo.ListEnabled(ctx)
	}
	if query.TypeFilter != nil {
		return q.ruleRepo.ListByType(ctx, *query.TypeFilter)
	}
	return q.ruleRepo.ListAll(ctx)
}

// GetUserScore 获取用户评分
func (q *QueryService) GetUserScore(
	ctx context.Context,
	userID string,
) (*domain.UserSurveillanceScore, error) {
	return q.userScoreRepo.GetByUserID(ctx, userID)
}

// AnalyzeTradingPatternQuery 交易模式分析参数
type AnalyzeTradingPatternQuery struct {
	UserID    string
	Symbol    string
	StartTime time.Time
	EndTime   time.Time
}

// AnalyzeTradingPattern 分析用户交易模式
func (q *QueryService) AnalyzeTradingPattern(
	ctx context.Context,
	query AnalyzeTradingPatternQuery,
) (*domain.TradingPattern, error) {
	records, err := q.eventRepo.GetByUserAndSymbol(
		ctx, query.UserID, query.Symbol, query.StartTime, query.EndTime,
	)
	if err != nil {
		q.logger.ErrorContext(ctx, "failed to load events for pattern analysis",
			"user_id", query.UserID, "symbol", query.Symbol, "error", err)
		return nil, err
	}
	events := recordsToEvents(records)
	pattern := q.engine.AnalyzeTradingPattern(events)
	return pattern, nil
}
