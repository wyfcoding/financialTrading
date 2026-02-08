package application

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	pb "github.com/wyfcoding/financialtrading/go-api/alternativedata/v1"
	"github.com/wyfcoding/financialtrading/internal/alternativedata/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AlternativeDataService struct {
	repo domain.AlternativeDataRepository
}

func NewAlternativeDataService(repo domain.AlternativeDataRepository) *AlternativeDataService {
	return &AlternativeDataService{repo: repo}
}

func (s *AlternativeDataService) GetSentiment(ctx context.Context, symbol string) (*pb.GetSentimentResponse, error) {
	sent, err := s.repo.GetLatestSentiment(ctx, symbol)
	if err != nil {
		return nil, err
	}
	if sent == nil {
		return &pb.GetSentimentResponse{
			Symbol:      symbol,
			Score:       0,
			Trend:       "NEUTRAL",
			LastUpdated: timestamppb.Now(),
		}, nil
	}

	return &pb.GetSentimentResponse{
		Symbol:      sent.Symbol,
		Score:       sent.Score,
		Trend:       sent.Trend,
		LastUpdated: timestamppb.New(sent.Timestamp),
	}, nil
}

func (s *AlternativeDataService) ListNews(ctx context.Context, symbol string, limit int32) (*pb.ListNewsResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	news, err := s.repo.ListNews(ctx, symbol, int(limit))
	if err != nil {
		return nil, err
	}

	var pbNews []*pb.NewsItem
	for _, n := range news {
		pbNews = append(pbNews, &pb.NewsItem{
			NewsId:         n.NewsID,
			Title:          n.Title,
			Content:        n.Content,
			Source:         n.Source,
			SentimentScore: n.SentimentScore,
			PublishedAt:    timestamppb.New(n.PublishedAt),
		})
	}

	return &pb.ListNewsResponse{News: pbNews}, nil
}

func (s *AlternativeDataService) IngestData(ctx context.Context, dataType, payload string) (*pb.IngestDataResponse, error) {
	dataID := fmt.Sprintf("data_%d", time.Now().UnixNano())

	switch dataType {
	case "NEWS":
		var n domain.NewsItem
		if err := json.Unmarshal([]byte(payload), &n); err != nil {
			return nil, err
		}
		n.NewsID = dataID
		if n.PublishedAt.IsZero() {
			n.PublishedAt = time.Now()
		}
		if err := s.repo.SaveNews(ctx, &n); err != nil {
			return nil, err
		}
	case "SENTIMENT":
		var sent domain.Sentiment
		if err := json.Unmarshal([]byte(payload), &sent); err != nil {
			return nil, err
		}
		sent.Timestamp = time.Now()
		if err := s.repo.SaveSentiment(ctx, &sent); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported data type %s", dataType)
	}

	return &pb.IngestDataResponse{
		Success: true,
		DataId:  dataID,
	}, nil
}
