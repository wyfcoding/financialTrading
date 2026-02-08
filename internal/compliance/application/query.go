// Package application 合规服务查询服务
package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/wyfcoding/financialtrading/internal/compliance/domain"
)

// QueryService 合规查询服务
type QueryService struct {
	kycRepo domain.KYCRepository
	logger  *slog.Logger
}

// NewQueryService 创建查询服务
func NewQueryService(
	kycRepo domain.KYCRepository,
	logger *slog.Logger,
) *QueryService {
	return &QueryService{
		kycRepo: kycRepo,
		logger:  logger,
	}
}

// KYCApplicationDTO KYC申请 DTO
type KYCApplicationDTO struct {
	ApplicationID  string
	UserID         uint64
	Level          domain.KYCLevel
	Status         domain.KYCStatus
	FirstName      string
	LastName       string
	IDNumber       string
	DateOfBirth    string
	Country        string
	IDCardFrontURL string
	IDCardBackURL  string
	FacePhotoURL   string
	RejectReason   string
	ReviewerID     string
	ReviewedAt     *time.Time
	CreatedAt      time.Time
}

// GetKYCStatus 获取KYC状态
func (s *QueryService) GetKYCStatus(ctx context.Context, userID uint64) (*KYCApplicationDTO, error) {
	kyc, err := s.kycRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.toDTO(kyc), nil
}

// toDTO 转换 DTO
func (s *QueryService) toDTO(kyc *domain.KYCApplication) *KYCApplicationDTO {
	return &KYCApplicationDTO{
		ApplicationID:  kyc.ApplicationID,
		UserID:         kyc.UserID,
		Level:          kyc.Level,
		Status:         kyc.Status,
		FirstName:      kyc.FirstName,
		LastName:       kyc.LastName,
		IDNumber:       kyc.IDNumber,
		DateOfBirth:    kyc.DateOfBirth,
		Country:        kyc.Country,
		IDCardFrontURL: kyc.IDCardFrontURL,
		IDCardBackURL:  kyc.IDCardBackURL,
		FacePhotoURL:   kyc.FacePhotoURL,
		RejectReason:   kyc.RejectReason,
		ReviewerID:     kyc.ReviewerID,
		ReviewedAt:     kyc.ReviewedAt,
		CreatedAt:      kyc.CreatedAt,
	}
}
