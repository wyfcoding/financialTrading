package application

import (
	"context"
	"log/slog"

	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
)

// ReferenceDataProjectionService 将写模型投影到读模型（Redis/ES）。
type ReferenceDataProjectionService struct {
	repo               domain.ReferenceDataRepository
	symbolReadRepo     domain.SymbolReadRepository
	exchangeReadRepo   domain.ExchangeReadRepository
	instrumentReadRepo domain.InstrumentReadRepository
	searchRepo         domain.ReferenceDataSearchRepository
	logger             *slog.Logger
}

func NewReferenceDataProjectionService(
	repo domain.ReferenceDataRepository,
	symbolReadRepo domain.SymbolReadRepository,
	exchangeReadRepo domain.ExchangeReadRepository,
	instrumentReadRepo domain.InstrumentReadRepository,
	searchRepo domain.ReferenceDataSearchRepository,
	logger *slog.Logger,
) *ReferenceDataProjectionService {
	return &ReferenceDataProjectionService{
		repo:               repo,
		symbolReadRepo:     symbolReadRepo,
		exchangeReadRepo:   exchangeReadRepo,
		instrumentReadRepo: instrumentReadRepo,
		searchRepo:         searchRepo,
		logger:             logger,
	}
}

func (s *ReferenceDataProjectionService) RefreshSymbol(ctx context.Context, idOrCode string, syncSearch bool) error {
	if idOrCode == "" {
		return nil
	}
	symbol, err := s.repo.GetSymbol(ctx, idOrCode)
	if err != nil || symbol == nil {
		if err != nil {
			return err
		}
		symbol, err = s.repo.GetSymbolByCode(ctx, idOrCode)
		if err != nil || symbol == nil {
			return err
		}
	}
	if s.symbolReadRepo != nil {
		if err := s.symbolReadRepo.Save(ctx, symbol); err != nil {
			s.logger.WarnContext(ctx, "failed to update symbol cache", "error", err, "symbol", idOrCode)
		}
	}
	if syncSearch && s.searchRepo != nil {
		if err := s.searchRepo.IndexSymbol(ctx, symbol); err != nil {
			s.logger.WarnContext(ctx, "failed to index symbol", "error", err, "symbol", idOrCode)
			return err
		}
	}
	return nil
}

func (s *ReferenceDataProjectionService) RefreshExchange(ctx context.Context, idOrName string, syncSearch bool) error {
	if idOrName == "" {
		return nil
	}
	exchange, err := s.repo.GetExchange(ctx, idOrName)
	if err != nil || exchange == nil {
		if err != nil {
			return err
		}
		exchange, err = s.repo.GetExchangeByName(ctx, idOrName)
		if err != nil || exchange == nil {
			return err
		}
	}
	if s.exchangeReadRepo != nil {
		if err := s.exchangeReadRepo.Save(ctx, exchange); err != nil {
			s.logger.WarnContext(ctx, "failed to update exchange cache", "error", err, "exchange", idOrName)
		}
	}
	if syncSearch && s.searchRepo != nil {
		if err := s.searchRepo.IndexExchange(ctx, exchange); err != nil {
			s.logger.WarnContext(ctx, "failed to index exchange", "error", err, "exchange", idOrName)
			return err
		}
	}
	return nil
}

func (s *ReferenceDataProjectionService) RefreshInstrument(ctx context.Context, symbol string, syncSearch bool) error {
	if symbol == "" {
		return nil
	}
	instrument, err := s.repo.GetInstrument(ctx, symbol)
	if err != nil || instrument == nil {
		return err
	}
	if s.instrumentReadRepo != nil {
		if err := s.instrumentReadRepo.Save(ctx, instrument); err != nil {
			s.logger.WarnContext(ctx, "failed to update instrument cache", "error", err, "symbol", symbol)
		}
	}
	if syncSearch && s.searchRepo != nil {
		if err := s.searchRepo.IndexInstrument(ctx, instrument); err != nil {
			s.logger.WarnContext(ctx, "failed to index instrument", "error", err, "symbol", symbol)
			return err
		}
	}
	return nil
}
