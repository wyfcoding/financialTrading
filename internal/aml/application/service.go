package application

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialtrading/go-api/aml/v1"
	"github.com/wyfcoding/financialtrading/internal/aml/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AMLService struct {
	repo domain.AMLRepository
}

func NewAMLService(repo domain.AMLRepository) *AMLService {
	return &AMLService{repo: repo}
}

func (s *AMLService) MonitorTransaction(ctx context.Context, req *pb.MonitorTransactionRequest) (*pb.MonitorTransactionResponse, error) {
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, err
	}

	isSuspicious := false
	alertID := ""
	riskLevel := "LOW"

	// 规则 1：金额超过 10000 视为可疑
	if amount.GreaterThanOrEqual(decimal.NewFromInt(10000)) {
		isSuspicious = true
		riskLevel = "HIGH"
		alertID = fmt.Sprintf("alert_%d", time.Now().UnixNano())

		alert := &domain.AMLAlert{
			AlertID:     alertID,
			UserID:      req.UserId,
			Type:        "LARGE_AMOUNT",
			Description: fmt.Sprintf("Transaction %s is large: %s %s", req.TransactionId, req.Amount, req.Currency),
			Status:      "PENDING",
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
		// 默认低分
		return &pb.GetRiskScoreResponse{
			UserId:    userID,
			Score:     10.0,
			RiskLevel: "LOW",
		}, nil
	}

	return &pb.GetRiskScoreResponse{
		UserId:    score.UserID,
		Score:     score.Score,
		RiskLevel: score.RiskLevel,
	}, nil
}

func (s *AMLService) ListAlerts(ctx context.Context, status string) (*pb.ListAlertsResponse, error) {
	alerts, err := s.repo.ListAlertsByStatus(ctx, status)
	if err != nil {
		return nil, err
	}

	var pbAlerts []*pb.AlertItem
	for _, a := range alerts {
		pbAlerts = append(pbAlerts, &pb.AlertItem{
			AlertId:     a.AlertID,
			UserId:      a.UserID,
			Type:        a.Type,
			Description: a.Description,
			Status:      a.Status,
			CreatedAt:   timestamppb.New(a.CreatedAt),
		})
	}

	return &pb.ListAlertsResponse{Alerts: pbAlerts}, nil
}
