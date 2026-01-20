package grpc

import (
	"context"

	v1 "github.com/wyfcoding/financialtrading/go-api/referencedata/v1"
	"github.com/wyfcoding/financialtrading/internal/referencedata/application"
	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	v1.UnimplementedReferenceDataServiceServer
	app *application.ReferenceDataApplicationService
}

func NewServer(s *grpc.Server, app *application.ReferenceDataApplicationService) *Server {
	srv := &Server{app: app}
	v1.RegisterReferenceDataServiceServer(s, srv)
	return srv
}

func (s *Server) GetInstrument(ctx context.Context, req *v1.GetInstrumentRequest) (*v1.GetInstrumentResponse, error) {
	dto, err := s.app.GetInstrument(ctx, req.Symbol)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.GetInstrumentResponse{Instrument: s.toProto(dto)}, nil
}

func (s *Server) ListInstruments(ctx context.Context, req *v1.ListInstrumentsRequest) (*v1.ListInstrumentsResponse, error) {
	dtos, err := s.app.ListInstruments(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	var instruments []*v1.Instrument
	for _, d := range dtos {
		instruments = append(instruments, s.toProto(d))
	}
	return &v1.ListInstrumentsResponse{Instruments: instruments}, nil
}

func (s *Server) toProto(d *application.InstrumentDTO) *v1.Instrument {
	return &v1.Instrument{
		Symbol:        d.Symbol,
		BaseCurrency:  d.BaseCurrency,
		QuoteCurrency: d.QuoteCurrency,
		TickSize:      d.TickSize,
		LotSize:       d.LotSize,
		Type:          s.mapType(d.Type),
		MaxLeverage:   int32(d.MaxLeverage),
	}
}

func (s *Server) mapType(t string) v1.InstrumentType {
	switch t {
	case string(domain.Spot):
		return v1.InstrumentType_SPOT
	case string(domain.Future):
		return v1.InstrumentType_FUTURE
	case string(domain.Option):
		return v1.InstrumentType_OPTION
	default:
		return v1.InstrumentType_INSTRUMENT_TYPE_UNSPECIFIED
	}
}
