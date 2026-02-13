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
		TradeID:   req.TradeId,
		Symbol:    req.Symbol,
		Quantity:  req.Quantity,
		Price:     req.Price,
		BuyerID:   req.BuyerAccountId,
		SellerID:  req.SellerAccountId,
		Currency:  req.Currency,
		CycleDays: int(req.SettlementCycleDays),
	}

	ins, err := s.app.CreateInstruction(ctx, cmd)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create instruction: %v", err)
	}

	return &pb.CreateInstructionResponse{
		InstructionId:  ins.InstructionID,
		SettlementDate: ins.SettleDate.Unix(),
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
	processed, success, failed, failedIDs, err := s.app.BatchSettle(ctx, targetDate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "batch settle failed: %v", err)
	}

	return &pb.BatchSettleResponse{
		ProcessedCount:       int32(processed),
		SuccessCount:         int32(success),
		FailureCount:         int32(failed),
		FailedInstructionIds: failedIDs,
	}, nil
}

func toProtoInstruction(ins *domain.SettlementInstruction) *pb.SettlementInstruction {
	return &pb.SettlementInstruction{
		Id:              ins.InstructionID,
		TradeId:         ins.TradeID,
		Symbol:          ins.Symbol,
		Quantity:        ins.Quantity,
		Price:           ins.Price,
		Amount:          ins.Amount,
		BuyerAccountId:  ins.BuyerAccount,
		SellerAccountId: ins.SellerAccount,
		TradeDate:       ins.TradeDate.Unix(),
		SettlementDate:  ins.SettleDate.Unix(),
		Status:          pb.SettlementStatus(ins.Status),
		Currency:        ins.Currency,
	}
}
