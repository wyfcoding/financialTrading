package grpc

import (
	"context"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialtrading/go-api/marginlending/v1"
	"github.com/wyfcoding/financialtrading/internal/margin/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedMarginLendingServiceServer
	app *application.MarginAppService
}

func NewServer(app *application.MarginAppService) *Server {
	return &Server{app: app}
}

func (s *Server) EvaluateMargin(ctx context.Context, req *pb.EvaluateMarginRequest) (*pb.EvaluateMarginResponse, error) {
	userID, err := parseUserID(req.GetUserId())
	if err != nil {
		return nil, err
	}
	quantity, err := parseInt64Field("quantity", req.GetQuantity())
	if err != nil {
		return nil, err
	}
	price, err := parseInt64Field("price", req.GetPrice())
	if err != nil {
		return nil, err
	}

	eligible, reqMargin, leverage, err := s.app.EvaluateMargin(ctx, userID, req.GetSymbol(), quantity, price)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "evaluate failed: %v", err)
	}

	requiredMargin := formatFloat(reqMargin)
	availableLeverage := formatFloat(leverage)

	return &pb.EvaluateMarginResponse{
		Eligible:          eligible,
		RequiredMargin:    requiredMargin,
		AvailableMargin:   "0",
		AvailableLeverage: availableLeverage,
		MaintenanceMargin: requiredMargin,
		MarginRatio:       "0",
	}, nil
}

func (s *Server) LockCollateral(ctx context.Context, req *pb.LockCollateralRequest) (*pb.LockCollateralResponse, error) {
	userID, err := parseUserID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	amountRaw := req.GetQuantity()
	if amountRaw == "" {
		amountRaw = req.GetValue()
	}
	amount, err := parseInt64Field("quantity", amountRaw)
	if err != nil {
		return nil, err
	}

	lockID, success, err := s.app.LockCollateral(ctx, userID, req.GetAsset(), amount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "lock failed: %v", err)
	}

	lockedValue := strconv.FormatInt(amount, 10)
	return &pb.LockCollateralResponse{
		CollateralId: lockID,
		Success:      success,
		LockedValue:  lockedValue,
		HaircutValue: lockedValue,
		Message:      "locked",
	}, nil
}

func (s *Server) MarginCall(ctx context.Context, req *pb.MarginCallRequest) (*pb.MarginCallResponse, error) {
	userID, err := parseUserID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	mm, eq, liq, err := s.app.MarginCall(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "margin call check failed: %v", err)
	}

	marginRatio := "0"
	if mm > 0 {
		marginRatio = formatFloat(eq / mm)
	}

	isMarginCall := eq < mm
	marginCallAmount := "0"
	if isMarginCall {
		marginCallAmount = formatFloat(mm - eq)
	}

	actions := []string{"increase_collateral", "reduce_position"}
	if liq {
		actions = append(actions, "start_liquidation")
	}

	return &pb.MarginCallResponse{
		AccountId:         req.GetAccountId(),
		MaintenanceMargin: formatFloat(mm),
		CurrentEquity:     formatFloat(eq),
		MarginRatio:       marginRatio,
		IsMarginCall:      isMarginCall,
		IsLiquidatable:    liq,
		MarginCallAmount:  marginCallAmount,
		Deadline:          time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		Actions:           actions,
	}, nil
}

func parseUserID(raw string) (uint64, error) {
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, status.Errorf(codes.InvalidArgument, "invalid user_id %q", raw)
	}
	return id, nil
}

func parseInt64Field(name, raw string) (int64, error) {
	if raw == "" {
		return 0, status.Errorf(codes.InvalidArgument, "%s is required", name)
	}

	if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return n, nil
	}

	d, err := decimal.NewFromString(raw)
	if err != nil {
		return 0, status.Errorf(codes.InvalidArgument, "invalid %s %q", name, raw)
	}
	return d.IntPart(), nil
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
