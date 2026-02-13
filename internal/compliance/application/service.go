package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wyfcoding/financialtrading/internal/compliance/domain"
)

type ComplianceAppService struct {
	kycRepo domain.KYCRepository
	amlRepo domain.AMLRepository
	logger  *slog.Logger
}

func NewComplianceAppService(kycRepo domain.KYCRepository, amlRepo domain.AMLRepository, logger *slog.Logger) *ComplianceAppService {
	return &ComplianceAppService{
		kycRepo: kycRepo,
		amlRepo: amlRepo,
		logger:  logger,
	}
}

// SubmitKYC 提交KYC申请
func (s *ComplianceAppService) SubmitKYC(ctx context.Context, userID uint64, level domain.KYCLevel, first, last, idNum, dob, country, front, back, face string) (*domain.KYCApplication, error) {
	app := domain.NewKYCApplication(userID, level, first, last, idNum, dob, country, front, back, face)
	app.ApplicationID = fmt.Sprintf("KYC-%d-%d", userID, time.Now().UnixNano())

	if err := s.kycRepo.Save(ctx, app); err != nil {
		return nil, fmt.Errorf("failed to save KYC application: %w", err)
	}

	s.logger.InfoContext(ctx, "KYC application submitted", "app_id", app.ApplicationID, "user_id", userID)
	return app, nil
}

// GetKYCStatus 获取KYC状态
func (s *ComplianceAppService) GetKYCStatus(ctx context.Context, userID uint64) (*domain.KYCApplication, error) {
	return s.kycRepo.GetByUserID(ctx, userID)
}

// ReviewKYC 审核KYC
func (s *ComplianceAppService) ReviewKYC(ctx context.Context, appID string, approved bool, reason, reviewerID string) error {
	app, err := s.kycRepo.GetByApplicationID(ctx, appID)
	if err != nil {
		return err
	}

	if approved {
		if err := app.Approve(reviewerID); err != nil {
			return err
		}
	} else {
		if err := app.Reject(reviewerID, reason); err != nil {
			return err
		}
	}

	if err := s.kycRepo.Save(ctx, app); err != nil {
		return fmt.Errorf("failed to save reviewed KYC: %w", err)
	}

	s.logger.InfoContext(ctx, "KYC reviewed", "app_id", appID, "approved", approved)
	return nil
}

// CheckAML 执行AML检查
func (s *ComplianceAppService) CheckAML(ctx context.Context, userID uint64, name, country string) (bool, string, string, error) {
	// Mock AML Logic: Check against a blacklist (In reality, call External Vendor)
	passed := true
	riskLevel := "LOW"
	reason := ""

	if name == "Terrorist" || country == "North Korea" {
		passed = false
		riskLevel = "CRITICAL"
		reason = "Sanctioned Entity/Country"
	}

	record := &domain.AMLRecord{
		UserID:    userID,
		Name:      name,
		Country:   country,
		Passed:    passed,
		RiskLevel: riskLevel,
		Reason:    reason,
	}

	if err := s.amlRepo.Save(ctx, record); err != nil {
		s.logger.ErrorContext(ctx, "failed to save AML record", "error", err)
	}

	if !passed {
		alert := &domain.AMLAlert{
			AlertID:     fmt.Sprintf("AML-%d-%d", userID, time.Now().UnixNano()),
			UserID:      userID,
			Type:        "SANCTION_MATCH",
			Description: fmt.Sprintf("User %s from %s matched sanctions list", name, country),
			Status:      "PENDING",
		}
		_ = s.amlRepo.SaveAlert(ctx, alert)
	}

	return passed, riskLevel, reason, nil
}

// AssessRisk 风险评估
func (s *ComplianceAppService) AssessRisk(ctx context.Context, userID uint64, amount int64, ip string) (bool, string, string, error) {
	// Mock Risk Logic
	allowed := true
	score := 0.0
	reason := ""

	// Check existing risk score
	riskScore, _ := s.amlRepo.GetRiskScore(ctx, userID)
	if riskScore != nil {
		if riskScore.RiskLevel == "HIGH" {
			allowed = false
			reason = "User is High Risk"
		}
	}

	if amount > 1000000 {
		allowed = false
		reason = "Large transaction requires manual review"
		score += 50
	}

	return allowed, fmt.Sprintf("%.2f", score), reason, nil
}
