package grpc

import (
	"context"

	pb "github.com/wyfcoding/financialtrading/go-api/derivatives/v1"
	"github.com/wyfcoding/financialtrading/internal/derivatives/application"
	"github.com/wyfcoding/financialtrading/internal/derivatives/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedDerivativesServiceServer
	app *application.DerivativesAppService
}

func NewServer(app *application.DerivativesAppService) *Server {
	return &Server{app: app}
}

func (s *Server) CreateContract(ctx context.Context, req *pb.CreateContractRequest) (*pb.CreateContractResponse, error) {
	id, err := s.app.CreateContract(ctx,
		req.Symbol,
		req.Underlying,
		req.Type,
		req.StrikePrice,
		req.ExpiryDate.AsTime(),
		req.Multiplier,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create failed: %v", err)
	}
	return &pb.CreateContractResponse{ContractId: id}, nil
}

func (s *Server) GetContract(ctx context.Context, req *pb.GetContractRequest) (*pb.GetContractResponse, error) {
	c, err := s.app.GetContract(ctx, req.ContractId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "contract not found: %v", err)
	}
	return &pb.GetContractResponse{Contract: toProtoContract(c)}, nil
}

func (s *Server) ListContracts(ctx context.Context, req *pb.ListContractsRequest) (*pb.ListContractsResponse, error) {
	contracts, err := s.app.ListContracts(ctx, req.Underlying, req.Type, req.ActiveOnly)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list failed: %v", err)
	}

	var pbContracts []*pb.Contract
	for _, c := range contracts {
		pbContracts = append(pbContracts, toProtoContract(c))
	}
	return &pb.ListContractsResponse{Contracts: pbContracts}, nil
}

func (s *Server) ExerciseContract(ctx context.Context, req *pb.ExerciseContractRequest) (*pb.ExerciseContractResponse, error) {
	success, setID, pnl, err := s.app.ExerciseContract(ctx, req.ContractId, req.UserId, req.Quantity)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "exercise failed: %v", err)
	}
	return &pb.ExerciseContractResponse{
		Success:      success,
		SettlementId: setID,
		Pnl:          pnl,
	}, nil
}

func toProtoContract(c *domain.Contract) *pb.Contract {
	strike, _ := c.StrikePrice.Float64()
	mult, _ := c.Multiplier.Float64()
	return &pb.Contract{
		ContractId:  c.ContractID,
		Symbol:      c.Symbol,
		Underlying:  c.Underlying,
		Type:        string(c.Type),
		StrikePrice: strike,
		ExpiryDate:  c.ExpiryDate.Format("2006-01-02"),
		Multiplier:  mult,
		Status:      c.Status.String(),
	}
}
