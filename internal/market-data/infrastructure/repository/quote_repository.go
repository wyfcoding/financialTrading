package repository

import (
	"context"
	"fmt"

	"github.com/wyfcoding/financialTrading/internal/market-data/domain"
	"github.com/wyfcoding/financialTrading/pkg/db"
	"github.com/wyfcoding/financialTrading/pkg/logger"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// QuoteModel 行情数据数据库模型
type QuoteModel struct {
	gorm.Model
	// 交易对符号
	Symbol string `gorm:"column:symbol;type:varchar(50);index;not null" json:"symbol"`
	// 买价
	BidPrice string `gorm:"column:bid_price;type:decimal(20,8);not null" json:"bid_price"`
	// 卖价
	AskPrice string `gorm:"column:ask_price;type:decimal(20,8);not null" json:"ask_price"`
	// 买量
	BidSize string `gorm:"column:bid_size;type:decimal(20,8);not null" json:"bid_size"`
	// 卖量
	AskSize string `gorm:"column:ask_size;type:decimal(20,8);not null" json:"ask_size"`
	// 最后成交价
	LastPrice string `gorm:"column:last_price;type:decimal(20,8);not null" json:"last_price"`
	// 最后成交量
	LastSize string `gorm:"column:last_size;type:decimal(20,8);not null" json:"last_size"`
	// 时间戳（毫秒）
	Timestamp int64 `gorm:"column:timestamp;type:bigint;index;not null" json:"timestamp"`
	// 数据来源
	Source string `gorm:"column:source;type:varchar(50);not null" json:"source"`
}

// TableName 指定表名
func (QuoteModel) TableName() string {
	return "market_quotes"
}

// QuoteRepositoryImpl 行情数据仓储实现
type QuoteRepositoryImpl struct {
	db *db.DB
}

// NewQuoteRepository 创建行情数据仓储
func NewQuoteRepository(database *db.DB) domain.QuoteRepository {
	return &QuoteRepositoryImpl{
		db: database,
	}
}

// Save 保存行情数据
func (qr *QuoteRepositoryImpl) Save(ctx context.Context, quote *domain.Quote) error {
	model := &QuoteModel{
		Symbol:    quote.Symbol,
		BidPrice:  quote.BidPrice.String(),
		AskPrice:  quote.AskPrice.String(),
		BidSize:   quote.BidSize.String(),
		AskSize:   quote.AskSize.String(),
		LastPrice: quote.LastPrice.String(),
		LastSize:  quote.LastSize.String(),
		Timestamp: quote.Timestamp,
		Source:    quote.Source,
	}

	if err := qr.db.WithContext(ctx).Create(model).Error; err != nil {
		logger.Error(ctx, "Failed to save quote",
			"symbol", quote.Symbol,
			"error", err,
		)
		return fmt.Errorf("failed to save quote: %w", err)
	}

	return nil
}

// GetLatest 获取最新行情
func (qr *QuoteRepositoryImpl) GetLatest(ctx context.Context, symbol string) (*domain.Quote, error) {
	var model QuoteModel

	if err := qr.db.WithContext(ctx).Where("symbol = ?", symbol).Order("timestamp DESC").First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logger.Error(ctx, "Failed to get latest quote",
			"symbol", symbol,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get latest quote: %w", err)
	}

	return qr.modelToDomain(&model), nil
}

// GetHistory 获取历史行情
func (qr *QuoteRepositoryImpl) GetHistory(ctx context.Context, symbol string, startTime, endTime int64) ([]*domain.Quote, error) {
	var models []QuoteModel

	if err := qr.db.WithContext(ctx).Where("symbol = ? AND timestamp >= ? AND timestamp <= ?", symbol, startTime, endTime).
		Order("timestamp DESC").
		Find(&models).Error; err != nil {
		logger.Error(ctx, "Failed to get historical quotes",
			"symbol", symbol,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get historical quotes: %w", err)
	}

	quotes := make([]*domain.Quote, 0, len(models))
	for _, model := range models {
		quotes = append(quotes, qr.modelToDomain(&model))
	}

	return quotes, nil
}

// DeleteExpired 删除过期行情
func (qr *QuoteRepositoryImpl) DeleteExpired(ctx context.Context, beforeTime int64) error {
	if err := qr.db.WithContext(ctx).Where("timestamp < ?", beforeTime).Delete(&QuoteModel{}).Error; err != nil {
		logger.Error(ctx, "Failed to delete expired quotes",
			"before_time", beforeTime,
			"error", err,
		)
		return fmt.Errorf("failed to delete expired quotes: %w", err)
	}

	return nil
}

// modelToDomain 将数据库模型转换为领域对象
func (qr *QuoteRepositoryImpl) modelToDomain(model *QuoteModel) *domain.Quote {
	bidPrice, _ := parseDecimal(model.BidPrice)
	askPrice, _ := parseDecimal(model.AskPrice)
	bidSize, _ := parseDecimal(model.BidSize)
	askSize, _ := parseDecimal(model.AskSize)
	lastPrice, _ := parseDecimal(model.LastPrice)
	lastSize, _ := parseDecimal(model.LastSize)

	return domain.NewQuote(
		model.Symbol,
		bidPrice,
		askPrice,
		bidSize,
		askSize,
		lastPrice,
		lastSize,
		model.Timestamp,
		model.Source,
	)
}

// parseDecimal 解析十进制字符串
func parseDecimal(s string) (decimal.Decimal, error) {
	return decimal.NewFromString(s)
}
