package grpc

import (
	"context"
	"time"

	pb "github.com/wyfcoding/financialtrading/go-api/settlement/v1"
	"github.com/wyfcoding/financialtrading/internal/settlement/application"
	"github.com/wyfcoding/financialtrading/internal/settlement/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedSettlementServiceServer
	app *application.SettlementAppService
}

func NewServer(app *application.SettlementAppService) *Server {
	return &Server{app: app}
}

func (s *Server) CreateInstruction(ctx context.Context, req *pb.CreateInstructionRequest) (*pb.CreateInstructionResponse, error) {
	cmd := application.CreateInstructionCommand{
		TradeID:         req.TradeId,
		Symbol:          req.Symbol,
		Quantity:        float64(req.Quantity),
		Price:           req.Price,
		BuyerAccountID:  req.BuyerAccountId,
		SellerAccountID: req.SellerAccountId,
		Currency:        req.Currency,
		CycleDays:       int(req.SettlementCycleDays),
	}

	ins, err := s.app.CreateInstruction(ctx, cmd)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create instruction: %v", err)
	}

	return &pb.CreateInstructionResponse{
		InstructionId:  ins.InstructionID,
		SettlementDate: ins.SettlementDate.Unix(),
	}, nil
}

func (s *Server) GetInstruction(ctx context.Context, req *pb.GetInstructionRequest) (*pb.GetInstructionResponse, error) {
	ins, err := s.app.GetInstruction(ctx, req.InstructionId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "instruction not found: %v", err)
	}

	return &pb.GetInstructionResponse{
		Instruction: toProtoInstruction(ins),
	}, nil
}

func (s *Server) BatchSettle(ctx context.Context, req *pb.BatchSettleRequest) (*pb.BatchSettleResponse, error) {
	targetDate := time.Unix(req.TargetDate, 0)
	result, err := s.app.BatchSettle(ctx, application.BatchSettleCommand{
		SettlementDate: targetDate,
		BatchSize:      1000,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "batch settle failed: %v", err)
	}

	return &pb.BatchSettleResponse{
		ProcessedCount:       int32(result.TotalCount),
		SuccessCount:         int32(result.SuccessCount),
		FailureCount:         int32(result.FailedCount),
		FailedInstructionIds: result.FailedIDs,
	}, nil
}

func toProtoInstruction(ins *domain.SettlementInstruction) *pb.SettlementInstruction {
	quantity, _ := ins.Quantity.Float64()
	price, _ := ins.Price.Float64()
	amount, _ := ins.Amount.Float64()
	return &pb.SettlementInstruction{
		Id:              ins.InstructionID,
		TradeId:         ins.TradeID,
		Symbol:          ins.Symbol,
		Quantity:        int64(quantity),
		Price:           price,
		Amount:          amount,
		BuyerAccountId:  ins.BuyerAccountID,
		SellerAccountId: ins.SellerAccountID,
		TradeDate:       ins.TradeDate.Unix(),
		SettlementDate:  ins.SettlementDate.Unix(),
		Status:          toProtoStatus(ins.Status),
		Currency:        ins.Currency,
	}
}

func toProtoStatus(status domain.SettlementStatus) pb.SettlementStatus {
	switch status {
	case domain.SettlementStatusPending:
		return pb.SettlementStatus_SETTLEMENT_STATUS_PENDING
	case domain.SettlementStatusCleared:
		return pb.SettlementStatus_SETTLEMENT_STATUS_CLEARED
	case domain.SettlementStatusSettled:
		return pb.SettlementStatus_SETTLEMENT_STATUS_SETTLED
	case domain.SettlementStatusFailed:
		return pb.SettlementStatus_SETTLEMENT_STATUS_FAILED
	default:
		return pb.SettlementStatus_SETTLEMENT_STATUS_UNSPECIFIED
	}
}
