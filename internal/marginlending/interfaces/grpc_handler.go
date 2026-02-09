package interfaces

import (
	"context"

	"github.com/wyfcoding/financialTrading/internal/marginlending/application"
)

type MarginLendingHandler struct {
	pb.UnimplementedMarginLendingServiceServer
	appService *application.MarginLendingApplicationService
}

func NewMarginLendingHandler(appService *application.MarginLendingApplicationService) *MarginLendingHandler {
	return &MarginLendingHandler{
		appService: appService,
	}
}

func (h *MarginLendingHandler) EvaluateMargin(ctx context.Context, req *pb.EvaluateMarginRequest) (*pb.EvaluateMarginResponse, error) {
	cmd := application.EvaluateMarginCommand{
		UserID:   req.UserId,
		Symbol:   req.Symbol,
		Quantity: req.Quantity,
		Price:    req.Price,
	}

	reqResult, err := h.appService.EvaluateMargin(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &pb.EvaluateMarginResponse{
		Eligible:       true,
		RequiredMargin: reqResult.InitialMargin,
		// ...
	}, nil
}

func (h *MarginLendingHandler) LockCollateral(ctx context.Context, req *pb.LockCollateralRequest) (*pb.LockCollateralResponse, error) {
	cmd := application.LockCollateralCommand{
		UserID: req.UserId,
		Asset:  req.Asset,
		Amount: req.Amount,
	}

	lockID, err := h.appService.LockCollateral(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &pb.LockCollateralResponse{
		LockId:  lockID,
		Success: true,
	}, nil
}
