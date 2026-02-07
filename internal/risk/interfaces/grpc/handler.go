package grpc

import (
	"context"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialtrading/go-api/risk/v1"
	"github.com/wyfcoding/financialtrading/internal/risk/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler gRPC 处理器
// 负责处理与风险管理相关的 gRPC 请求
type Handler struct {
	pb.UnimplementedRiskServiceServer
	cmd   *application.RiskCommandService
	query *application.RiskQueryService
}

// NewHandler 创建 gRPC 处理器实例
func NewHandler(cmd *application.RiskCommandService, query *application.RiskQueryService) *Handler {
	return &Handler{cmd: cmd, query: query}
}

// CheckRisk 检查交易风险 (Legacy, mapped to AssessRisk)
func (h *Handler) CheckRisk(ctx context.Context, req *pb.CheckRiskRequest) (*pb.CheckRiskResponse, error) {
	side := req.Side
	if side == "" {
		side = "buy"
	}
	dto, err := h.cmd.AssessRisk(ctx, application.AssessRiskCommand{
		UserID:   req.UserId,
		Symbol:   req.Symbol,
		Side:     side,
		Quantity: req.Quantity,
		Price:    req.Price,
	})
	if err != nil {
		return &pb.CheckRiskResponse{Passed: false, Reason: err.Error()}, nil
	}
	return &pb.CheckRiskResponse{Passed: dto.IsAllowed, Reason: dto.Reason}, nil
}

// SetRiskLimit 设置风险限额 (Legacy)
func (h *Handler) SetRiskLimit(ctx context.Context, req *pb.SetRiskLimitRequest) (*pb.SetRiskLimitResponse, error) {
	_, err := h.cmd.UpdateRiskLimit(ctx, application.UpdateRiskLimitCommand{
		UserID:     req.UserId,
		LimitType:  "ORDER_SIZE",
		LimitValue: req.MaxOrderSize,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.SetRiskLimitResponse{Success: true}, nil
}

// AssessRisk 评估交易风险 (Enhanced)
func (h *Handler) AssessRisk(ctx context.Context, req *pb.AssessRiskRequest) (*pb.AssessRiskResponse, error) {
	start := time.Now()
	slog.Info("gRPC AssessRisk received", "user_id", req.UserId, "symbol", req.Symbol, "side", req.Side)

	qty, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid quantity: %v", err)
	}
	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid price: %v", err)
	}

	dto, err := h.cmd.AssessRisk(ctx, application.AssessRiskCommand{
		UserID:   req.UserId,
		Symbol:   req.Symbol,
		Side:     req.Side,
		Quantity: qty.InexactFloat64(),
		Price:    price.InexactFloat64(),
	})
	if err != nil {
		slog.Error("gRPC AssessRisk failed", "user_id", req.UserId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to assess risk: %v", err)
	}

	return &pb.AssessRiskResponse{
		RiskLevel:         dto.RiskLevel,
		RiskScore:         dto.RiskScore,
		MarginRequirement: dto.MarginRequirement,
		IsAllowed:         dto.IsAllowed,
		Reason:            dto.Reason,
	}, nil
}

// GetRiskMetrics 获取风险指标
func (h *Handler) GetRiskMetrics(ctx context.Context, req *pb.GetRiskMetricsRequest) (*pb.GetRiskMetricsResponse, error) {
	metrics, err := h.query.GetRiskMetrics(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get risk metrics: %v", err)
	}
	if metrics == nil {
		return &pb.GetRiskMetricsResponse{}, nil
	}

	return &pb.GetRiskMetricsResponse{
		Metrics: &pb.RiskMetrics{
			Var_95:      metrics.VaR95,
			Var_99:      metrics.VaR99,
			MaxDrawdown: metrics.MaxDrawdown,
			SharpeRatio: metrics.SharpeRatio,
			Correlation: metrics.Correlation,
		},
	}, nil
}

// CheckRiskLimit 检查风险限额
func (h *Handler) CheckRiskLimit(ctx context.Context, req *pb.CheckRiskLimitRequest) (*pb.CheckRiskLimitResponse, error) {
	limit, err := h.query.CheckRiskLimit(ctx, req.UserId, req.LimitType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check risk limit: %v", err)
	}
	if limit == nil {
		return &pb.CheckRiskLimitResponse{}, nil
	}

	limitValue, _ := decimal.NewFromString(limit.LimitValue)
	currentValue, _ := decimal.NewFromString(limit.CurrentValue)
	remaining := limitValue.Sub(currentValue)

	return &pb.CheckRiskLimitResponse{
		LimitType:    limit.LimitType,
		LimitValue:   limit.LimitValue,
		CurrentValue: limit.CurrentValue,
		Remaining:    remaining.String(),
		IsExceeded:   limit.IsExceeded,
	}, nil
}

// GetRiskAlerts 获取风险告警
func (h *Handler) GetRiskAlerts(ctx context.Context, req *pb.GetRiskAlertsRequest) (*pb.GetRiskAlertsResponse, error) {
	alerts, err := h.query.GetRiskAlerts(ctx, req.UserId, 100)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get risk alerts: %v", err)
	}

	pbAlerts := make([]*pb.RiskAlert, 0, len(alerts))
	for _, alert := range alerts {
		pbAlerts = append(pbAlerts, &pb.RiskAlert{
			AlertId:   alert.ID,
			AlertType: alert.AlertType,
			Severity:  alert.Severity,
			Message:   alert.Message,
			Timestamp: alert.CreatedAt,
		})
	}

	return &pb.GetRiskAlertsResponse{Alerts: pbAlerts}, nil
}

// CalculatePortfolioRisk 组合风险计算
func (h *Handler) CalculatePortfolioRisk(ctx context.Context, req *pb.CalculatePortfolioRiskRequest) (*pb.CalculatePortfolioRiskResponse, error) {
	assets := make([]application.PortfolioAssetDTO, 0, len(req.Assets))
	for _, a := range req.Assets {
		assets = append(assets, application.PortfolioAssetDTO{
			Symbol:         a.Symbol,
			Position:       a.Position,
			CurrentPrice:   a.CurrentPrice,
			Volatility:     a.Volatility,
			ExpectedReturn: a.ExpectedReturn,
		})
	}

	corr := make([][]float64, 0, len(req.CorrelationMatrix))
	for _, row := range req.CorrelationMatrix {
		corr = append(corr, row.Values)
	}

	result, err := h.query.CalculatePortfolioRisk(ctx, &application.CalculatePortfolioRiskRequest{
		Assets:          assets,
		CorrelationData: corr,
		TimeHorizon:     req.TimeHorizon,
		Simulations:     int(req.Simulations),
		ConfidenceLevel: req.ConfidenceLevel,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to calculate portfolio risk: %v", err)
	}
	if result == nil {
		return &pb.CalculatePortfolioRiskResponse{}, nil
	}

	greeks := make(map[string]*pb.PortfolioGreeks, len(result.Greeks))
	for k, v := range result.Greeks {
		greeks[k] = &pb.PortfolioGreeks{
			Delta: v.Delta,
			Gamma: v.Gamma,
			Vega:  v.Vega,
			Theta: v.Theta,
		}
	}

	return &pb.CalculatePortfolioRiskResponse{
		TotalValue:      result.TotalValue,
		VarValue:        result.VaR,
		EsValue:         result.ES,
		ComponentVar:    result.ComponentVaR,
		Diversification: result.Diversification,
		StressTests:     result.StressTests,
		Greeks:          greeks,
	}, nil
}

// CalculateMonteCarloRisk 单资产 Monte Carlo 风险计算
func (h *Handler) CalculateMonteCarloRisk(ctx context.Context, req *pb.MonteCarloRiskRequest) (*pb.MonteCarloRiskResponse, error) {
	result, err := h.query.CalculateMonteCarloRisk(ctx, &application.MonteCarloRiskRequest{
		S:          req.S,
		Mu:         req.Mu,
		Sigma:      req.Sigma,
		T:          req.T,
		Iterations: int(req.Iterations),
		Steps:      int(req.Steps),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to calculate monte carlo risk: %v", err)
	}
	if result == nil {
		return &pb.MonteCarloRiskResponse{}, nil
	}
	return &pb.MonteCarloRiskResponse{
		Var_95: result.VaR95,
		Var_99: result.VaR99,
		Es_95:  result.ES95,
		Es_99:  result.ES99,
	}, nil
}
