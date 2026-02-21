package grpc

import (
	"context"
	"log/slog"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/wyfcoding/financialtrading/go-api/corporateaction/v1"
	"github.com/wyfcoding/financialtrading/internal/corporateaction/application"
	"github.com/wyfcoding/financialtrading/internal/corporateaction/domain"
)

type CorporateActionHandler struct {
	pb.UnimplementedCorporateActionServiceServer
	svc    *application.CorporateActionService
	logger *slog.Logger
}

func NewCorporateActionHandler(svc *application.CorporateActionService, logger *slog.Logger) *CorporateActionHandler {
	return &CorporateActionHandler{
		svc:    svc,
		logger: logger.With("service", "corporateaction_grpc"),
	}
}

func (h *CorporateActionHandler) DeclareAction(ctx context.Context, req *pb.DeclareActionRequest) (*pb.DeclareActionResponse, error) {
	num, _ := decimal.NewFromString(req.Factor) // Simplified

	cmd := application.CreateActionCmd{
		Symbol:           req.Symbol,
		Type:             domain.ActionType(req.ActionType),
		RecordDate:       req.RecordDate.AsTime(),
		PaymentDate:      req.PaymentDate.AsTime(),
		RatioNumerator:   num, // Uses factor as ratio numerator
		RatioDenominator: decimal.NewFromInt(1),
		Currency:         "USD",
	}

	eventID, err := h.svc.AnnounceAction(ctx, cmd)
	if err != nil {
		h.logger.Error("failed to announce action", "error", err)
		return nil, err
	}

	// Automatically calculate entitlements as a simple demo (often triggered via cron instead)
	_ = h.svc.CalculateEntitlements(ctx, eventID)

	return &pb.DeclareActionResponse{
		ActionId: eventID,
	}, nil
}

func (h *CorporateActionHandler) ExecuteAction(ctx context.Context, req *pb.ExecuteActionRequest) (*emptypb.Empty, error) {
	err := h.svc.ProcessPayments(ctx, req.ActionId)
	if err != nil {
		h.logger.Error("failed to process payments", "action_id", req.ActionId, "error", err)
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (h *CorporateActionHandler) ListActionsBySymbol(ctx context.Context, req *pb.ListActionsRequest) (*pb.ListActionsResponse, error) {
	// Not implemented completely yet, normally would call a UseCase
	return &pb.ListActionsResponse{}, nil
}
