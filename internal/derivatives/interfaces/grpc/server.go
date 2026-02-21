package grpc

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	pb "github.com/wyfcoding/financialtrading/go-api/derivatives/v1"
	"github.com/wyfcoding/financialtrading/internal/derivatives/application"
	"github.com/wyfcoding/financialtrading/internal/derivatives/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	pb.UnimplementedDerivativesServiceServer
	app *application.DerivativesAppService
}

func NewServer(app *application.DerivativesAppService) *Server {
	return &Server{app: app}
}

func (s *Server) CreateOptionContract(ctx context.Context, req *pb.CreateOptionContractRequest) (*pb.OptionContract, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if req.ExpiryDate == nil {
		return nil, status.Error(codes.InvalidArgument, "expiry_date is required")
	}

	strike, err := strconv.ParseFloat(req.StrikePrice, 64)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid strike_price: %v", err)
	}

	multiplier, err := strconv.ParseFloat(req.Multiplier, 64)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid multiplier: %v", err)
	}

	contractID, err := s.app.CreateContract(
		ctx,
		req.Symbol,
		req.Underlying,
		optionTypeToDomain(req.OptionType),
		strike,
		req.ExpiryDate.AsTime(),
		multiplier,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create failed: %v", err)
	}

	contract, err := s.app.GetContract(ctx, contractID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load created contract failed: %v", err)
	}

	return toProtoContract(contract), nil
}

func (s *Server) GetOptionContract(ctx context.Context, req *pb.GetOptionContractRequest) (*pb.OptionContract, error) {
	if req == nil || req.ContractId == "" {
		return nil, status.Error(codes.InvalidArgument, "contract_id is required")
	}

	c, err := s.app.GetContract(ctx, req.ContractId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "contract not found: %v", err)
	}

	return toProtoContract(c), nil
}

func (s *Server) ListOptionContracts(ctx context.Context, req *pb.ListOptionContractsRequest) (*pb.ListOptionContractsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	contracts, err := s.app.ListContracts(
		ctx,
		req.Underlying,
		optionTypeToDomain(req.OptionType),
		req.Status == pb.OptionStatus_OPTION_STATUS_ACTIVE,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list failed: %v", err)
	}

	pbContracts := make([]*pb.OptionContract, 0, len(contracts))
	for _, c := range contracts {
		pbContracts = append(pbContracts, toProtoContract(c))
	}

	return &pb.ListOptionContractsResponse{Contracts: pbContracts}, nil
}

func (s *Server) ExerciseOption(ctx context.Context, req *pb.ExerciseOptionRequest) (*pb.ExerciseOptionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.PositionId == "" {
		return nil, status.Error(codes.InvalidArgument, "position_id is required")
	}
	if req.AccountId == "" {
		return nil, status.Error(codes.InvalidArgument, "account_id is required")
	}

	qty, err := strconv.ParseFloat(req.Quantity, 64)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid quantity: %v", err)
	}

	success, settlementID, pnl, err := s.app.ExerciseContract(ctx, req.PositionId, req.AccountId, qty)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "exercise failed: %v", err)
	}
	if !success {
		return nil, status.Error(codes.FailedPrecondition, "exercise rejected")
	}

	now := time.Now()
	value := strconv.FormatFloat(pnl, 'f', -1, 64)
	return &pb.ExerciseOptionResponse{
		ExerciseId:       fmt.Sprintf("EX-%s-%d", req.PositionId, now.UnixNano()),
		SettlementId:     settlementID,
		ExerciseValue:    value,
		SettlementAmount: value,
		ExerciseType:     pb.ExerciseType_EXERCISE_TYPE_CASH,
		ExercisedAt:      timestamppb.New(now),
	}, nil
}

func toProtoContract(c *domain.Contract) *pb.OptionContract {
	if c == nil {
		return nil
	}

	return &pb.OptionContract{
		ContractId:     c.ContractID,
		Symbol:         c.Symbol,
		Underlying:     c.Underlying,
		OptionType:     domainTypeToOptionType(c.Type),
		OptionStyle:    pb.OptionStyle_OPTION_STYLE_EUROPEAN,
		ExerciseType:   pb.ExerciseType_EXERCISE_TYPE_PHYSICAL,
		StrikePrice:    c.StrikePrice.String(),
		ExpiryDate:     timestamppb.New(c.ExpiryDate),
		Multiplier:     c.Multiplier.String(),
		ContractSize:   "1",
		SettlementType: pb.SettlementType_SETTLEMENT_TYPE_T_PLUS_0,
		Status:         domainStatusToOptionStatus(c.Status),
		CreatedAt:      timestamppb.New(c.CreatedAt),
		UpdatedAt:      timestamppb.New(c.UpdatedAt),
	}
}

func optionTypeToDomain(t pb.OptionType) string {
	switch t {
	case pb.OptionType_OPTION_TYPE_CALL:
		return string(domain.TypeCall)
	case pb.OptionType_OPTION_TYPE_PUT:
		return string(domain.TypePut)
	default:
		return string(domain.TypeCall)
	}
}

func domainTypeToOptionType(t domain.ContractType) pb.OptionType {
	switch strings.ToUpper(string(t)) {
	case string(domain.TypePut):
		return pb.OptionType_OPTION_TYPE_PUT
	case string(domain.TypeCall):
		return pb.OptionType_OPTION_TYPE_CALL
	default:
		return pb.OptionType_OPTION_TYPE_UNSPECIFIED
	}
}

func domainStatusToOptionStatus(s domain.ContractStatus) pb.OptionStatus {
	switch s {
	case domain.StatusTrading:
		return pb.OptionStatus_OPTION_STATUS_ACTIVE
	case domain.StatusSettled:
		return pb.OptionStatus_OPTION_STATUS_EXERCISED
	case domain.StatusExpired:
		return pb.OptionStatus_OPTION_STATUS_EXPIRED
	default:
		return pb.OptionStatus_OPTION_STATUS_UNSPECIFIED
	}
}
