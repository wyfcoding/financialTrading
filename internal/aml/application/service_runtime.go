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
	isSuspicious := riskLevel == "HIGH"

	if isSuspicious {
		alertID = fmt.Sprintf("aml_%d", time.Now().UnixNano())
		alert := &domain.AMLAlert{
			AlertID:     alertID,
			UserID:      req.UserId,
			Type:        "TRANSACTION",
			Status:      "NEW",
			RiskLevel:   riskLevel,
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
			RiskLevel: "LOW",
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

	items := make([]*pb.AlertItem, 0, len(alerts))
	for _, a := range alerts {
		if a == nil {
			continue
		}
		items = append(items, &pb.AlertItem{
			AlertId:     a.AlertID,
			UserId:      a.UserID,
			Type:        a.Type,
			Description: a.Description,
			Status:      a.Status,
			CreatedAt:   timestamppb.New(a.CreatedAt),
		})
	}

	return &pb.ListAlertsResponse{Alerts: items}, nil
}

func classifyRisk(amount string) string {
	value := strings.TrimSpace(amount)
	switch {
	case value == "":
		return "LOW"
	case strings.HasPrefix(value, "-"):
		return "LOW"
	case len(strings.SplitN(value, ".", 2)[0]) >= 6:
		return "HIGH"
	case len(strings.SplitN(value, ".", 2)[0]) >= 5:
		return "MEDIUM"
	default:
		return "LOW"
	}
}

func normalizeRiskLevel(level string) string {
	upper := strings.ToUpper(strings.TrimSpace(level))
	switch upper {
	case "LOW", "MEDIUM", "HIGH":
		return upper
	default:
		return "LOW"
	}
}
