package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
)

// RiskQueryService 处理所有风险管理相关的查询操作（Queries）。
type RiskQueryService struct {
	repo       domain.RiskRepository
	readRepo   domain.RiskReadRepository
	searchRepo domain.RiskSearchRepository
}

// NewRiskQueryService 构造函数。
func NewRiskQueryService(repo domain.RiskRepository, readRepo domain.RiskReadRepository, searchRepo domain.RiskSearchRepository) *RiskQueryService {
	return &RiskQueryService{
		repo:       repo,
		readRepo:   readRepo,
		searchRepo: searchRepo,
	}
}

// GetRiskMetrics 获取风险指标（优先缓存）
func (s *RiskQueryService) GetRiskMetrics(ctx context.Context, userID string) (*RiskMetricsDTO, error) {
	if s.readRepo != nil {
		if cached, err := s.readRepo.GetMetrics(ctx, userID); err == nil && cached != nil {
			return toRiskMetricsDTO(cached), nil
		}
	}

	metrics, err := s.repo.GetMetrics(ctx, userID)
	if err != nil {
		return nil, err
	}
	if metrics != nil && s.readRepo != nil {
		_ = s.readRepo.SaveMetrics(ctx, userID, metrics)
	}
	return toRiskMetricsDTO(metrics), nil
}

// CheckRiskLimit 检查风险限额（优先缓存）
func (s *RiskQueryService) CheckRiskLimit(ctx context.Context, userID, limitType string) (*RiskLimitDTO, error) {
	if s.readRepo != nil {
		if cached, err := s.readRepo.GetLimit(ctx, userID, limitType); err == nil && cached != nil {
			return toRiskLimitDTO(cached), nil
		}
	}

	limit, err := s.repo.GetLimitByUserIDAndType(ctx, userID, limitType)
	if err != nil {
		return nil, err
	}
	if limit != nil && s.readRepo != nil {
		_ = s.readRepo.SaveLimit(ctx, userID, limit)
	}
	return toRiskLimitDTO(limit), nil
}

// GetRiskAlerts 获取风险告警（告警不设缓存）
func (s *RiskQueryService) GetRiskAlerts(ctx context.Context, userID string, limit int) ([]*RiskAlertDTO, error) {
	alerts, err := s.repo.GetAlertsByUser(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	return toRiskAlertDTOs(alerts), nil
}

// GetCircuitBreaker 获取熔断状态（优先缓存）
func (s *RiskQueryService) GetCircuitBreaker(ctx context.Context, userID string) (*CircuitBreakerDTO, error) {
	if s.readRepo != nil {
		if cached, err := s.readRepo.GetCircuitBreaker(ctx, userID); err == nil && cached != nil {
			return toCircuitBreakerDTO(cached), nil
		}
	}

	cb, err := s.repo.GetCircuitBreakerByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if cb != nil && s.readRepo != nil {
		_ = s.readRepo.SaveCircuitBreaker(ctx, userID, cb)
	}
	return toCircuitBreakerDTO(cb), nil
}

// SearchRiskAssessments 搜索风险评估
func (s *RiskQueryService) SearchRiskAssessments(ctx context.Context, userID, symbol string, level domain.RiskLevel, limit, offset int) ([]*RiskAssessmentDTO, int64, error) {
	if s.searchRepo == nil {
		return nil, 0, nil
	}
	assessments, total, err := s.searchRepo.SearchAssessments(ctx, userID, symbol, level, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*RiskAssessmentDTO, 0, len(assessments))
	for _, a := range assessments {
		result = append(result, toRiskAssessmentDTO(a))
	}
	return result, total, nil
}

// SearchRiskAlerts 搜索风险告警
func (s *RiskQueryService) SearchRiskAlerts(ctx context.Context, userID, severity, alertType string, limit, offset int) ([]*RiskAlertDTO, int64, error) {
	if s.searchRepo == nil {
		return nil, 0, nil
	}
	alerts, total, err := s.searchRepo.SearchAlerts(ctx, userID, severity, alertType, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return toRiskAlertDTOs(alerts), total, nil
}

// CalculatePortfolioRisk 组合风险计算
func (s *RiskQueryService) CalculatePortfolioRisk(ctx context.Context, req *CalculatePortfolioRiskRequest) (*CalculatePortfolioRiskResponse, error) {
	if req == nil || len(req.Assets) == 0 {
		return nil, errors.New("assets required")
	}

	assets := make([]domain.PortfolioAsset, 0, len(req.Assets))
	for _, a := range req.Assets {
		pos, err := decimal.NewFromString(a.Position)
		if err != nil {
			return nil, fmt.Errorf("invalid position for %s", a.Symbol)
		}
		price, err := decimal.NewFromString(a.CurrentPrice)
		if err != nil {
			return nil, fmt.Errorf("invalid current_price for %s", a.Symbol)
		}
		assets = append(assets, domain.PortfolioAsset{
			Symbol:         a.Symbol,
			Position:       pos,
			CurrentPrice:   price,
			Volatility:     a.Volatility,
			ExpectedReturn: a.ExpectedReturn,
		})
	}

	corr := req.CorrelationData
	if len(corr) == 0 {
		corr = make([][]float64, len(assets))
		for i := range assets {
			corr[i] = make([]float64, len(assets))
			corr[i][i] = 1
		}
	}

	simulations := req.Simulations
	if simulations <= 0 {
		simulations = 1000
	}
	confidence := req.ConfidenceLevel
	if confidence <= 0 {
		confidence = 0.95
	}
	horizon := req.TimeHorizon
	if horizon <= 0 {
		horizon = 1.0 / 252.0
	}

	result, err := domain.CalculatePortfolioRisk(domain.PortfolioRiskInput{
		Assets:            assets,
		CorrelationMatrix: corr,
		TimeHorizon:       horizon,
		Simulations:       simulations,
		ConfidenceLevel:   confidence,
	})
	if err != nil {
		return nil, err
	}

	stress := domain.RunStressTests(assets)
	stressDTO := make(map[string]string, len(stress))
	for k, v := range stress {
		stressDTO[k] = v.String()
	}

	greeks := domain.EstimatePortfolioGreeks(assets)
	greeksDTO := make(map[string]PortfolioGreeksDTO, len(greeks))
	for k, v := range greeks {
		greeksDTO[k] = PortfolioGreeksDTO{
			Delta: v.Delta.String(),
			Gamma: v.Gamma.String(),
			Vega:  v.Vega.String(),
			Theta: v.Theta.String(),
		}
	}

	componentVar := make(map[string]string, len(result.ComponentVaR))
	for k, v := range result.ComponentVaR {
		componentVar[k] = v.String()
	}

	return &CalculatePortfolioRiskResponse{
		TotalValue:      result.TotalValue.String(),
		VaR:             result.VaR.String(),
		ES:              result.ES.String(),
		ComponentVaR:    componentVar,
		Diversification: result.Diversification.String(),
		StressTests:     stressDTO,
		Greeks:          greeksDTO,
	}, nil
}

// CalculateMonteCarloRisk 单资产 Monte Carlo 风险计算
func (s *RiskQueryService) CalculateMonteCarloRisk(ctx context.Context, req *MonteCarloRiskRequest) (*MonteCarloRiskResponse, error) {
	if req == nil {
		return nil, errors.New("request required")
	}
	iterations := req.Iterations
	if iterations <= 0 {
		iterations = 10000
	}
	steps := req.Steps
	if steps <= 0 {
		steps = 252
	}

	result := domain.CalculateVaR(domain.MonteCarloInput{
		S:          req.S,
		Mu:         req.Mu,
		Sigma:      req.Sigma,
		T:          req.T,
		Iterations: iterations,
		Steps:      steps,
	})

	return &MonteCarloRiskResponse{
		VaR95: result.VaR95.String(),
		VaR99: result.VaR99.String(),
		ES95:  result.ES95.String(),
		ES99:  result.ES99.String(),
	}, nil
}
