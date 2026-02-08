package application

import (
	"context"
	"fmt"
	"time"

	pb "github.com/wyfcoding/financialtrading/go-api/customeronboarding/v1"
	"github.com/wyfcoding/financialtrading/internal/customeronboarding/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OnboardingService struct {
	repo domain.OnboardingRepository
}

func NewOnboardingService(repo domain.OnboardingRepository) *OnboardingService {
	return &OnboardingService{repo: repo}
}

func (s *OnboardingService) SubmitApplication(ctx context.Context, req *pb.SubmitApplicationRequest) (*pb.SubmitApplicationResponse, error) {
	appID := fmt.Sprintf("app_%d", time.Now().UnixNano())

	app := &domain.OnboardingApplication{
		ApplicationID: appID,
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		Email:         req.Email,
		IDNumber:      req.IdNumber,
		Address:       req.Address,
		Status:        domain.StatusPending,
		KYCStatus:     "PENDING",
	}

	if err := s.repo.Save(ctx, app); err != nil {
		return nil, err
	}

	return &pb.SubmitApplicationResponse{
		ApplicationId: appID,
		Status:        string(domain.StatusPending),
	}, nil
}

func (s *OnboardingService) GetApplication(ctx context.Context, appID string) (*pb.GetApplicationResponse, error) {
	app, err := s.repo.Get(ctx, appID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, fmt.Errorf("application %s not found", appID)
	}

	return &pb.GetApplicationResponse{
		ApplicationId: app.ApplicationID,
		Status:        string(app.Status),
		KycStatus:     app.KYCStatus,
		SubmittedAt:   timestamppb.New(app.CreatedAt),
	}, nil
}

func (s *OnboardingService) UploadDocument(ctx context.Context, appID, docType, fileURL string) (*pb.UploadDocumentResponse, error) {
	app, err := s.repo.Get(ctx, appID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, fmt.Errorf("application %s not found", appID)
	}

	// 模拟处理：更新 KYC 状态
	app.KYCStatus = "VERIFYING"
	app.Status = domain.StatusProcessing

	if err := s.repo.Save(ctx, app); err != nil {
		return nil, err
	}

	return &pb.UploadDocumentResponse{Success: true}, nil
}
