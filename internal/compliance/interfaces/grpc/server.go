package grpc

import (
	"context"

	pb "github.com/wyfcoding/financialtrading/go-api/compliance/v1"
	"github.com/wyfcoding/financialtrading/internal/compliance/application"
	"github.com/wyfcoding/financialtrading/internal/compliance/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	pb.UnimplementedComplianceServiceServer
	app *application.ComplianceAppService
}

func NewServer(app *application.ComplianceAppService) *Server {
	return &Server{app: app}
}

func (s *Server) SubmitKYC(ctx context.Context, req *pb.SubmitKYCRequest) (*pb.SubmitKYCResponse, error) {
	app, err := s.app.SubmitKYC(ctx,
		req.UserId,
		domain.KYCLevel(req.Level),
		req.FirstName,
		req.LastName,
		req.IdNumber,
		req.DateOfBirth,
		req.Country,
		req.IdCardFrontUrl,
		req.IdCardBackUrl,
		req.FacePhotoUrl,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to submit KYC: %v", err)
	}

	return &pb.SubmitKYCResponse{
		ApplicationId: app.ApplicationID,
	}, nil
}

func (s *Server) GetKYCStatus(ctx context.Context, req *pb.GetKYCStatusRequest) (*pb.GetKYCStatusResponse, error) {
	app, err := s.app.GetKYCStatus(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "KYC not found: %v", err)
	}

	return &pb.GetKYCStatusResponse{
		Application: toProtoKYC(app),
	}, nil
}

func (s *Server) ReviewKYC(ctx context.Context, req *pb.ReviewKYCRequest) (*pb.ReviewKYCResponse, error) {
	err := s.app.ReviewKYC(ctx, req.ApplicationId, req.Approved, req.RejectReason, req.ReviewerId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to review KYC: %v", err)
	}
	return &pb.ReviewKYCResponse{Success: true}, nil
}

func (s *Server) CheckAML(ctx context.Context, req *pb.CheckAMLRequest) (*pb.CheckAMLResponse, error) {
	passed, riskLevel, reason, err := s.app.CheckAML(ctx, req.UserId, req.Name, req.Country)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "AML check failed: %v", err)
	}
	return &pb.CheckAMLResponse{
		Passed:    passed,
		RiskLevel: riskLevel,
		Reason:    reason,
	}, nil
}

func (s *Server) AssessRisk(ctx context.Context, req *pb.AssessRiskRequest) (*pb.AssessRiskResponse, error) {
	allowed, score, reason, err := s.app.AssessRisk(ctx, req.UserId, req.Amount, req.IpAddress)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "risk assessment failed: %v", err)
	}
	return &pb.AssessRiskResponse{
		Allowed:   allowed,
		RiskScore: score,
		Reason:    reason,
	}, nil
}

func toProtoKYC(app *domain.KYCApplication) *pb.KYCApplication {
	if app == nil {
		return nil
	}
	pbApp := &pb.KYCApplication{
		ApplicationId:  app.ApplicationID,
		UserId:         app.UserID,
		Level:          pb.KYCLevel(app.Level),
		Status:         pb.KYCStatus(app.Status),
		FirstName:      app.FirstName,
		LastName:       app.LastName,
		IdNumber:       app.IDNumber,
		DateOfBirth:    app.DateOfBirth,
		Country:        app.Country,
		IdCardFrontUrl: app.IDCardFrontURL,
		IdCardBackUrl:  app.IDCardBackURL,
		FacePhotoUrl:   app.FacePhotoURL,
		RejectReason:   app.RejectReason,
		ReviewerId:     app.ReviewerID,
		CreatedAt:      timestamppb.New(app.CreatedAt),
		UpdatedAt:      timestamppb.New(app.UpdatedAt),
	}
	if app.ReviewedAt != nil {
		pbApp.ReviewedAt = timestamppb.New(*app.ReviewedAt)
	}
	return pbApp
}
