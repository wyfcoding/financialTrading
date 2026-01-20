package grpc

import (
	"context"

	v1 "github.com/wyfcoding/financialtrading/go-api/quant/v1"
	"github.com/wyfcoding/financialtrading/internal/quant/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	v1.UnimplementedQuantServiceServer
	app *application.QuantApplicationService
}

func NewServer(s *grpc.Server, app *application.QuantApplicationService) *Server {
	srv := &Server{app: app}
	v1.RegisterQuantServiceServer(s, srv)
	return srv
}

func (s *Server) GetSignal(ctx context.Context, req *v1.GetSignalRequest) (*v1.GetSignalResponse, error) {
	dto, err := s.app.GetSignal(ctx, req.Symbol, s.mapIndicator(req.Indicator), int(req.Period))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.GetSignalResponse{
		Signal: &v1.Signal{
			Symbol:    dto.Symbol,
			Indicator: req.Indicator,
			Value:     dto.Value,
			Period:    int32(dto.Period),
		},
	}, nil
}

func (s *Server) mapIndicator(i v1.IndicatorType) string {
	if i == v1.IndicatorType_RSI {
		return "RSI"
	}
	if i == v1.IndicatorType_SMA {
		return "SMA"
	}
	if i == v1.IndicatorType_EMA {
		return "EMA"
	}
	return "UNKNOWN"
}
