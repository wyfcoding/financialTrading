package interfaces

import (
	"context"

	pb "github.com/wyfcoding/financialtrading/go-api/alternativedata/v1"
	"github.com/wyfcoding/financialtrading/internal/alternativedata/application"
)

type AlternativeDataHandler struct {
	pb.UnimplementedAlternativeDataServiceServer
	app *application.AlternativeDataService
}

func NewAlternativeDataHandler(app *application.AlternativeDataService) *AlternativeDataHandler {
	return &AlternativeDataHandler{app: app}
}

func (h *AlternativeDataHandler) GetSentiment(ctx context.Context, req *pb.GetSentimentRequest) (*pb.GetSentimentResponse, error) {
	return h.app.GetSentiment(ctx, req.Symbol)
}

func (h *AlternativeDataHandler) ListNews(ctx context.Context, req *pb.ListNewsRequest) (*pb.ListNewsResponse, error) {
	return h.app.ListNews(ctx, req.Symbol, req.Limit)
}

func (h *AlternativeDataHandler) IngestData(ctx context.Context, req *pb.IngestDataRequest) (*pb.IngestDataResponse, error) {
	return h.app.IngestData(ctx, req.Type, req.Payload)
}
