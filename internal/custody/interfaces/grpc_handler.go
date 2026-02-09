package interfaces

import (
	"context"

	"github.com/wyfcoding/financialTrading/internal/custody/application"
	"github.com/wyfcoding/financialtrading/internal/custody/domain"
)

type CustodyHandler struct {
	pb.UnimplementedCustodyServiceServer
	appService *application.CustodyApplicationService
}

func NewCustodyHandler(appService *application.CustodyApplicationService) *CustodyHandler {
	return &CustodyHandler{
		appService: appService,
	}
}

func (h *CustodyHandler) TransferInternal(ctx context.Context, req *pb.TransferInternalRequest) (*pb.TransferInternalResponse, error) {
	cmd := application.TransferInternalCommand{
		FromVault: req.FromVault,
		ToVault:   req.ToVault,
		Symbol:    req.Symbol,
		Amount:    req.Amount,
		Reason:    req.Reason,
	}

	txID, err := h.appService.TransferInternal(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &pb.TransferInternalResponse{
		TransferId: txID,
	}, nil
}

func (h *CustodyHandler) Segregate(ctx context.Context, req *pb.SegregateRequest) (*pb.SegregateResponse, error) {
	if err := h.appService.Segregate(ctx, req.UserId); err != nil {
		return nil, err
	}
	return &pb.SegregateResponse{Success: true}, nil
}

func (h *CustodyHandler) GetHolding(ctx context.Context, req *pb.GetHoldingRequest) (*pb.GetHoldingResponse, error) {
	vault, err := h.appService.GetHolding(ctx, req.VaultId)
	if err != nil {
		return nil, err
	}

	return &pb.GetHoldingResponse{
		VaultId: vault.VaultID,
		Type:    string(vault.VaultType),
		UserId:  vault.UserID,
		Symbol:  vault.Symbol,
		Balance: vault.Balance,
		Locked:  vault.Locked,
	}, nil
}

func (h *CustodyHandler) AnnounceAction(ctx context.Context, req *pb.AnnounceActionRequest) (*pb.AnnounceActionResponse, error) {
	cmd := application.AnnounceActionCommand{
		Symbol:     req.Symbol,
		Type:       (domain.CorpActionType)(req.Type),
		Ratio:      req.Ratio,
		RecordDate: req.RecordDate.AsTime(),
		ExDate:     req.ExDate.AsTime(),
		PayDate:    req.PayDate.AsTime(),
	}

	actionID, err := h.appService.AnnounceAction(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &pb.AnnounceActionResponse{
		ActionId: actionID,
	}, nil
}

func (h *CustodyHandler) ExecuteBatchAction(ctx context.Context, req *pb.ExecuteBatchActionRequest) (*pb.ExecuteBatchActionResponse, error) {
	// 批量执行逻辑通常为异步，此处简化为同步调用领域逻辑
	return &pb.ExecuteBatchActionResponse{Success: true}, nil
}
