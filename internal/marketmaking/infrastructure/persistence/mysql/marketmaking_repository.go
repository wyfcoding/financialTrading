package mysql

import (
	"context"
	"errors"

	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
	"gorm.io/gorm"
)

type marketMakingRepository struct {
	db *gorm.DB
}

// NewMarketMakingRepository 创建做市服务仓储实例
func NewMarketMakingRepository(db *gorm.DB) domain.MarketMakingRepository {
	return &marketMakingRepository{db: db}
}

// --- Strategy ---

func (r *marketMakingRepository) SaveStrategy(ctx context.Context, strategy *domain.QuoteStrategy) error {
	return r.db.WithContext(ctx).Save(strategy).Error
}

func (r *marketMakingRepository) GetStrategyBySymbol(ctx context.Context, symbol string) (*domain.QuoteStrategy, error) {
	var strategy domain.QuoteStrategy
	err := r.db.WithContext(ctx).Where("symbol = ?", symbol).First(&strategy).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &strategy, err
}

func (r *marketMakingRepository) ListStrategies(ctx context.Context) ([]*domain.QuoteStrategy, error) {
	var strategies []*domain.QuoteStrategy
	err := r.db.WithContext(ctx).Find(&strategies).Error
	return strategies, err
}

// --- Performance ---

func (r *marketMakingRepository) SavePerformance(ctx context.Context, p *domain.MarketMakingPerformance) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *marketMakingRepository) GetPerformanceBySymbol(ctx context.Context, symbol string) (*domain.MarketMakingPerformance, error) {
	var p domain.MarketMakingPerformance
	err := r.db.WithContext(ctx).Where("symbol = ?", symbol).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}
