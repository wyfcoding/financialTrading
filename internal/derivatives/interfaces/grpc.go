//go:build ignore

package interfaces

import (
	"context"

	pb "github.com/wyfcoding/financialtrading/go-api/derivatives/v1"
	"github.com/wyfcoding/financialtrading/internal/derivatives/application"
	"github.com/wyfcoding/financialtrading/internal/derivatives/domain"
)

type DerivativesHandler struct {
	pb.UnimplementedDerivativesServiceServer
	app  *application.DerivativesService
	repo domain.ContractRepository
}

func NewDerivativesHandler(app *application.DerivativesService, repo domain.ContractRepository) *DerivativesHandler {
	return &DerivativesHandler{app: app, repo: repo}
}

func (h *DerivativesHandler) CreateContract(ctx context.Context, req *pb.CreateContractRequest) (*pb.CreateContractResponse, error) {
	id, err := h.app.CreateContract(ctx, req.Symbol, req.Underlying, req.Type, req.StrikePrice, req.ExpiryDate.AsTime(), req.Multiplier)
	if err != nil {
		return nil, err
	}
	return &pb.CreateContractResponse{ContractId: id}, nil
}

func (h *DerivativesHandler) GetContract(ctx context.Context, req *pb.GetContractRequest) (*pb.GetContractResponse, error) {
	c, err := h.app.GetContract(ctx, req.ContractId)
	if err != nil {
		return nil, err
	}
	return &pb.GetContractResponse{
		Contract: &pb.Contract{
			ContractId:  c.ContractID,
			Symbol:      c.Symbol,
			Underlying:  c.Underlying,
			Type:        string(c.Type),
			StrikePrice: c.StrikePrice.InexactFloat64(),
			ExpiryDate:  c.ExpiryDate.Format("2006-01-02"), // Simplified
			Multiplier:  c.Multiplier.InexactFloat64(),
			Status:      c.Status.String(),
		},
	}, nil
}

func (h *DerivativesHandler) ListContracts(ctx context.Context, req *pb.ListContractsRequest) (*pb.ListContractsResponse, error) {
	list, err := h.app.ListContracts(ctx, req.Underlying, req.ActiveOnly)
	if err != nil {
		return nil, err
	}

	var res []*pb.Contract
	for _, c := range list {
		res = append(res, &pb.Contract{
			ContractId:  c.ContractID,
			Symbol:      c.Symbol,
			Underlying:  c.Underlying,
			Type:        string(c.Type),
			StrikePrice: c.StrikePrice.InexactFloat64(),
			ExpiryDate:  c.ExpiryDate.Format("2006-01-02"),
			Multiplier:  c.Multiplier.InexactFloat64(),
			Status:      c.Status.String(),
		})
	}
	return &pb.ListContractsResponse{Contracts: res}, nil
}

func (h *DerivativesHandler) ExerciseContract(ctx context.Context, req *pb.ExerciseContractRequest) (*pb.ExerciseContractResponse, error) {
	id, pnl, err := h.app.ExerciseContract(ctx, req.ContractId, req.UserId, req.Quantity)
	if err != nil {
		return nil, err
	}
	return &pb.ExerciseContractResponse{Success: true, SettlementId: id, Pnl: pnl}, nil
}
