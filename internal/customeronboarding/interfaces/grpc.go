package interfaces

import (
	"context"

	pb "github.com/wyfcoding/financialtrading/go-api/customeronboarding/v1"
	"github.com/wyfcoding/financialtrading/internal/customeronboarding/application"
)

type OnboardingHandler struct {
	pb.UnimplementedCustomerOnboardingServiceServer
	app *application.OnboardingService
}

func NewOnboardingHandler(app *application.OnboardingService) *OnboardingHandler {
	return &OnboardingHandler{app: app}
}

func (h *OnboardingHandler) SubmitApplication(ctx context.Context, req *pb.SubmitApplicationRequest) (*pb.SubmitApplicationResponse, error) {
	return h.app.SubmitApplication(ctx, req)
}

func (h *OnboardingHandler) GetApplication(ctx context.Context, req *pb.GetApplicationRequest) (*pb.GetApplicationResponse, error) {
	return h.app.GetApplication(ctx, req.ApplicationId)
}

func (h *OnboardingHandler) UploadDocument(ctx context.Context, req *pb.UploadDocumentRequest) (*pb.UploadDocumentResponse, error) {
	return h.app.UploadDocument(ctx, req.ApplicationId, req.DocType, req.FileUrl)
}
