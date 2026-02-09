package interfaces

import (
	"context"

	"github.com/wyfcoding/financialTrading/internal/primebrokerage/application"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// PrimeBrokerageHandler gRPC 处理程序
type PrimeBrokerageHandler struct {
	pb.UnimplementedPrimeBrokerageServiceServer
	appService *application.PrimeBrokerageApplicationService
}

func NewPrimeBrokerageHandler(appService *application.PrimeBrokerageApplicationService) *PrimeBrokerageHandler {
	return &PrimeBrokerageHandler{
		appService: appService,
	}
}

func (h *PrimeBrokerageHandler) RouteToSeat(ctx context.Context, req *pb.RouteToSeatRequest) (*pb.RouteToSeatResponse, error) {
	cmd := application.RouteToSeatCommand{
		Symbol:   req.Symbol,
		Amount:   req.Amount,
		Exchange: req.Exchange,
	}

	seat, err := h.appService.RouteToSeat(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &pb.RouteToSeatResponse{
		SeatId:   seat.ID,
		SeatName: seat.Name,
		Latency:  seat.Latency,
		Cost:     seat.CostPerTrade,
	}, nil
}

func (h *PrimeBrokerageHandler) BorrowSecurity(ctx context.Context, req *pb.BorrowSecurityRequest) (*pb.BorrowSecurityResponse, error) {
	cmd := application.BorrowSecurityCommand{
		UserID:   req.UserId,
		Symbol:   req.Symbol,
		Quantity: req.Quantity,
	}

	loan, err := h.appService.BorrowSecurity(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &pb.BorrowSecurityResponse{
		LoanId: loan.LoanID,
		DueAt:  timestamppb.New(loan.DueAt),
		Rate:   loan.Rate,
	}, nil
}

func (h *PrimeBrokerageHandler) ReturnSecurity(ctx context.Context, req *pb.ReturnSecurityRequest) (*pb.ReturnSecurityResponse, error) {
	cmd := application.ReturnSecurityCommand{
		LoanID: req.LoanId,
	}

	if err := h.appService.ReturnSecurity(ctx, cmd); err != nil {
		return nil, err
	}

	return &pb.ReturnSecurityResponse{
		Success: true,
	}, nil
}
