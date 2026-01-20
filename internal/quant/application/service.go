package application

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/wyfcoding/financialtrading/internal/quant/domain"
)

type QuantApplicationService struct {
	repo  domain.SignalRepository
	logic *domain.IndicatorLogic
	// In the future: Market Data Repository to fetch prices
}

func NewQuantApplicationService(repo domain.SignalRepository) *QuantApplicationService {
	return &QuantApplicationService{
		repo:  repo,
		logic: &domain.IndicatorLogic{},
	}
}

// GetSignal retrieves the latest signal, calculating it if necessary (or simulating it)
func (s *QuantApplicationService) GetSignal(ctx context.Context, symbol string, indicator string, period int) (*SignalDTO, error) {
	// 1. Try Cache
	indType := domain.IndicatorType(indicator)
	signal, err := s.repo.GetLatest(ctx, symbol, indType, period)
	if err == nil {
		return s.toDTO(signal), nil
	}

	// 2. Calculate New
	// Simulation: Generate random value around 50 (RSI) or 100 (SMA)
	// In production: Fetch klines -> s.logic.CalculateXXX -> Save
	val := 0.0
	if indType == domain.RSI {
		val = 30 + rand.Float64()*40 // 30-70 range
	} else {
		val = 100 + rand.Float64()*10
	}

	newSignal := &domain.Signal{
		Symbol:    symbol,
		Indicator: indType,
		Period:    period,
		Value:     val,
		Timestamp: time.Now(),
	}

	// 3. Save
	if err := s.repo.Save(ctx, newSignal); err != nil {
		return nil, fmt.Errorf("failed to save signal: %w", err)
	}

	return s.toDTO(newSignal), nil
}

func (s *QuantApplicationService) toDTO(sig *domain.Signal) *SignalDTO {
	return &SignalDTO{
		Symbol:    sig.Symbol,
		Indicator: string(sig.Indicator),
		Period:    sig.Period,
		Value:     sig.Value,
		Timestamp: sig.Timestamp.Unix(),
	}
}
