package infrastructure

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/limit/domain"
	"gorm.io/gorm"
)

type LimitPO struct {
	ID               uint64          `gorm:"column:id;primaryKey;autoIncrement"`
	LimitID          string          `gorm:"column:limit_id;type:varchar(32);uniqueIndex;not null"`
	AccountID        uint64          `gorm:"column:account_id;index;not null"`
	Type             string          `gorm:"column:type;type:varchar(20);not null"`
	Scope            string          `gorm:"column:scope;type:varchar(20);not null"`
	Symbol           string          `gorm:"column:symbol;type:varchar(20)"`
	CurrentValue     decimal.Decimal `gorm:"column:current_value;type:decimal(20,4);not null"`
	LimitValue       decimal.Decimal `gorm:"column:limit_value;type:decimal(20,4);not null"`
	WarningThreshold decimal.Decimal `gorm:"column:warning_threshold;type:decimal(10,4);not null"`
	UsedPercent      decimal.Decimal `gorm:"column:used_percent;type:decimal(10,4);not null"`
	Status           string          `gorm:"column:status;type:varchar(20);not null;default:'ACTIVE'"`
	IsActive         bool            `gorm:"column:is_active;default:true"`
	StartTime        *time.Time      `gorm:"column:start_time"`
	EndTime          *time.Time      `gorm:"column:end_time"`
	ResetAt          *time.Time      `gorm:"column:reset_at"`
	CreatedAt        time.Time       `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time       `gorm:"column:updated_at;autoUpdateTime"`
}

func (LimitPO) TableName() string { return "account_limits" }

type LimitBreachPO struct {
	ID            uint64          `gorm:"column:id;primaryKey;autoIncrement"`
	BreachID      string          `gorm:"column:breach_id;type:varchar(32);uniqueIndex;not null"`
	LimitID       string          `gorm:"column:limit_id;type:varchar(32);index;not null"`
	AccountID     uint64          `gorm:"column:account_id;index;not null"`
	Type          string          `gorm:"column:type;type:varchar(20);not null"`
	CurrentValue  decimal.Decimal `gorm:"column:current_value;type:decimal(20,4);not null"`
	LimitValue    decimal.Decimal `gorm:"column:limit_value;type:decimal(20,4);not null"`
	BreachPercent decimal.Decimal `gorm:"column:breach_percent;type:decimal(10,4);not null"`
	Action        string          `gorm:"column:action;type:varchar(20);not null"`
	ResolvedAt    *time.Time      `gorm:"column:resolved_at"`
	CreatedAt     time.Time       `gorm:"column:created_at;autoCreateTime"`
}

func (LimitBreachPO) TableName() string { return "limit_breaches" }

type AccountLimitConfigPO struct {
	ID                uint64          `gorm:"column:id;primaryKey;autoIncrement"`
	AccountID         uint64          `gorm:"column:account_id;uniqueIndex;not null"`
	MaxPositionSize   decimal.Decimal `gorm:"column:max_position_size;type:decimal(20,4);not null"`
	MaxOrderSize      decimal.Decimal `gorm:"column:max_order_size;type:decimal(20,4);not null"`
	MaxDailyVolume    decimal.Decimal `gorm:"column:max_daily_volume;type:decimal(20,4);not null"`
	MaxDailyLoss      decimal.Decimal `gorm:"column:max_daily_loss;type:decimal(20,4);not null"`
	MaxExposure       decimal.Decimal `gorm:"column:max_exposure;type:decimal(20,4);not null"`
	MaxLeverage       decimal.Decimal `gorm:"column:max_leverage;type:decimal(10,4);not null"`
	MaxConcentration  decimal.Decimal `gorm:"column:max_concentration;type:decimal(10,4);not null"`
	WarningThreshold  decimal.Decimal `gorm:"column:warning_threshold;type:decimal(10,4);not null"`
	AutoFreezeEnabled bool            `gorm:"column:auto_freeze_enabled;default:true"`
	CreatedAt         time.Time       `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time       `gorm:"column:updated_at;autoUpdateTime"`
}

func (AccountLimitConfigPO) TableName() string { return "account_limit_configs" }

type GormLimitRepository struct {
	db *gorm.DB
}

func NewGormLimitRepository(db *gorm.DB) *GormLimitRepository {
	return &GormLimitRepository{db: db}
}

func (r *GormLimitRepository) Save(ctx context.Context, l *domain.Limit) error {
	po := toLimitPO(l)
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *GormLimitRepository) Update(ctx context.Context, l *domain.Limit) error {
	po := toLimitPO(l)
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *GormLimitRepository) GetByID(ctx context.Context, id uint64) (*domain.Limit, error) {
	var po LimitPO
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&po).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toLimit(&po), nil
}

func (r *GormLimitRepository) GetByLimitID(ctx context.Context, limitID string) (*domain.Limit, error) {
	var po LimitPO
	err := r.db.WithContext(ctx).Where("limit_id = ?", limitID).First(&po).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toLimit(&po), nil
}

func (r *GormLimitRepository) GetByAccountAndType(ctx context.Context, accountID uint64, limitType domain.LimitType, scope domain.LimitScope, symbol string) (*domain.Limit, error) {
	var po LimitPO
	query := r.db.WithContext(ctx).Where("account_id = ? AND type = ? AND scope = ?", accountID, limitType, scope)
	if symbol != "" {
		query = query.Where("symbol = ?", symbol)
	}
	err := query.First(&po).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toLimit(&po), nil
}

func (r *GormLimitRepository) ListByAccount(ctx context.Context, accountID uint64) ([]*domain.Limit, error) {
	var pos []*LimitPO
	err := r.db.WithContext(ctx).Where("account_id = ?", accountID).Find(&pos).Error
	if err != nil {
		return nil, err
	}

	limits := make([]*domain.Limit, len(pos))
	for i, po := range pos {
		limits[i] = toLimit(po)
	}
	return limits, nil
}

func (r *GormLimitRepository) ListExceeded(ctx context.Context) ([]*domain.Limit, error) {
	var pos []*LimitPO
	err := r.db.WithContext(ctx).Where("status = ?", "EXCEEDED").Find(&pos).Error
	if err != nil {
		return nil, err
	}

	limits := make([]*domain.Limit, len(pos))
	for i, po := range pos {
		limits[i] = toLimit(po)
	}
	return limits, nil
}

func (r *GormLimitRepository) ResetDailyLimits(ctx context.Context) error {
	return r.db.WithContext(ctx).Model(&LimitPO{}).
		Where("type IN ?", []string{"DAILY", "LOSS"}).
		Updates(map[string]any{
			"current_value": 0,
			"used_percent":  0,
			"status":        "ACTIVE",
			"reset_at":      time.Now(),
		}).Error
}

type GormLimitBreachRepository struct {
	db *gorm.DB
}

func NewGormLimitBreachRepository(db *gorm.DB) *GormLimitBreachRepository {
	return &GormLimitBreachRepository{db: db}
}

func (r *GormLimitBreachRepository) Save(ctx context.Context, b *domain.LimitBreach) error {
	po := toLimitBreachPO(b)
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *GormLimitBreachRepository) GetByID(ctx context.Context, id uint64) (*domain.LimitBreach, error) {
	var po LimitBreachPO
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&po).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toLimitBreach(&po), nil
}

func (r *GormLimitBreachRepository) ListByAccount(ctx context.Context, accountID uint64, resolved bool, page, pageSize int) ([]*domain.LimitBreach, int64, error) {
	var pos []*LimitBreachPO
	var total int64

	query := r.db.WithContext(ctx).Model(&LimitBreachPO{}).Where("account_id = ?", accountID)
	if resolved {
		query = query.Where("resolved_at IS NOT NULL")
	} else {
		query = query.Where("resolved_at IS NULL")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	breaches := make([]*domain.LimitBreach, len(pos))
	for i, po := range pos {
		breaches[i] = toLimitBreach(po)
	}

	return breaches, total, nil
}

func (r *GormLimitBreachRepository) ListUnresolved(ctx context.Context, limit int) ([]*domain.LimitBreach, error) {
	var pos []*LimitBreachPO
	err := r.db.WithContext(ctx).
		Where("resolved_at IS NULL").
		Order("created_at DESC").
		Limit(limit).
		Find(&pos).Error
	if err != nil {
		return nil, err
	}

	breaches := make([]*domain.LimitBreach, len(pos))
	for i, po := range pos {
		breaches[i] = toLimitBreach(po)
	}
	return breaches, nil
}

type GormAccountLimitConfigRepository struct {
	db *gorm.DB
}

func NewGormAccountLimitConfigRepository(db *gorm.DB) *GormAccountLimitConfigRepository {
	return &GormAccountLimitConfigRepository{db: db}
}

func (r *GormAccountLimitConfigRepository) Save(ctx context.Context, c *domain.AccountLimitConfig) error {
	po := toAccountLimitConfigPO(c)
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *GormAccountLimitConfigRepository) Update(ctx context.Context, c *domain.AccountLimitConfig) error {
	po := toAccountLimitConfigPO(c)
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *GormAccountLimitConfigRepository) GetByAccountID(ctx context.Context, accountID uint64) (*domain.AccountLimitConfig, error) {
	var po AccountLimitConfigPO
	err := r.db.WithContext(ctx).Where("account_id = ?", accountID).First(&po).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toAccountLimitConfig(&po), nil
}

func toLimitPO(l *domain.Limit) *LimitPO {
	return &LimitPO{
		ID:               l.ID,
		LimitID:          l.LimitID,
		AccountID:        l.AccountID,
		Type:             string(l.Type),
		Scope:            string(l.Scope),
		Symbol:           l.Symbol,
		CurrentValue:     l.CurrentValue,
		LimitValue:       l.LimitValue,
		WarningThreshold: l.WarningThreshold,
		UsedPercent:      l.UsedPercent,
		Status:           string(l.Status),
		IsActive:         l.IsActive,
		StartTime:        l.StartTime,
		EndTime:          l.EndTime,
		ResetAt:          l.ResetAt,
		CreatedAt:        l.CreatedAt,
		UpdatedAt:        l.UpdatedAt,
	}
}

func toLimit(po *LimitPO) *domain.Limit {
	return &domain.Limit{
		ID:               po.ID,
		LimitID:          po.LimitID,
		AccountID:        po.AccountID,
		Type:             domain.LimitType(po.Type),
		Scope:            domain.LimitScope(po.Scope),
		Symbol:           po.Symbol,
		CurrentValue:     po.CurrentValue,
		LimitValue:       po.LimitValue,
		WarningThreshold: po.WarningThreshold,
		UsedPercent:      po.UsedPercent,
		Status:           domain.LimitStatus(po.Status),
		IsActive:         po.IsActive,
		StartTime:        po.StartTime,
		EndTime:          po.EndTime,
		ResetAt:          po.ResetAt,
		CreatedAt:        po.CreatedAt,
		UpdatedAt:        po.UpdatedAt,
	}
}

func toLimitBreachPO(b *domain.LimitBreach) *LimitBreachPO {
	return &LimitBreachPO{
		BreachID:      b.BreachID,
		LimitID:       b.LimitID,
		AccountID:     b.AccountID,
		Type:          string(b.Type),
		CurrentValue:  b.CurrentValue,
		LimitValue:    b.LimitValue,
		BreachPercent: b.BreachPercent,
		Action:        b.Action,
		ResolvedAt:    b.ResolvedAt,
		CreatedAt:     b.CreatedAt,
	}
}

func toLimitBreach(po *LimitBreachPO) *domain.LimitBreach {
	return &domain.LimitBreach{
		ID:            po.ID,
		BreachID:      po.BreachID,
		LimitID:       po.LimitID,
		AccountID:     po.AccountID,
		Type:          domain.LimitType(po.Type),
		CurrentValue:  po.CurrentValue,
		LimitValue:    po.LimitValue,
		BreachPercent: po.BreachPercent,
		Action:        po.Action,
		ResolvedAt:    po.ResolvedAt,
		CreatedAt:     po.CreatedAt,
	}
}

func toAccountLimitConfigPO(c *domain.AccountLimitConfig) *AccountLimitConfigPO {
	return &AccountLimitConfigPO{
		ID:                c.ID,
		AccountID:         c.AccountID,
		MaxPositionSize:   c.MaxPositionSize,
		MaxOrderSize:      c.MaxOrderSize,
		MaxDailyVolume:    c.MaxDailyVolume,
		MaxDailyLoss:      c.MaxDailyLoss,
		MaxExposure:       c.MaxExposure,
		MaxLeverage:       c.MaxLeverage,
		MaxConcentration:  c.MaxConcentration,
		WarningThreshold:  c.WarningThreshold,
		AutoFreezeEnabled: c.AutoFreezeEnabled,
		CreatedAt:         c.CreatedAt,
		UpdatedAt:         c.UpdatedAt,
	}
}

func toAccountLimitConfig(po *AccountLimitConfigPO) *domain.AccountLimitConfig {
	return &domain.AccountLimitConfig{
		ID:                po.ID,
		AccountID:         po.AccountID,
		MaxPositionSize:   po.MaxPositionSize,
		MaxOrderSize:      po.MaxOrderSize,
		MaxDailyVolume:    po.MaxDailyVolume,
		MaxDailyLoss:      po.MaxDailyLoss,
		MaxExposure:       po.MaxExposure,
		MaxLeverage:       po.MaxLeverage,
		MaxConcentration:  po.MaxConcentration,
		WarningThreshold:  po.WarningThreshold,
		AutoFreezeEnabled: po.AutoFreezeEnabled,
		CreatedAt:         po.CreatedAt,
		UpdatedAt:         po.UpdatedAt,
	}
}
