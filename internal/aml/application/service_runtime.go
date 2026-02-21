package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	pb "github.com/wyfcoding/financialtrading/go-api/aml/v1"
	"github.com/wyfcoding/financialtrading/internal/aml/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type amlRepo interface {
	SaveAlert(ctx context.Context, alert *domain.AMLAlert) error
	ListAlertsByStatus(ctx context.Context, status string) ([]*domain.AMLAlert, error)
	GetRiskScore(ctx context.Context, userID string) (*domain.UserRiskScore, error)
	SaveRiskScore(ctx context.Context, score *domain.UserRiskScore) error
}

type AMLService struct {
	repo amlRepo
}

func NewAMLService(repo amlRepo) *AMLService {
	return &AMLService{repo: repo}
}

func (s *AMLService) MonitorTransaction(ctx context.Context, req *pb.MonitorTransactionRequest) (*pb.MonitorTransactionResponse, error) {
	alertID := ""
	riskLevel := classifyRisk(req.Amount)
	isSuspicious := riskLevel >= pb.RiskLevel_RISK_LEVEL_HIGH

	if isSuspicious {
		alertID = fmt.Sprintf("aml_%d", time.Now().UnixNano())
		alert := &domain.AMLAlert{
			AlertID:     alertID,
			UserID:      req.UserId,
			Type:        "TRANSACTION",
			Status:      "NEW",
			RiskLevel:   riskLevel.String(),
			Title:       "Suspicious transaction",
			Description: fmt.Sprintf("transaction=%s amount=%s currency=%s", req.TransactionId, req.Amount, req.Currency),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := s.repo.SaveAlert(ctx, alert); err != nil {
			return nil, err
		}
	}

	return &pb.MonitorTransactionResponse{
		IsSuspicious: isSuspicious,
		AlertId:      alertID,
		RiskLevel:    riskLevel,
	}, nil
}

func (s *AMLService) GetRiskScore(ctx context.Context, userID string) (*pb.GetRiskScoreResponse, error) {
	score, err := s.repo.GetRiskScore(ctx, userID)
	if err != nil {
		return nil, err
	}
	if score == nil {
		return &pb.GetRiskScoreResponse{
			UserId:    userID,
			Score:     0,
			RiskLevel: pb.RiskLevel_RISK_LEVEL_LOW,
		}, nil
	}
	return &pb.GetRiskScoreResponse{
		UserId:    score.UserID,
		Score:     score.Score,
		RiskLevel: normalizeRiskLevel(score.RiskLevel),
	}, nil
}

func (s *AMLService) ListAlerts(ctx context.Context, status string) (*pb.ListAlertsResponse, error) {
	alerts, err := s.repo.ListAlertsByStatus(ctx, status)
	if err != nil {
		return nil, err
	}

	items := make([]*pb.AMLAlert, 0, len(alerts))
	for _, a := range alerts {
		if a == nil {
			continue
		}
		items = append(items, &pb.AMLAlert{
			AlertId:     a.AlertID,
			UserId:      a.UserID,
			Type:        mapAlertType(a.Type),
			Status:      mapAlertStatus(a.Status),
			RiskLevel:   normalizeRiskLevel(a.RiskLevel),
			Title:       a.Title,
			Description: a.Description,
			AssignedTo:  a.AssignedTo,
			CreatedAt:   timestamppb.New(a.CreatedAt),
			UpdatedAt:   timestamppb.New(a.UpdatedAt),
		})
	}

	return &pb.ListAlertsResponse{Alerts: items}, nil
}

func classifyRisk(amount string) pb.RiskLevel {
	value := strings.TrimSpace(amount)
	switch {
	case value == "":
		return pb.RiskLevel_RISK_LEVEL_LOW
	case strings.HasPrefix(value, "-"):
		return pb.RiskLevel_RISK_LEVEL_LOW
	case len(strings.SplitN(value, ".", 2)[0]) >= 6:
		return pb.RiskLevel_RISK_LEVEL_HIGH
	case len(strings.SplitN(value, ".", 2)[0]) >= 5:
		return pb.RiskLevel_RISK_LEVEL_MEDIUM
	default:
		return pb.RiskLevel_RISK_LEVEL_LOW
	}
}

func normalizeRiskLevel(level string) pb.RiskLevel {
	upper := strings.ToUpper(strings.TrimSpace(level))
	switch upper {
	case "RISK_LEVEL_LOW", "LOW":
		return pb.RiskLevel_RISK_LEVEL_LOW
	case "RISK_LEVEL_MEDIUM", "MEDIUM":
		return pb.RiskLevel_RISK_LEVEL_MEDIUM
	case "RISK_LEVEL_HIGH", "HIGH":
		return pb.RiskLevel_RISK_LEVEL_HIGH
	case "RISK_LEVEL_CRITICAL", "CRITICAL":
		return pb.RiskLevel_RISK_LEVEL_CRITICAL
	default:
		return pb.RiskLevel_RISK_LEVEL_LOW
	}
}

func mapAlertType(t string) pb.AlertType {
	switch strings.ToUpper(strings.TrimSpace(t)) {
	case "TRANSACTION", "LARGE_TRANSACTION":
		return pb.AlertType_ALERT_TYPE_LARGE_TRANSACTION
	case "FREQUENT_TRANSACTION":
		return pb.AlertType_ALERT_TYPE_FREQUENT_TRANSACTION
	case "STRUCTURING":
		return pb.AlertType_ALERT_TYPE_STRUCTURING
	case "RAPID_MOVEMENT":
		return pb.AlertType_ALERT_TYPE_RAPID_MOVEMENT
	case "HIGH_RISK_COUNTRY":
		return pb.AlertType_ALERT_TYPE_HIGH_RISK_COUNTRY
	case "SANCTIONS_MATCH":
		return pb.AlertType_ALERT_TYPE_SANCTIONS_MATCH
	case "PEP_ASSOCIATION":
		return pb.AlertType_ALERT_TYPE_PEP_ASSOCIATION
	case "UNUSUAL_PATTERN":
		return pb.AlertType_ALERT_TYPE_UNUSUAL_PATTERN
	case "ROUND_TRIPPING":
		return pb.AlertType_ALERT_TYPE_ROUND_TRIPPING
	case "SHELL_COMPANY":
		return pb.AlertType_ALERT_TYPE_SHELL_COMPANY
	default:
		return pb.AlertType_ALERT_TYPE_UNSPECIFIED
	}
}

func mapAlertStatus(s string) pb.AlertStatus {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "NEW":
		return pb.AlertStatus_ALERT_STATUS_NEW
	case "UNDER_REVIEW":
		return pb.AlertStatus_ALERT_STATUS_UNDER_REVIEW
	case "ESCALATED":
		return pb.AlertStatus_ALERT_STATUS_ESCALATED
	case "FALSE_POSITIVE":
		return pb.AlertStatus_ALERT_STATUS_FALSE_POSITIVE
	case "CONFIRMED":
		return pb.AlertStatus_ALERT_STATUS_CONFIRMED
	case "REPORTED":
		return pb.AlertStatus_ALERT_STATUS_REPORTED
	case "CLOSED":
		return pb.AlertStatus_ALERT_STATUS_CLOSED
	default:
		return pb.AlertStatus_ALERT_STATUS_UNSPECIFIED
	}
}
