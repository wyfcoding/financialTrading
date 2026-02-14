package application

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialtrading/go-api/aml/v1"
	"github.com/wyfcoding/financialtrading/internal/aml/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AMLService struct {
	repo      domain.AMLRepository
	ruleEngine *RuleEngine
	screening *ScreeningService
}

func NewAMLService(repo domain.AMLRepository) *AMLService {
	return &AMLService{
		repo:        repo,
		ruleEngine:  NewRuleEngine(repo),
		screening:   NewScreeningService(repo),
	}
}

type RuleEngine struct {
	repo domain.AMLRepository
}

func NewRuleEngine(repo domain.AMLRepository) *RuleEngine {
	return &RuleEngine{repo: repo}
}

type ScreeningService struct {
	repo domain.AMLRepository
}

func NewScreeningService(repo domain.AMLRepository) *ScreeningService {
	return &ScreeningService{repo: repo}
}

func (s *AMLService) MonitorTransaction(ctx context.Context, req *pb.MonitorTransactionRequest) (*pb.MonitorTransactionResponse, error) {
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, err
	}

	isSuspicious := false
	alertID := ""
	riskLevel := pb.RiskLevel_RISK_LEVEL_LOW
	var triggeredRules []string
	var recommendations []string
	riskScore := 0.0

	rules, err := s.repo.ListActiveRules(ctx)
	if err != nil {
		return nil, err
	}

	for _, rule := range rules {
		if s.evaluateRule(rule, req, amount) {
			isSuspicious = true
			triggeredRules = append(triggeredRules, rule.Name)
			
			switch rule.DefaultRiskLevel {
			case pb.RiskLevel_RISK_LEVEL_HIGH:
				riskScore += 40
			case pb.RiskLevel_RISK_LEVEL_MEDIUM:
				riskScore += 20
			case pb.RiskLevel_RISK_LEVEL_CRITICAL:
				riskScore += 60
			default:
				riskScore += 10
			}

			if len(rule.Actions) > 0 {
				recommendations = append(recommendations, rule.Actions...)
			}
		}
	}

	if isSuspicious {
		riskLevel = s.calculateRiskLevel(riskScore)
		alertID = fmt.Sprintf("alert_%d", time.Now().UnixNano())

		alert := &domain.AMLAlert{
			AlertID:     alertID,
			UserID:      req.UserId,
			Type:        s.determineAlertType(triggeredRules),
			Description: fmt.Sprintf("Transaction %s triggered rules: %v", req.TransactionId, triggeredRules),
			Status:      string(pb.AlertStatus_ALERT_STATUS_NEW),
			RiskLevel:  string(riskLevel),
			CreatedAt:   time.Now(),
		}
		if err := s.repo.SaveAlert(ctx, alert); err != nil {
			return nil, err
		}
	}

	return &pb.MonitorTransactionResponse{
		IsSuspicious:    isSuspicious,
		AlertId:         alertID,
		RiskLevel:       riskLevel,
		TriggeredRules:  triggeredRules,
		RiskScore:       riskScore,
		Recommendations: recommendations,
	}, nil
}

func (s *RuleEngine) evaluateRule(rule *domain.AMLRule, req *pb.MonitorTransactionRequest, amount decimal.Decimal) bool {
	amountFloat, _ := amount.Float64()

	switch rule.Type {
	case string(pb.RuleType_RULE_TYPE_THRESHOLD):
		return s.evaluateThreshold(rule, amountFloat)
	case string(pb.RuleType_RULE_TYPE_VELOCITY):
		return s.evaluateVelocity(rule, req)
	case string(pb.RuleType_RULE_TYPE_PATTERN):
		return s.evaluatePattern(rule, req)
	default:
		return false
	}
}

func (s *RuleEngine) evaluateThreshold(rule *domain.AMLRule, amount float64) bool {
	var condition struct {
		MinAmount float64 `json:"min_amount"`
		MaxAmount float64 `json:"max_amount"`
	}
	if err := json.Unmarshal([]byte(rule.Condition), &condition); err != nil {
		return false
	}
	
	if condition.MinAmount > 0 && amount < condition.MinAmount {
		return false
	}
	if condition.MaxAmount > 0 && amount > condition.MaxAmount {
		return true
	}
	return false
}

func (s *RuleEngine) evaluateVelocity(rule *domain.AMLRule, req *pb.MonitorTransactionRequest) bool {
	var condition struct {
		MaxTransactionsPerHour int     `json:"max_transactions_per_hour"`
		MaxAmountPerHour       float64 `json:"max_amount_per_hour"`
	}
	if err := json.Unmarshal([]byte(rule.Condition), &condition); err != nil {
		return false
	}
	return false
}

func (s *RuleEngine) evaluatePattern(rule *domain.AMLRule, req *pb.MonitorTransactionRequest) bool {
	var condition struct {
		Countries        []string `json:"countries"`
		TransactionTypes  []string `json:"transaction_types"`
	}
	if err := json.Unmarshal([]byte(rule.Condition), &condition); err != nil {
		return false
	}
	
	for _, country := range condition.Countries {
		if req.SourceCountry == country || req.DestinationCountry == country {
			return true
		}
	}
	return false
}

func (s *AMLService) calculateRiskLevel(score float64) pb.RiskLevel {
	switch {
	case score >= 80:
		return pb.RiskLevel_RISK_LEVEL_CRITICAL
	case score >= 60:
		return pb.RiskLevel_RISK_LEVEL_HIGH
	case score >= 40:
		return pb.RiskLevel_RISK_LEVEL_MEDIUM
	default:
		return pb.RiskLevel_RISK_LEVEL_LOW
	}
}

func (s *AMLService) determineAlertType(rules []string) string {
	for _, r := range rules {
		lower := strings.ToLower(r)
		if strings.Contains(lower, "structur") {
			return string(pb.AlertType_ALERT_TYPE_STRUCTURING)
		}
		if strings.Contains(lower, "sanction") {
			return string(pb.AlertType_ALERT_TYPE_SANCTIONS_MATCH)
		}
		if strings.Contains(lower, "pep") {
			return string(pb.AlertType_ALERT_TYPE_PEP_ASSOCIATION)
		}
	}
	return string(pb.AlertType_ALERT_TYPE_LARGE_TRANSACTION)
}

func (s *AMLService) BatchMonitorTransactions(ctx context.Context, req *pb.BatchMonitorTransactionsRequest) (*pb.BatchMonitorTransactionsResponse, error) {
	results := make([]*pb.MonitorTransactionResponse, 0, len(req.Transactions))
	suspiciousCount := 0

	for _, tx := range req.Transactions {
		result, err := s.MonitorTransaction(ctx, tx)
		if err != nil {
			continue
		}
		results = append(results, result)
		if result.IsSuspicious {
			suspiciousCount++
		}
	}

	return &pb.BatchMonitorTransactionsResponse{
		Results:          results,
		TotalProcessed:   int32(len(req.Transactions)),
		SuspiciousCount:  int32(suspiciousCount),
	}, nil
}

func (s *AMLService) ScreenTransaction(ctx context.Context, req *pb.ScreenTransactionRequest) (*pb.ScreenTransactionResponse, error) {
	return s.screening.ScreenTransaction(ctx, req)
}

func (s *ScreeningService) ScreenTransaction(ctx context.Context, req *pb.ScreenTransactionRequest) (*pb.ScreenTransactionResponse, error) {
	response := &pb.ScreenTransactionResponse{
		Passed:       true,
		Warnings:     []string{},
		OverallRisk:  pb.RiskLevel_RISK_LEVEL_LOW,
	}

	watchlist, err := s.repo.GetWatchlist(ctx)
	if err != nil {
		return nil, err
	}

	for _, entry := range watchlist {
		if req.CounterpartyName != "" && strings.Contains(strings.ToLower(entry.Name), strings.ToLower(req.CounterpartyName)) {
			response.Passed = false
			response.WatchlistMatches = append(response.WatchlistMatches, &pb.WatchlistMatch{
				ListName:    entry.ListType,
				MatchedName: entry.Name,
				MatchScore:  0.95,
				Reason:      entry.Reason,
			})
			response.OverallRisk = pb.RiskLevel_RISK_LEVEL_HIGH
		}
	}

	highRiskCountries := []string{"KP", "IR", "SY", "CU", "RU"}
	for _, country := range []string{req.SourceCountry, req.DestinationCountry} {
		for _, risk := range highRiskCountries {
			if country == risk {
				response.Warnings = append(response.Warnings, fmt.Sprintf("High risk country: %s", country))
				if response.OverallRisk == pb.RiskLevel_RISK_LEVEL_LOW {
					response.OverallRisk = pb.RiskLevel_RISK_LEVEL_MEDIUM
				}
			}
		}
	}

	return response, nil
}

func (s *AMLService) GetRiskScore(ctx context.Context, req *pb.GetRiskScoreRequest) (*pb.GetRiskScoreResponse, error) {
	score, err := s.repo.GetRiskScore(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	if score == nil {
		return &pb.GetRiskScoreResponse{
			UserId:     req.UserId,
			Score:      10.0,
			RiskLevel:  pb.RiskLevel_RISK_LEVEL_LOW,
			LastUpdated: timestamppb.Now(),
		}, nil
	}

	riskLevel := pb.RiskLevel_RISK_LEVEL_LOW
	switch {
	case score.Score >= 80:
		riskLevel = pb.RiskLevel_RISK_LEVEL_CRITICAL
	case score.Score >= 60:
		riskLevel = pb.RiskLevel_RISK_LEVEL_HIGH
	case score.Score >= 40:
		riskLevel = pb.RiskLevel_RISK_LEVEL_MEDIUM
	}

	return &pb.GetRiskScoreResponse{
		UserId:     score.UserID,
		Score:      score.Score,
		RiskLevel:  riskLevel,
		LastUpdated: timestamppb.New(score.UpdatedAt),
	}, nil
}

func (s *AMLService) CalculateRiskScore(ctx context.Context, req *pb.CalculateRiskScoreRequest) (*pb.CalculateRiskScoreResponse, error) {
	profile, err := s.repo.GetUserRiskProfile(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	var factors []*pb.RiskFactor
	score := 0.0

	if profile != nil {
		if profile.IsPep {
			factors = append(factors, &pb.RiskFactor{
				Name:        "PEP Association",
				Weight:      30,
				Value:       30,
				Description: "Politically Exposed Person",
			})
			score += 30
		}
		if profile.IsSanctioned {
			factors = append(factors, &pb.RiskFactor{
				Name:        "Sanctioned",
				Weight:      50,
				Value:       50,
				Description: "On sanctions list",
			})
			score += 50
		}
		score += float64(len(profile.HighRiskCountries) * 10)
	}

	kycLevel := 0
	if profile != nil {
		fmt.Sscanf(profile.KycLevel, "%d", &kycLevel)
	}
	score -= float64(kycLevel * 5)
	score = math.Max(0, math.Min(100, score))

	riskLevel := pb.RiskLevel_RISK_LEVEL_LOW
	switch {
	case score >= 80:
		riskLevel = pb.RiskLevel_RISK_LEVEL_CRITICAL
	case score >= 60:
		riskLevel = pb.RiskLevel_RISK_LEVEL_HIGH
	case score >= 40:
		riskLevel = pb.RiskLevel_RISK_LEVEL_MEDIUM
	}

	now := time.Now()
	if err := s.repo.SaveRiskScore(ctx, &domain.UserRiskScore{
		UserID:    req.UserId,
		Score:     score,
		RiskLevel: string(riskLevel),
		UpdatedAt: now,
	}); err != nil {
		return nil, err
	}

	return &pb.CalculateRiskScoreResponse{
		UserId:       req.UserId,
		Score:        score,
		RiskLevel:    riskLevel,
		Factors:      factors,
		CalculatedAt: timestamppb.New(now),
	}, nil
}

func (s *AMLService) GetUserRiskProfile(ctx context.Context, req *pb.GetUserRiskProfileRequest) (*pb.UserRiskProfile, error) {
	profile, err := s.repo.GetUserRiskProfile(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return &pb.UserRiskProfile{
			UserId:       req.UserId,
			OverallScore: 10.0,
			RiskLevel:    pb.RiskLevel_RISK_LEVEL_LOW,
			KycLevel:     "0",
		}, nil
	}

	riskLevel := pb.RiskLevel_RISK_LEVEL_LOW
	switch profile.RiskLevel {
	case "HIGH":
		riskLevel = pb.RiskLevel_RISK_LEVEL_HIGH
	case "MEDIUM":
		riskLevel = pb.RiskLevel_RISK_LEVEL_MEDIUM
	case "CRITICAL":
		riskLevel = pb.RiskLevel_RISK_LEVEL_CRITICAL
	}

	return &pb.UserRiskProfile{
		UserId:             profile.UserID,
		OverallScore:       profile.Score,
		RiskLevel:          riskLevel,
		KycLevel:           profile.KycLevel,
		IsPep:              profile.IsPep,
		IsSanctioned:       profile.IsSanctioned,
		HighRiskCountries:  profile.HighRiskCountries,
		CreatedAt:          timestamppb.New(profile.CreatedAt),
		UpdatedAt:          timestamppb.New(profile.UpdatedAt),
	}, nil
}

func (s *AMLService) ListAlerts(ctx context.Context, req *pb.ListAlertsRequest) (*pb.ListAlertsResponse, error) {
	var status string
	if req.Status != 0 {
		status = req.Status.String()
	}

	alerts, err := s.repo.ListAlerts(ctx, status, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, err
	}

	var pbAlerts []*pb.AMLAlert
	for _, a := range alerts {
		alertStatus := pb.AlertStatus_ALERT_STATUS_NEW
		switch a.Status {
		case "UNDER_REVIEW":
			alertStatus = pb.AlertStatus_ALERT_STATUS_UNDER_REVIEW
		case "ESCALATED":
			alertStatus = pb.AlertStatus_ALERT_STATUS_ESCALATED
		case "CLOSED":
			alertStatus = pb.AlertStatus_ALERT_STATUS_CLOSED
		}

		riskLevel := pb.RiskLevel_RISK_LEVEL_LOW
		switch a.RiskLevel {
		case "HIGH":
			riskLevel = pb.RiskLevel_RISK_LEVEL_HIGH
		case "MEDIUM":
			riskLevel = pb.RiskLevel_RISK_LEVEL_MEDIUM
		case "CRITICAL":
			riskLevel = pb.RiskLevel_RISK_LEVEL_CRITICAL
		}

		pbAlerts = append(pbAlerts, &pb.AMLAlert{
			AlertId:     a.AlertID,
			UserId:      a.UserID,
			Status:      alertStatus,
			RiskLevel:   riskLevel,
			Title:       a.Type,
			Description: a.Description,
			CreatedAt:   timestamppb.New(a.CreatedAt),
			UpdatedAt:   timestamppb.New(a.UpdatedAt),
		})
	}

	return &pb.ListAlertsResponse{
		Alerts:   pbAlerts,
		Total:    int32(len(alerts)),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (s *AMLService) ReviewAlert(ctx context.Context, req *pb.ReviewAlertRequest) (*pb.AMLAlert, error) {
	alert, err := s.repo.GetAlert(ctx, req.AlertId)
	if err != nil {
		return nil, err
	}
	if alert == nil {
		return nil, fmt.Errorf("alert not found")
	}

	if req.IsFalsePositive {
		alert.Status = string(pb.AlertStatus_ALERT_STATUS_FALSE_POSITIVE)
	} else {
		alert.Status = string(pb.AlertStatus_ALERT_STATUS_CONFIRMED)
	}
	alert.UpdatedAt = time.Now()

	if err := s.repo.UpdateAlert(ctx, alert); err != nil {
		return nil, err
	}

	history := &domain.AlertHistory{
		AlertID:     req.AlertId,
		Action:      "REVIEWED",
		PerformedBy: req.ReviewerId,
		Notes:       req.Notes,
		Timestamp:   time.Now(),
	}
	s.repo.SaveAlertHistory(ctx, history)

	return &pb.AMLAlert{
		AlertId:      alert.AlertID,
		Status:       pb.AlertStatus_ALERT_STATUS_UNDER_REVIEW,
		UpdatedAt:    timestamppb.New(alert.UpdatedAt),
		ResolutionNotes: req.Notes,
	}, nil
}

func (s *AMLService) CreateRule(ctx context.Context, req *pb.CreateRuleRequest) (*pb.AMLRule, error) {
	rule := &domain.AMLRule{
		RuleID:           fmt.Sprintf("rule_%d", time.Now().Unix()),
		Name:             req.Name,
		Description:      req.Description,
		Type:             string(req.Type),
		Condition:        req.Condition,
		Actions:          req.Actions,
		DefaultRiskLevel: string(req.DefaultRiskLevel),
		Priority:         req.Priority,
		IsActive:         true,
		CreatedAt:        time.Now(),
		CreatedBy:        req.CreatedBy,
	}

	if err := s.repo.SaveRule(ctx, rule); err != nil {
		return nil, err
	}

	return &pb.AMLRule{
		RuleId:             rule.RuleID,
		Name:               rule.Name,
		Description:        rule.Description,
		Type:               req.Type,
		IsActive:           rule.IsActive,
		Priority:           rule.Priority,
		Condition:          rule.Condition,
		Actions:            rule.Actions,
		DefaultRiskLevel:   req.DefaultRiskLevel,
		CreatedAt:          timestamppb.New(rule.CreatedAt),
		CreatedBy:          rule.CreatedBy,
	}, nil
}

func (s *AMLService) ListRules(ctx context.Context, req *pb.ListRulesRequest) (*pb.ListRulesResponse, error) {
	rules, err := s.repo.ListRules(ctx, string(req.Type), req.IsActive, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, err
	}

	var pbRules []*pb.AMLRule
	for _, r := range rules {
		pbRules = append(pbRules, &pb.AMLRule{
			RuleId:             r.RuleID,
			Name:               r.Name,
			Description:        r.Description,
			IsActive:           r.IsActive,
			Priority:           r.Priority,
			Condition:          r.Condition,
			DefaultRiskLevel:   pb.RiskLevel_RISK_LEVEL_LOW,
			UpdatedAt:          timestamppb.New(r.UpdatedAt),
		})
	}

	return &pb.ListRulesResponse{
		Rules:    pbRules,
		Total:    int32(len(rules)),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (s *AMLService) ScreenAgainstLists(ctx context.Context, req *pb.ScreenAgainstListsRequest) (*pb.ScreenAgainstListsResponse, error) {
	response := &pb.ScreenAgainstListsResponse{
		HasMatches: false,
	}

	watchlist, err := s.repo.GetWatchlist(ctx)
	if err != nil {
		return nil, err
	}

	reqNameLower := strings.ToLower(req.Name)
	for _, entry := range watchlist {
		entryNameLower := strings.ToLower(entry.Name)
		
		matchScore := s.calculateMatchScore(reqNameLower, entryNameLower, req.Aliases)
		if matchScore > 0.8 {
			response.HasMatches = true
			response.WatchlistMatches = append(response.WatchlistMatches, &pb.WatchlistMatch{
				ListName:    entry.ListType,
				MatchedName: entry.Name,
				MatchScore:  matchScore,
				Reason:      entry.Reason,
			})
		}
	}

	for _, alias := range req.Aliases {
		aliasLower := strings.ToLower(alias)
		for _, entry := range watchlist {
			entryLower := strings.ToLower(entry.Name)
			score := s.calculateMatchScore(aliasLower, entryLower, nil)
			if score > 0.8 {
				response.HasMatches = true
				response.WatchlistMatches = append(response.WatchlistMatches, &pb.WatchlistMatch{
					ListName:    entry.ListType,
					MatchedName: entry.Name,
					MatchScore:  score,
				})
			}
		}
	}

	sort.Slice(response.WatchlistMatches, func(i, j int) bool {
		return response.WatchlistMatches[i].MatchScore > response.WatchlistMatches[j].MatchScore
	})

	return response, nil
}

func (s *AMLService) calculateMatchScore(name1 string, name2 string, aliases []string) float64 {
	if name1 == name2 {
		return 1.0
	}

	commonChars := 0
	minLen := len(name1)
	if len(name2) < minLen {
		minLen = len(name2)
	}
	
	for i := 0; i < minLen; i++ {
		if name1[i] == name2[i] {
			commonChars++
		}
	}

	baseScore := float64(commonChars) / float64(max(len(name1), len(name2)))
	
	levenshtein := s.levenshteinDistance(name1, name2)
	maxLen := max(len(name1), len(name2))
	penalty := float64(levenshtein) / float64(maxLen)
	
	return baseScore * (1 - penalty*0.5)
}

func (s *AMLService) levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,
				min(matrix[i][j-1]+1, matrix[i-1][j-1]+cost),
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

func (s *AMLService) GetAMLStatistics(ctx context.Context, req *pb.GetAMLStatisticsRequest) (*pb.AMLStatistics, error) {
	alerts, err := s.repo.ListAlerts(ctx, "", 1, 1000)
	if err != nil {
		return nil, err
	}

	stats := &pb.AMLStatistics{
		TotalTransactionsMonitored: 10000,
		TotalAlertsGenerated:       int64(len(alerts)),
		AlertsByRiskCritical:       0,
		AlertsByRiskHigh:           0,
		AlertsByRiskMedium:         0,
		AlertsByRiskLow:            int64(len(alerts)),
		FalsePositives:             0,
		ConfirmedSuspicious:        0,
	}

	for _, alert := range alerts {
		switch alert.RiskLevel {
		case "CRITICAL":
			stats.AlertsByRiskCritical++
		case "HIGH":
			stats.AlertsByRiskHigh++
		case "MEDIUM":
			stats.AlertsByRiskMedium++
		case "LOW":
			stats.AlertsByRiskLow++
		}
		if alert.Status == "FALSE_POSITIVE" {
			stats.FalsePositives++
		}
		if alert.Status == "CONFIRMED" {
			stats.ConfirmedSuspicious++
		}
	}

	return stats, nil
}

func (s *AMLService) GenerateSARReport(ctx context.Context, req *pb.GenerateSARReportRequest) (*pb.SARReport, error) {
	reportID := fmt.Sprintf("sar_%d", time.Now().Unix())
	
	report := &domain.SARReport{
		ReportID:         reportID,
		UserID:            req.UserId,
		AlertIDs:          req.AlertIds,
		Narrative:         req.Narrative,
		FilingStatus:      "DRAFT",
		RegulatoryBody:    req.RegulatoryBody,
		CreatedAt:         time.Now(),
	}

	if err := s.repo.SaveSARReport(ctx, report); err != nil {
		return nil, err
	}

	return &pb.SARReport{
		ReportId:         report.ReportID,
		UserId:           report.UserID,
		AlertIds:         report.AlertIDs,
		Narrative:        report.Narrative,
		FilingStatus:     report.FilingStatus,
		CreatedAt:        timestamppb.New(report.CreatedAt),
		RegulatoryBody:   report.RegulatoryBody,
	}, nil
}
