// Package infrastructure 包含基础设施层实现
package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/wyfcoding/financialTrading/internal/reference-data/domain"
	"github.com/wyfcoding/financialTrading/pkg/logger"
	"gorm.io/gorm"
)

// SymbolModel 交易对数据库模型
// 对应数据库中的 symbols 表
type SymbolModel struct {
	gorm.Model
	BaseCurrency   string  `gorm:"column:base_currency;type:varchar(10);not null;comment:基础货币"`
	QuoteCurrency  string  `gorm:"column:quote_currency;type:varchar(10);not null;comment:计价货币"`
	ExchangeID     string  `gorm:"column:exchange_id;type:varchar(36);not null;index;comment:交易所ID"`
	SymbolCode     string  `gorm:"column:symbol_code;type:varchar(20);uniqueIndex;not null;comment:交易对代码"`
	Status         string  `gorm:"column:status;type:varchar(20);default:'ACTIVE';comment:状态"`
	MinOrderSize   float64 `gorm:"column:min_order_size;type:decimal(20,8);default:0;comment:最小下单数量"`
	PricePrecision float64 `gorm:"column:price_precision;type:decimal(20,8);default:0;comment:价格精度"`
}

// TableName 指定表名
func (SymbolModel) TableName() string {
	return "symbols"
}

// ToDomain 将数据库模型转换为领域实体
func (m *SymbolModel) ToDomain() *domain.Symbol {
	return &domain.Symbol{
		Model:          m.Model,
		ID:             m.SymbolCode, // 使用 SymbolCode 作为 ID
		BaseCurrency:   m.BaseCurrency,
		QuoteCurrency:  m.QuoteCurrency,
		ExchangeID:     m.ExchangeID,
		SymbolCode:     m.SymbolCode,
		Status:         m.Status,
		MinOrderSize:   m.MinOrderSize,
		PricePrecision: m.PricePrecision,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

// ExchangeModel 交易所数据库模型
type ExchangeModel struct {
	gorm.Model
	Name     string `gorm:"column:name;type:varchar(50);uniqueIndex;not null;comment:交易所名称"`
	Country  string `gorm:"column:country;type:varchar(50);comment:国家"`
	Status   string `gorm:"column:status;type:varchar(20);default:'ACTIVE';comment:状态"`
	Timezone string `gorm:"column:timezone;type:varchar(50);comment:时区"`
}

// TableName 指定表名
func (ExchangeModel) TableName() string {
	return "exchanges"
}

// ToDomain 转换为领域实体
func (m *ExchangeModel) ToDomain() *domain.Exchange {
	return &domain.Exchange{
		Model:     m.Model,
		ID:        m.Name, // 使用 Name 作为 ID
		Name:      m.Name,
		Country:   m.Country,
		Status:    m.Status,
		Timezone:  m.Timezone,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

// SymbolRepositoryImpl 交易对仓储实现
type SymbolRepositoryImpl struct {
	db *gorm.DB
}

// NewSymbolRepository 创建交易对仓储实例
func NewSymbolRepository(db *gorm.DB) domain.SymbolRepository {
	return &SymbolRepositoryImpl{db: db}
}

// GetByID 根据 ID 获取交易对
func (r *SymbolRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.Symbol, error) {
	var model SymbolModel
	if err := r.db.WithContext(ctx).Where("symbol_code = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.Error(ctx, "Failed to get symbol by ID",
			"symbol_id", id,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get symbol by ID: %w", err)
	}
	return model.ToDomain(), nil
}

// GetByCode 根据代码获取交易对
func (r *SymbolRepositoryImpl) GetByCode(ctx context.Context, code string) (*domain.Symbol, error) {
	return r.GetByID(ctx, code)
}

// List 列出交易对
func (r *SymbolRepositoryImpl) List(ctx context.Context, exchangeID string, status string, limit int, offset int) ([]*domain.Symbol, error) {
	var models []SymbolModel
	query := r.db.WithContext(ctx).Model(&SymbolModel{})
	if exchangeID != "" {
		query = query.Where("exchange_id = ?", exchangeID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		logger.Error(ctx, "Failed to list symbols",
			"exchange_id", exchangeID,
			"status", status,
			"error", err,
		)
		return nil, fmt.Errorf("failed to list symbols: %w", err)
	}

	result := make([]*domain.Symbol, len(models))
	for i, m := range models {
		result[i] = m.ToDomain()
	}
	return result, nil
}

// ExchangeRepositoryImpl 交易所仓储实现
type ExchangeRepositoryImpl struct {
	db *gorm.DB
}

// NewExchangeRepository 创建交易所仓储实例
func NewExchangeRepository(db *gorm.DB) domain.ExchangeRepository {
	return &ExchangeRepositoryImpl{db: db}
}

// GetByID 根据 ID 获取交易所
func (r *ExchangeRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.Exchange, error) {
	var model ExchangeModel
	// 这里假设 ID 就是 Name
	if err := r.db.WithContext(ctx).Where("name = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.Error(ctx, "Failed to get exchange by ID",
			"exchange_id", id,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get exchange by ID: %w", err)
	}
	return model.ToDomain(), nil
}

// List 列出交易所
func (r *ExchangeRepositoryImpl) List(ctx context.Context, limit int, offset int) ([]*domain.Exchange, error) {
	var models []ExchangeModel
	if err := r.db.WithContext(ctx).Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		logger.Error(ctx, "Failed to list exchanges", "error", err)
		return nil, fmt.Errorf("failed to list exchanges: %w", err)
	}

	result := make([]*domain.Exchange, len(models))
	for i, m := range models {
		result[i] = m.ToDomain()
	}
	return result, nil
}
