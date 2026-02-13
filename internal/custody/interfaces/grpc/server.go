package grpc

import (
	"context"

	pb "github.com/wyfcoding/financialtrading/go-api/custody/v1"
	"github.com/wyfcoding/financialtrading/internal/custody/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedCustodyServiceServer
	app *application.CustodyAppService
}

func NewServer(app *application.CustodyAppService) *Server {
	return &Server{app: app}
}

func (s *Server) TransferInternal(ctx context.Context, req *pb.TransferInternalRequest) (*pb.TransferInternalResponse, error) {
	txID, err := s.app.TransferInternal(ctx, req.FromVault, req.ToVault, req.Amount, req.Reason)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "transfer failed: %v", err)
	}
	return &pb.TransferInternalResponse{TransferId: txID}, nil
}

func (s *Server) Segregate(ctx context.Context, req *pb.SegregateRequest) (*pb.SegregateResponse, error) {
	err := s.app.Segregate(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "segregation failed: %v", err)
	}
	return &pb.SegregateResponse{Success: true}, nil
}

func (s *Server) GetHolding(ctx context.Context, req *pb.GetHoldingRequest) (*pb.GetHoldingResponse, error) {
	vault, err := s.app.GetHolding(ctx, req.VaultId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "vault not found: %v", err)
	}
	return &pb.GetHoldingResponse{
		VaultId: vault.VaultID,
		Type:    string(vault.Type),
		UserId:  vault.UserID,
		Symbol:  vault.Symbol,
		Balance: vault.Balance,
		Locked:  vault.Locked,
	}, nil
}

func (s *Server) AnnounceAction(ctx context.Context, req *pb.AnnounceActionRequest) (*pb.AnnounceActionResponse, error) {
	id, err := s.app.AnnounceAction(ctx, req.Symbol, req.Type, req.Ratio, req.RecordDate.AsTime(), req.ExDate.AsTime(), req.PayDate.AsTime())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "announce failed: %v", err)
	}
	return &pb.AnnounceActionResponse{ActionId: id}, nil
}

func (s *Server) ExecuteBatchAction(ctx context.Context, req *pb.ExecuteBatchActionRequest) (*pb.ExecuteBatchActionResponse, error) {
	err := s.app.ExecuteBatchAction(ctx, req.ActionId)
	if err != nil {
		return &pb.ExecuteBatchActionResponse{Success: false}, err
	}
	return &pb.ExecuteBatchActionResponse{Success: true}, nil
}
