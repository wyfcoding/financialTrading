// Package application 合规服务应用层
package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wyfcoding/financialtrading/internal/compliance/domain"
	"github.com/wyfcoding/pkg/messagequeue"
)

// CommandService 合规命令服务
type CommandService struct {
	kycRepo        domain.KYCRepository
	amlRepo        domain.AMLRepository
	eventPublisher messagequeue.EventPublisher
	logger         *slog.Logger
}

// NewCommandService 创建命令服务
func NewCommandService(
	kycRepo domain.KYCRepository,
	amlRepo domain.AMLRepository,
	eventPublisher messagequeue.EventPublisher,
	logger *slog.Logger,
) *CommandService {
	return &CommandService{
		kycRepo:        kycRepo,
		amlRepo:        amlRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// SubmitKYCCommand 提交KYC命令
type SubmitKYCCommand struct {
	UserID         uint64
	Level          domain.KYCLevel
	FirstName      string
	LastName       string
	IDNumber       string
	DateOfBirth    string
	Country        string
	IDCardFrontURL string
	IDCardBackURL  string
	FacePhotoURL   string
}

// SubmitKYC 提交KYC申请
func (s *CommandService) SubmitKYC(ctx context.Context, cmd SubmitKYCCommand) (string, error) {
	start := time.Now()

	// 检查是否已有申请
	existing, err := s.kycRepo.GetByUserID(ctx, cmd.UserID)
	if err == nil && existing != nil && existing.Status == domain.KYCStatusPending {
		return "", fmt.Errorf("pending application exists: %s", existing.ApplicationID)
	}

	kyc := domain.NewKYCApplication(
		cmd.UserID,
		cmd.Level,
		cmd.FirstName,
		cmd.LastName,
		cmd.IDNumber,
		cmd.DateOfBirth,
		cmd.Country,
		cmd.IDCardFrontURL,
		cmd.IDCardBackURL,
		cmd.FacePhotoURL,
	)

	now := time.Now()
	kyc.ApplicationID = fmt.Sprintf("KYC%s%04d", now.Format("20060102150405"), now.UnixNano()%10000)

	if err := s.kycRepo.Save(ctx, kyc); err != nil {
		s.logger.ErrorContext(ctx, "failed to save kyc application",
			"user_id", cmd.UserID,
			"error", err,
			"duration", time.Since(start))
		return "", err
	}

	s.logger.InfoContext(ctx, "kyc submitted",
		"application_id", kyc.ApplicationID,
		"user_id", cmd.UserID,
		"duration", time.Since(start))

	return kyc.ApplicationID, nil
}

// ReviewKYCCommand 审核KYC命令
type ReviewKYCCommand struct {
	ApplicationID string
	Approved      bool
	RejectReason  string
	ReviewerID    string
}

// ReviewKYC 审核KYC
func (s *CommandService) ReviewKYC(ctx context.Context, cmd ReviewKYCCommand) error {
	kyc, err := s.kycRepo.GetByApplicationID(ctx, cmd.ApplicationID)
	if err != nil {
		return err
	}

	if cmd.Approved {
		if err := kyc.Approve(cmd.ReviewerID); err != nil {
			return err
		}
	} else {
		if err := kyc.Reject(cmd.ReviewerID, cmd.RejectReason); err != nil {
			return err
		}
	}

	if err := s.kycRepo.Save(ctx, kyc); err != nil {
		return err
	}

	s.publishEvents(ctx, kyc.GetDomainEvents())
	kyc.ClearDomainEvents()

	s.logger.InfoContext(ctx, "kyc reviewed",
		"application_id", cmd.ApplicationID,
		"approved", cmd.Approved,
		"reviewer_id", cmd.ReviewerID)

	return nil
}

// CheckAMLCommand AML检查命令
type CheckAMLCommand struct {
	UserID  uint64
	Name    string
	Country string
}

// CheckAMLResult AML检查结果
type CheckAMLResult struct {
	Passed    bool
	RiskLevel string
	Reason    string
}

// CheckAML 执行AML检查
func (s *CommandService) CheckAML(ctx context.Context, cmd CheckAMLCommand) (*CheckAMLResult, error) {
	// 模拟检查逻辑
	passed := true
	riskLevel := "LOW"
	reason := "Automatic check passed"

	// 简单模拟：如果名字包含 "sanction"，则拒绝
	if cmd.Name == "sanction" {
		passed = false
		riskLevel = "HIGH"
		reason = "Name matched in sanction list"
	}

	record := &domain.AMLRecord{
		UserID:    cmd.UserID,
		Name:      cmd.Name,
		Country:   cmd.Country,
		Passed:    passed,
		RiskLevel: riskLevel,
		Reason:    reason,
	}

	if err := s.amlRepo.Save(ctx, record); err != nil {
		return nil, err
	}

	return &CheckAMLResult{
		Passed:    passed,
		RiskLevel: riskLevel,
		Reason:    reason,
	}, nil
}

// publishEvents 发布领域事件
func (s *CommandService) publishEvents(ctx context.Context, events []domain.DomainEvent) {
	for _, event := range events {
		if err := s.eventPublisher.Publish(ctx, event.EventName(), "", event); err != nil {
			s.logger.ErrorContext(ctx, "failed to publish event",
				"event", event.EventName(),
				"error", err)
		}
	}
}
