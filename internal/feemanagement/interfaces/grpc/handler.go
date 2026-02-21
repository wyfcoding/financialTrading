package grpc

import (
	"context"
	"log/slog"

	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/wyfcoding/financialtrading/go-api/feemanagement/v1"
	"github.com/wyfcoding/financialtrading/internal/feemanagement/application"
	"github.com/wyfcoding/financialtrading/internal/feemanagement/domain"
)

type FeeHandler struct {
	pb.UnimplementedFeeManagementServiceServer
	svc    *application.FeeService
	logger *slog.Logger
}

func NewFeeHandler(svc *application.FeeService, logger *slog.Logger) *FeeHandler {
	return &FeeHandler{
		svc:    svc,
		logger: logger.With("service", "feemanagement_grpc"),
	}
}

func (h *FeeHandler) CreateFeeSchedule(ctx context.Context, req *pb.CreateFeeScheduleRequest) (*pb.FeeSchedule, error) {
	schedule, err := h.svc.CreateSchedule(ctx, req)
	if err != nil {
		h.logger.Error("failed to create fee schedule", "error", err)
		return nil, err
	}
	return convertScheduleToProto(schedule), nil
}

func (h *FeeHandler) EstimateFees(ctx context.Context, req *pb.EstimateFeesRequest) (*pb.FeeEstimationResponse, error) {
	amount := req.Price * req.Quantity
	// Note: We use "EQUITY" as default asset class for simple estimation if not provided in request
	record, err := h.svc.EstimateFees(ctx, req.UserId, req.Symbol, "EQUITY", amount)
	if err != nil {
		h.logger.Error("failed to estimate fees", "error", err)
		return nil, err
	}

	components := make([]*pb.FeeBreakdown, 0, len(record.Components))
	for _, c := range record.Components {
		components = append(components, &pb.FeeBreakdown{
			Type:        pb.FeeType(c.Type),
			Amount:      c.Amount,
			Currency:    c.Currency,
			Description: c.Description,
		})
	}

	return &pb.FeeEstimationResponse{
		TotalEstimatedFee: record.TotalFee,
		Components:        components,
	}, nil
}

func (h *FeeHandler) CalculateTradeFees(ctx context.Context, req *pb.CalculateTradeFeesRequest) (*pb.TradeFeesResponse, error) {
	amount := req.Price * req.Quantity
	record, err := h.svc.CalculateTradeFees(ctx, req.TradeId, req.OrderId, req.UserId, req.Symbol, "EQUITY", amount)
	if err != nil {
		h.logger.Error("failed to calculate trade fees", "error", err)
		return nil, err
	}

	components := make([]*pb.FeeBreakdown, 0, len(record.Components))
	for _, c := range record.Components {
		components = append(components, &pb.FeeBreakdown{
			Type:        pb.FeeType(c.Type),
			Amount:      c.Amount,
			Currency:    c.Currency,
			Description: c.Description,
		})
	}

	return &pb.TradeFeesResponse{
		TradeId:    record.TradeID,
		TotalFee:   record.TotalFee,
		Components: components,
	}, nil
}

func (h *FeeHandler) GetFeeSchedule(ctx context.Context, req *pb.GetFeeScheduleRequest) (*pb.FeeSchedule, error) {
	schedule, err := h.svc.GetFeeSchedule(ctx, req.Id)
	if err != nil {
		h.logger.Error("failed to get fee schedule", "error", err)
		return nil, err
	}
	return convertScheduleToProto(schedule), nil
}

func (h *FeeHandler) ListFeeSchedules(ctx context.Context, req *pb.ListFeeSchedulesRequest) (*pb.ListFeeSchedulesResponse, error) {
	schedules, err := h.svc.ListSchedules(ctx, req.UserTier, req.AssetClass)
	if err != nil {
		h.logger.Error("failed to list fee schedules", "error", err)
		return nil, err
	}

	pbSchedules := make([]*pb.FeeSchedule, 0, len(schedules))
	for _, s := range schedules {
		pbSchedules = append(pbSchedules, convertScheduleToProto(s))
	}

	return &pb.ListFeeSchedulesResponse{
		Schedules: pbSchedules,
	}, nil
}

func (h *FeeHandler) CalculateRebate(ctx context.Context, req *pb.CalculateRebateRequest) (*pb.RebateResponse, error) {
	rebate, err := h.svc.CalculateRebate(ctx, req.UserId, req.StartTime.AsTime(), req.EndTime.AsTime())
	if err != nil {
		h.logger.Error("failed to calculate rebate", "error", err)
		return nil, err
	}

	return &pb.RebateResponse{
		UserId:      req.UserId,
		TotalRebate: rebate,
		Currency:    "USD",
	}, nil
}

func convertScheduleToProto(s *domain.FeeSchedule) *pb.FeeSchedule {
	return &pb.FeeSchedule{
		Id:         s.ID,
		Name:       s.Name,
		UserTier:   s.UserTier,
		AssetClass: s.AssetClass,
		BaseRate:   s.BaseRate,
		MinFee:     s.MinFee,
		MaxFee:     s.MaxFee,
		CreatedAt:  timestamppb.New(s.CreatedAt),
	}
}
