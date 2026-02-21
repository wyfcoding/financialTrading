// Package handler 市场监察服务 gRPC Handler
// 生成摘要：
//  1. 实现 MarketSurveillanceServiceServer 全部 11 个 RPC
//  2. 负责 proto ↔ domain 类型转换
//  3. 委托 CommandService / QueryService 处理业务
package handler

import (
	"context"
	"log/slog"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/wyfcoding/financialtrading/go-api/marketsurveillance/v1"
	"github.com/wyfcoding/financialtrading/internal/marketsurveillance/application"
	"github.com/wyfcoding/financialtrading/internal/marketsurveillance/domain"
)

// MarketSurveillanceHandler gRPC 服务实现
type MarketSurveillanceHandler struct {
	pb.UnimplementedMarketSurveillanceServiceServer
	cmdSvc   *application.CommandService
	querySvc *application.QueryService
	logger   *slog.Logger
}

// NewMarketSurveillanceHandler 创建 Handler
func NewMarketSurveillanceHandler(
	cmdSvc *application.CommandService,
	querySvc *application.QueryService,
	logger *slog.Logger,
) *MarketSurveillanceHandler {
	return &MarketSurveillanceHandler{
		cmdSvc:   cmdSvc,
		querySvc: querySvc,
		logger:   logger,
	}
}

// Register 注册到 gRPC 服务器
func (h *MarketSurveillanceHandler) Register(server *grpc.Server) {
	pb.RegisterMarketSurveillanceServiceServer(server, h)
}

// SubmitOrderEvent 提交订单事件检测
func (h *MarketSurveillanceHandler) SubmitOrderEvent(
	ctx context.Context,
	req *pb.SubmitOrderEventRequest,
) (*pb.SubmitOrderEventResponse, error) {
	if req.Event == nil {
		return nil, status.Error(codes.InvalidArgument, "event is required")
	}

	price, _ := decimal.NewFromString(req.Event.Price)
	qty, _ := decimal.NewFromString(req.Event.Quantity)

	ts := req.Event.Timestamp.AsTime()

	cmd := application.SubmitOrderEventCmd{
		OrderID:   req.Event.OrderId,
		UserID:    req.Event.UserId,
		Symbol:    req.Event.Symbol,
		EventType: domain.OrderEventType(req.Event.EventType),
		Side:      req.Event.Side,
		Price:     price,
		Quantity:  qty,
		Timestamp: ts,
		Venue:     req.Event.Venue,
		AccountID: req.Event.AccountId,
	}

	result, err := h.cmdSvc.SubmitOrderEvent(ctx, cmd)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "submit order event failed: %v", err)
	}

	resp := &pb.SubmitOrderEventResponse{
		AlertTriggered: result.AlertTriggered,
	}
	if result.Alert != nil {
		resp.Alert = alertToProto(result.Alert)
	}
	return resp, nil
}

// CreateRule 创建监察规则
func (h *MarketSurveillanceHandler) CreateRule(
	ctx context.Context,
	req *pb.CreateRuleRequest,
) (*pb.CreateRuleResponse, error) {
	threshold, _ := decimal.NewFromString(req.Threshold)
	cancelRatio, _ := decimal.NewFromString(req.MinCancelRatio)
	washRatio, _ := decimal.NewFromString(req.MaxWashVolumeRatio)
	priceDev, _ := decimal.NewFromString(req.PriceDeviationThreshold)

	cmd := application.CreateRuleCmd{
		Name:                    req.Name,
		Type:                    domain.ManipulationType(req.Type),
		WindowSeconds:           req.WindowSeconds,
		Threshold:               threshold,
		MinCancelRatio:          cancelRatio,
		MaxWashVolumeRatio:      washRatio,
		PriceDeviationThreshold: priceDev,
		Description:             req.Description,
	}

	rule, err := h.cmdSvc.CreateRule(ctx, cmd)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create rule failed: %v", err)
	}
	return &pb.CreateRuleResponse{Rule: ruleToProto(rule)}, nil
}

// UpdateRule 更新规则
func (h *MarketSurveillanceHandler) UpdateRule(
	ctx context.Context,
	req *pb.UpdateRuleRequest,
) (*pb.UpdateRuleResponse, error) {
	threshold, _ := decimal.NewFromString(req.Threshold)
	cancelRatio, _ := decimal.NewFromString(req.MinCancelRatio)
	washRatio, _ := decimal.NewFromString(req.MaxWashVolumeRatio)
	priceDev, _ := decimal.NewFromString(req.PriceDeviationThreshold)

	cmd := application.UpdateRuleCmd{
		RuleID:                  req.RuleId,
		Name:                    req.Name,
		Enabled:                 req.Enabled,
		WindowSeconds:           req.WindowSeconds,
		Threshold:               threshold,
		MinCancelRatio:          cancelRatio,
		MaxWashVolumeRatio:      washRatio,
		PriceDeviationThreshold: priceDev,
		Description:             req.Description,
	}

	rule, err := h.cmdSvc.UpdateRule(ctx, cmd)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update rule failed: %v", err)
	}
	return &pb.UpdateRuleResponse{Rule: ruleToProto(rule)}, nil
}

// GetRule 获取规则
func (h *MarketSurveillanceHandler) GetRule(
	ctx context.Context,
	req *pb.GetRuleRequest,
) (*pb.GetRuleResponse, error) {
	rule, err := h.querySvc.GetRule(ctx, req.RuleId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "rule not found: %v", err)
	}
	return &pb.GetRuleResponse{Rule: ruleToProto(rule)}, nil
}

// ListRules 列出规则
func (h *MarketSurveillanceHandler) ListRules(
	ctx context.Context,
	req *pb.ListRulesRequest,
) (*pb.ListRulesResponse, error) {
	query := application.ListRulesQuery{
		EnabledOnly: req.EnabledOnly,
	}
	if req.TypeFilter != pb.ManipulationType_MANIPULATION_TYPE_UNSPECIFIED {
		t := domain.ManipulationType(req.TypeFilter)
		query.TypeFilter = &t
	}
	rules, err := h.querySvc.ListRules(ctx, query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list rules failed: %v", err)
	}
	var pbRules []*pb.SurveillanceRule
	for _, r := range rules {
		pbRules = append(pbRules, ruleToProto(r))
	}
	return &pb.ListRulesResponse{Rules: pbRules}, nil
}

// GetAlert 获取告警
func (h *MarketSurveillanceHandler) GetAlert(
	ctx context.Context,
	req *pb.GetAlertRequest,
) (*pb.GetAlertResponse, error) {
	alert, err := h.querySvc.GetAlert(ctx, req.AlertId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "alert not found: %v", err)
	}
	return &pb.GetAlertResponse{Alert: alertToProto(alert)}, nil
}

// ListAlerts 列出告警
func (h *MarketSurveillanceHandler) ListAlerts(
	ctx context.Context,
	req *pb.ListAlertsRequest,
) (*pb.ListAlertsResponse, error) {
	query := application.ListAlertsQuery{
		UserID:   req.UserId,
		Symbol:   req.Symbol,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	}
	if req.StatusFilter != pb.AlertStatus_ALERT_STATUS_UNSPECIFIED {
		s := domain.AlertStatus(req.StatusFilter)
		query.Status = &s
	}
	if req.SeverityFilter != pb.AlertSeverity_ALERT_SEVERITY_UNSPECIFIED {
		sev := domain.AlertSeverity(req.SeverityFilter)
		query.Severity = &sev
	}

	alerts, total, err := h.querySvc.ListAlerts(ctx, query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list alerts failed: %v", err)
	}
	var pbAlerts []*pb.SurveillanceAlert
	for _, a := range alerts {
		pbAlerts = append(pbAlerts, alertToProto(a))
	}
	return &pb.ListAlertsResponse{Alerts: pbAlerts, Total: total}, nil
}

// ReviewAlert 审核告警
func (h *MarketSurveillanceHandler) ReviewAlert(
	ctx context.Context,
	req *pb.ReviewAlertRequest,
) (*pb.ReviewAlertResponse, error) {
	cmd := application.ReviewAlertCmd{
		AlertID:    req.AlertId,
		ReviewerID: req.ReviewerId,
		Confirmed:  req.Confirmed,
		Comment:    req.Comment,
	}
	if err := h.cmdSvc.ReviewAlert(ctx, cmd); err != nil {
		return nil, status.Errorf(codes.Internal, "review alert failed: %v", err)
	}
	return &pb.ReviewAlertResponse{Success: true}, nil
}

// GetUserSurveillanceScore 获取用户评分
func (h *MarketSurveillanceHandler) GetUserSurveillanceScore(
	ctx context.Context,
	req *pb.GetUserScoreRequest,
) (*pb.GetUserScoreResponse, error) {
	score, err := h.querySvc.GetUserScore(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "score not found: %v", err)
	}
	return &pb.GetUserScoreResponse{Score: scoreToProto(score)}, nil
}

// AnalyzeTradingPattern 分析交易模式
func (h *MarketSurveillanceHandler) AnalyzeTradingPattern(
	ctx context.Context,
	req *pb.AnalyzeTradingPatternRequest,
) (*pb.AnalyzeTradingPatternResponse, error) {
	query := application.AnalyzeTradingPatternQuery{
		UserID: req.UserId,
		Symbol: req.Symbol,
	}
	if req.StartTime != nil {
		query.StartTime = req.StartTime.AsTime()
	}
	if req.EndTime != nil {
		query.EndTime = req.EndTime.AsTime()
	}

	pattern, err := h.querySvc.AnalyzeTradingPattern(ctx, query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "analyze pattern failed: %v", err)
	}
	return &pb.AnalyzeTradingPatternResponse{Pattern: patternToProto(pattern)}, nil
}

// ===== Proto 转换辅助函数 =====

func alertToProto(a *domain.SurveillanceAlert) *pb.SurveillanceAlert {
	pbAlert := &pb.SurveillanceAlert{
		AlertId:         a.AlertID,
		Type:            pb.ManipulationType(a.Type),
		Severity:        pb.AlertSeverity(a.Severity),
		Status:          pb.AlertStatus(a.Status),
		UserId:          a.UserID,
		Symbol:          a.Symbol,
		Description:     a.Description,
		ConfidenceScore: a.ConfidenceScore,
		RuleId:          a.RuleID,
		ReviewerId:      a.ReviewerID,
		ReviewComment:   a.ReviewComment,
		DetectedAt:      timestamppb.New(a.DetectedAt),
	}
	if a.RelatedOrderIDs != "" {
		pbAlert.RelatedOrderIds = splitNonEmpty(a.RelatedOrderIDs, ",")
	}
	if a.ReviewedAt != nil {
		pbAlert.ReviewedAt = timestamppb.New(*a.ReviewedAt)
	}
	return pbAlert
}

func ruleToProto(r *domain.SurveillanceRule) *pb.SurveillanceRule {
	return &pb.SurveillanceRule{
		RuleId:                  r.RuleID,
		Name:                    r.Name,
		Type:                    pb.ManipulationType(r.Type),
		Enabled:                 r.Enabled,
		WindowSeconds:           r.WindowSeconds,
		Threshold:               r.Threshold.String(),
		MinCancelRatio:          r.MinCancelRatio.String(),
		MaxWashVolumeRatio:      r.MaxWashVolumeRatio.String(),
		PriceDeviationThreshold: r.PriceDeviationThreshold.String(),
		Description:             r.Description,
		CreatedAt:               timestamppb.New(r.CreatedAt),
		UpdatedAt:               timestamppb.New(r.UpdatedAt),
	}
}

func scoreToProto(s *domain.UserSurveillanceScore) *pb.UserSurveillanceScore {
	return &pb.UserSurveillanceScore{
		UserId:           s.UserID,
		OverallScore:     s.OverallScore,
		SpoofingScore:    s.SpoofingScore,
		WashTradingScore: s.WashTradingScore,
		LayeringScore:    s.LayeringScore,
		TotalAlerts:      s.TotalAlerts,
		ConfirmedAlerts:  s.ConfirmedAlerts,
		LastUpdated:      timestamppb.New(s.UpdatedAt),
	}
}

func patternToProto(p *domain.TradingPattern) *pb.TradingPattern {
	return &pb.TradingPattern{
		UserId:             p.UserID,
		Symbol:             p.Symbol,
		CancelRatio:        p.CancelRatio,
		AvgOrderLifetimeMs: p.AvgOrderLifetimeMs,
		SelfTradeRatio:     p.SelfTradeRatio,
		TotalOrders:        p.TotalOrders,
		TotalCancels:       p.TotalCancels,
		TotalFills:         p.TotalFills,
		RiskLevel:          p.RiskLevel,
	}
}

// splitNonEmpty 分割非空字符串
func splitNonEmpty(s, sep string) []string {
	if s == "" {
		return nil
	}
	parts := make([]string, 0)
	for _, p := range splitString(s, sep) {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// splitString 简单分割
func splitString(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if string(s[i]) == sep {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}
