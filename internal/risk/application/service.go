package application

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	pb "github.com/wyfcoding/financialtrading/go-api/risk/v1"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
	"github.com/shopspring/decimal"
)

type RiskService struct {
	repo domain.RiskRepository
}

func NewRiskService(repo domain.RiskRepository) *RiskService {
	return &RiskService{repo: repo}
}

func (s *RiskService) CheckRisk(ctx context.Context, req *pb.CheckRiskRequest) (*pb.CheckRiskResponse, error) {
	orderValue := req.Price * req.Quantity

	maxOrderSize := 1000000.0
	if orderValue > maxOrderSize {
		return &pb.CheckRiskResponse{
			Passed: false,
			Reason: fmt.Sprintf("Order value %.2f exceeds maximum allowed %.2f", orderValue, maxOrderSize),
		}, nil
	}

	return &pb.CheckRiskResponse{Passed: true}, nil
}

func (s *RiskService) SetRiskLimit(ctx context.Context, req *pb.SetRiskLimitRequest) (*pb.SetRiskLimitResponse, error) {
	limit := &domain.RiskLimit{
		UserID:     req.UserId,
		LimitType:  "max_order_size",
		LimitValue: req.MaxOrderSize,
		Period:     "daily",
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.repo.SaveRiskLimit(ctx, limit); err != nil {
		return nil, err
	}

	if req.MaxDailyLoss > 0 {
		dailyLossLimit := &domain.RiskLimit{
			UserID:     req.UserId,
			LimitType:  "max_daily_loss",
			LimitValue: req.MaxDailyLoss,
			Period:     "daily",
			IsActive:   true,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		s.repo.SaveRiskLimit(ctx, dailyLossLimit)
	}

	return &pb.SetRiskLimitResponse{Success: true}, nil
}

func (s *RiskService) AssessRisk(ctx context.Context, req *pb.AssessRiskRequest) (*pb.AssessRiskResponse, error) {
	quantity, _ := decimal.NewFromString(req.Quantity)
	price, _ := decimal.NewFromString(req.Price)
	orderValue := quantity.Mul(price)

	marginRate := 0.1
	marginRequirement := orderValue.Mul(decimal.NewFromFloat(marginRate))

	riskScore := 0.0
	if req.Side == "buy" {
		riskScore = 30.0
	} else {
		riskScore = 25.0
	}

	limits, err := s.repo.ListRiskLimits(ctx, req.UserId)
	if err == nil && len(limits) > 0 {
		for _, limit := range limits {
			if limit.LimitType == "max_order_size" && orderValue.GreaterThan(decimal.NewFromFloat(limit.LimitValue)) {
				riskScore += 50
				break
			}
		}
	}

	riskLevel := "LOW"
	if riskScore >= 80 {
		riskLevel = "CRITICAL"
	} else if riskScore >= 60 {
		riskLevel = "HIGH"
	} else if riskScore >= 40 {
		riskLevel = "MEDIUM"
	}

	return &pb.AssessRiskResponse{
		IsAllowed:          riskScore < 80,
		Reason:             "",
		RiskLevel:          riskLevel,
		MarginRequirement:  marginRequirement.String(),
		RiskScore:          fmt.Sprintf("%.2f", riskScore),
	}, nil
}

func (s *RiskService) GetRiskMetrics(ctx context.Context, req *pb.GetRiskMetricsRequest) (*pb.GetRiskMetricsResponse, error) {
	metrics, err := s.repo.GetRiskMetrics(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	if metrics == nil {
		return &pb.GetRiskMetricsResponse{
			Metrics: &pb.RiskMetrics{
				Var95:        "0",
				Var99:        "0",
				MaxDrawdown:  "0",
				SharpeRatio:  "0",
				Correlation:  "0",
			},
		}, nil
	}

	return &pb.GetRiskMetricsResponse{
		Metrics: &pb.RiskMetrics{
			Var95:        fmt.Sprintf("%.2f", metrics.VaR95),
			Var99:        fmt.Sprintf("%.2f", metrics.VaR99),
			MaxDrawdown:  fmt.Sprintf("%.2f", metrics.MaxDrawdown),
			SharpeRatio:  fmt.Sprintf("%.2f", metrics.SharpeRatio),
			Correlation:  fmt.Sprintf("%.2f", metrics.Volatility),
		},
	}, nil
}

func (s *RiskService) CheckRiskLimit(ctx context.Context, req *pb.CheckRiskLimitRequest) (*pb.CheckRiskLimitResponse, error) {
	limit, err := s.repo.GetRiskLimit(ctx, req.UserId, req.LimitType)
	if err != nil {
		return nil, err
	}

	if limit == nil {
		return &pb.CheckRiskLimitResponse{
			LimitType:   req.LimitType,
			LimitValue:  "0",
			CurrentValue: "0",
			Remaining:   "0",
			IsExceeded:  false,
		}, nil
	}

	remaining := limit.LimitValue - limit.UsedValue

	return &pb.CheckRiskLimitResponse{
		LimitType:   limit.LimitType,
		LimitValue:  fmt.Sprintf("%.2f", limit.LimitValue),
		CurrentValue: fmt.Sprintf("%.2f", limit.UsedValue),
		Remaining:   fmt.Sprintf("%.2f", remaining),
		IsExceeded:  remaining < 0,
	}, nil
}

func (s *RiskService) GetRiskAlerts(ctx context.Context, req *pb.GetRiskAlertsRequest) (*pb.GetRiskAlertsResponse, error) {
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20
	}

	alerts, err := s.repo.ListRiskAlerts(ctx, req.UserId, limit)
	if err != nil {
		return nil, err
	}

	pbAlerts := make([]*pb.RiskAlert, len(alerts))
	for i, alert := range alerts {
		pbAlerts[i] = &pb.RiskAlert{
			AlertId:   alert.AlertID,
			AlertType: alert.AlertType,
			Severity:  alert.Severity,
			Message:   alert.Message,
			Timestamp: alert.CreatedAt.Unix(),
		}
	}

	return &pb.GetRiskAlertsResponse{Alerts: pbAlerts}, nil
}

func (s *RiskService) CalculatePortfolioRisk(ctx context.Context, req *pb.CalculatePortfolioRiskRequest) (*pb.CalculatePortfolioRiskResponse, error) {
	var totalValue float64
	var portfolioVolatility float64

	for _, asset := range req.Assets {
		position, _ := decimal.NewFromString(asset.Position)
		price, _ := decimal.NewFromString(asset.CurrentPrice)
		totalValue += position.Mul(price).Float64()
		portfolioVolatility += asset.Volatility * asset.Volatility
	}

	portfolioVolatility = math.Sqrt(portfolioVolatility)

	confidenceMultiplier := 1.65
	if req.ConfidenceLevel >= 0.99 {
		confidenceMultiplier = 2.33
	} else if req.ConfidenceLevel >= 0.95 {
		confidenceMultiplier = 1.95
	}

	var95 := totalValue * portfolioVolatility * confidenceMultiplier
	var99 := totalValue * portfolioVolatility * (confidenceMultiplier + 0.5)

	return &pb.CalculatePortfolioRiskResponse{
		PortfolioValue:  fmt.Sprintf("%.2f", totalValue),
		Volatility:      fmt.Sprintf("%.2f", portfolioVolatility*100),
		ValueAtRisk95:   fmt.Sprintf("%.2f", var95),
		ValueAtRisk99:   fmt.Sprintf("%.2f", var99),
		SharpeRatio:     fmt.Sprintf("%.2f", totalValue*0.15/var95),
	}, nil
}

func (s *RiskService) CalculateMonteCarloRisk(ctx context.Context, req *pb.CalculateMonteCarloRiskRequest) (*pb.MonteCarloRiskResponse, error) {
	if req.Simulations <= 0 {
		req.Simulations = 1000
	}
	if req.ConfidenceLevel <= 0 {
		req.ConfidenceLevel = 0.95
	}

	rand.Seed(time.Now().UnixNano())

	initialValue, _ := decimal.NewFromString(req.InitialValue)
	returnValue, _ := decimal.NewFromString(req.ExpectedReturn)
	volatility, _ := decimal.NewFromString(req.Volatility)

	results := make([]float64, req.Simulations)
	for i := 0; i < int(req.Simulations); i++ {
		z := rand.NormFloat64()
		simReturn := returnValue.Float64() + volatility.Float64()*z
		results[i] = initialValue.Float64() * (1 + simReturn)
	}

	sort.Float64s(results)

	percentile := int((1 - req.ConfidenceLevel) * float64(req.Simulations))
	VaR := initialValue.Float64() - results[percentile]

	var50, var95, var99 float64
	p50 := int(0.5 * float64(req.Simulations))
	p95 := int(0.95 * float64(req.Simulations))
	p99 := int(0.99 * float64(req.Simulations))

	if p50 < len(results) {
		var50 = initialValue.Float64() - results[p50]
	}
	if p95 < len(results) {
		var95 = initialValue.Float64() - results[p95]
	}
	if p99 < len(results) {
		var99 = initialValue.Float64() - results[p99]
	}

	var returns []float64
	for i := 1; i < len(results); i++ {
		returns = append(returns, (results[i]-results[i-1])/results[i-1])
	}

	avgReturn := 0.0
	if len(returns) > 0 {
		for _, r := range returns {
			avgReturn += r
		}
		avgReturn /= float64(len(returns))
	}

	maxDrawdown := 0.0
	peak := results[0]
	for _, v := range results {
		if v > peak {
			peak = v
		}
		dd := (peak - v) / peak
		if dd > maxDrawdown {
			maxDrawdown = dd
		}
	}

	return &pb.MonteCarloRiskResponse{
		Simulations:      req.Simulations,
		FinalValueMean:    fmt.Sprintf("%.2f", initialValue.Float64()*(1+returnValue.Float64())),
		ValueAtRisk50:    fmt.Sprintf("%.2f", var50),
		ValueAtRisk95:    fmt.Sprintf("%.2f", var95),
		ValueAtRisk99:    fmt.Sprintf("%.2f", var99),
		MaxDrawdown:      fmt.Sprintf("%.2f", maxDrawdown*100),
		PercentileValues:  results,
	}, nil
}

func (s *RiskService) RunStressTest(ctx context.Context, req *pb.RunStressTestRequest) (*pb.RunStressTestResponse, error) {
	scenarios := map[string]float64{
		"market_crash":    -0.30,
		"interest_rate_hike": -0.15,
		"liquidity_crisis": -0.25,
		"black_swan":      -0.50,
	}

	results := make(map[string]*pb.StressTestResult)
	initialValue, _ := decimal.NewFromString(req.InitialValue)

	for scenario, impact := range scenarios {
		stressValue := initialValue.Mul(decimal.NewFromFloat(1 + impact))
		loss := initialValue.Sub(stressValue)

		results[scenario] = &pb.StressTestResult{
			Scenario:     scenario,
			InitialValue: req.InitialValue,
			StressValue:  stressValue.String(),
			Loss:         loss.String(),
			Impact:       fmt.Sprintf("%.2f", impact*100) + "%",
		}
	}

	return &pb.RunStressTestResponse{
		Results: results,
	}, nil
}

func (s *RiskService) GetAnomalyReport(ctx context.Context, req *pb.GetAnomalyReportRequest) (*pb.GetAnomalyReportResponse, error) {
	metrics, err := s.repo.GetRiskMetrics(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	var anomalies []*pb.Anomaly

	if metrics != nil {
		if metrics.MaxDrawdown > 20 {
			anomalies = append(anomalies, &pb.Anomaly{
				Type:        "HIGH_DRAWDOWN",
				Severity:    "HIGH",
				Description: fmt.Sprintf("Max drawdown %.2f%% exceeds threshold", metrics.MaxDrawdown),
				Timestamp:   time.Now().Unix(),
			})
		}

		if metrics.Volatility > 0.5 {
			anomalies = append(anomalies, &pb.Anomaly{
				Type:        "HIGH_VOLATILITY",
				Severity:    "MEDIUM",
				Description: fmt.Sprintf("Volatility %.2f%% is abnormally high", metrics.Volatility*100),
				Timestamp:   time.Now().Unix(),
			})
		}

		if metrics.WinRate < 0.4 && metrics.WinRate > 0 {
			anomalies = append(anomalies, &pb.Anomaly{
				Type:        "LOW_WIN_RATE",
				Severity:    "MEDIUM",
				Description: fmt.Sprintf("Win rate %.2f%% is below acceptable threshold", metrics.WinRate*100),
				Timestamp:   time.Now().Unix(),
			})
		}
	}

	return &pb.GetAnomalyReportResponse{
		Anomalies: anomalies,
	}, nil
}
