// 生成摘要：充值和提现订单 MySQL 仓储实现。
package mysql

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/account/domain"
	"github.com/wyfcoding/pkg/contextx"
)

// DepositOrderModel 充值订单数据库模型
type DepositOrderModel struct {
	gorm.Model
	DepositNo     string          `gorm:"column:deposit_no;type:varchar(64);uniqueIndex;not null"`
	UserID        string          `gorm:"column:user_id;type:varchar(32);index;not null"`
	AccountID     string          `gorm:"column:account_id;type:varchar(32);index;not null"`
	Amount        decimal.Decimal `gorm:"column:amount;type:decimal(32,18);not null"`
	Currency      string          `gorm:"column:currency;type:varchar(16);not null;default:'USDT'"`
	GatewayType   string          `gorm:"column:gateway_type;type:varchar(32);not null"`
	Status        string          `gorm:"column:status;type:varchar(32);not null;index"`
	TransactionID string          `gorm:"column:transaction_id;type:varchar(128)"`
	ThirdPartyNo  string          `gorm:"column:third_party_no;type:varchar(128)"`
	PaymentURL    string          `gorm:"column:payment_url;type:text"`
	FailureReason string          `gorm:"column:failure_reason;type:text"`
	ConfirmedAt   *time.Time      `gorm:"column:confirmed_at"`
	CompletedAt   *time.Time      `gorm:"column:completed_at"`
}

func (DepositOrderModel) TableName() string {
	return "deposit_orders"
}

// WithdrawalOrderModel 提现订单数据库模型
type WithdrawalOrderModel struct {
	gorm.Model
	WithdrawalNo     string          `gorm:"column:withdrawal_no;type:varchar(64);uniqueIndex;not null"`
	UserID           string          `gorm:"column:user_id;type:varchar(32);index;not null"`
	AccountID        string          `gorm:"column:account_id;type:varchar(32);index;not null"`
	Amount           decimal.Decimal `gorm:"column:amount;type:decimal(32,18);not null"`
	Fee              decimal.Decimal `gorm:"column:fee;type:decimal(32,18);not null;default:0"`
	NetAmount        decimal.Decimal `gorm:"column:net_amount;type:decimal(32,18);not null"`
	Currency         string          `gorm:"column:currency;type:varchar(16);not null;default:'USDT'"`
	BankAccountNo    string          `gorm:"column:bank_account_no;type:varchar(64);not null"`
	BankName         string          `gorm:"column:bank_name;type:varchar(128);not null"`
	BankHolder       string          `gorm:"column:bank_holder;type:varchar(128);not null"`
	Status           string          `gorm:"column:status;type:varchar(32);not null;index"`
	AuditRemark      string          `gorm:"column:audit_remark;type:text"`
	AuditedBy        string          `gorm:"column:audited_by;type:varchar(64)"`
	AuditedAt        *time.Time      `gorm:"column:audited_at"`
	GatewayReference string          `gorm:"column:gateway_reference;type:varchar(128)"`
	FailureReason    string          `gorm:"column:failure_reason;type:text"`
	CompletedAt      *time.Time      `gorm:"column:completed_at"`
}

func (WithdrawalOrderModel) TableName() string {
	return "withdrawal_orders"
}

// DepositOrderMySQLRepository 充值订单 MySQL 仓储实现
type DepositOrderMySQLRepository struct {
	db *gorm.DB
}

// NewDepositOrderRepository 创建充值订单仓储
func NewDepositOrderRepository(db *gorm.DB) domain.DepositOrderRepository {
	_ = db.AutoMigrate(&DepositOrderModel{})
	return &DepositOrderMySQLRepository{db: db}
}

func (r *DepositOrderMySQLRepository) getDB(ctx context.Context) *gorm.DB {
	if tx := contextx.GetTx(ctx); tx != nil {
		if gormTx, ok := tx.(*gorm.DB); ok {
			return gormTx
		}
	}
	return r.db.WithContext(ctx)
}

func (r *DepositOrderMySQLRepository) Save(ctx context.Context, d *domain.DepositOrder) error {
	model := r.toModel(d)
	return r.getDB(ctx).Create(model).Error
}

func (r *DepositOrderMySQLRepository) FindByID(ctx context.Context, id uint) (*domain.DepositOrder, error) {
	var model DepositOrderModel
	if err := r.getDB(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomain(&model), nil
}

func (r *DepositOrderMySQLRepository) FindByDepositNo(ctx context.Context, depositNo string) (*domain.DepositOrder, error) {
	var model DepositOrderModel
	if err := r.getDB(ctx).Where("deposit_no = ?", depositNo).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomain(&model), nil
}

func (r *DepositOrderMySQLRepository) FindByUserID(ctx context.Context, userID string, offset, limit int) ([]*domain.DepositOrder, int64, error) {
	var models []DepositOrderModel
	var total int64
	db := r.getDB(ctx).Model(&DepositOrderModel{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	result := make([]*domain.DepositOrder, len(models))
	for i, m := range models {
		result[i] = r.toDomain(&m)
	}
	return result, total, nil
}

func (r *DepositOrderMySQLRepository) FindByAccountID(ctx context.Context, accountID string, offset, limit int) ([]*domain.DepositOrder, int64, error) {
	var models []DepositOrderModel
	var total int64
	db := r.getDB(ctx).Model(&DepositOrderModel{}).Where("account_id = ?", accountID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	result := make([]*domain.DepositOrder, len(models))
	for i, m := range models {
		result[i] = r.toDomain(&m)
	}
	return result, total, nil
}

func (r *DepositOrderMySQLRepository) Update(ctx context.Context, d *domain.DepositOrder) error {
	model := r.toModel(d)
	return r.getDB(ctx).Model(&DepositOrderModel{}).Where("id = ?", d.ID).Updates(model).Error
}

func (r *DepositOrderMySQLRepository) WithTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

func (r *DepositOrderMySQLRepository) toModel(d *domain.DepositOrder) *DepositOrderModel {
	return &DepositOrderModel{
		Model:         gorm.Model{ID: d.ID, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt},
		DepositNo:     d.DepositNo,
		UserID:        d.UserID,
		AccountID:     d.AccountID,
		Amount:        d.Amount,
		Currency:      d.Currency,
		GatewayType:   string(d.GatewayType),
		Status:        string(d.Status),
		TransactionID: d.TransactionID,
		ThirdPartyNo:  d.ThirdPartyNo,
		PaymentURL:    d.PaymentURL,
		FailureReason: d.FailureReason,
		ConfirmedAt:   d.ConfirmedAt,
		CompletedAt:   d.CompletedAt,
	}
}

func (r *DepositOrderMySQLRepository) toDomain(m *DepositOrderModel) *domain.DepositOrder {
	d := &domain.DepositOrder{
		ID:            m.Model.ID,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		DepositNo:     m.DepositNo,
		UserID:        m.UserID,
		AccountID:     m.AccountID,
		Amount:        m.Amount,
		Currency:      m.Currency,
		GatewayType:   domain.GatewayType(m.GatewayType),
		Status:        domain.DepositStatus(m.Status),
		TransactionID: m.TransactionID,
		ThirdPartyNo:  m.ThirdPartyNo,
		PaymentURL:    m.PaymentURL,
		FailureReason: m.FailureReason,
		ConfirmedAt:   m.ConfirmedAt,
		CompletedAt:   m.CompletedAt,
	}
	d.InitFSM()
	return d
}

// WithdrawalOrderMySQLRepository 提现订单 MySQL 仓储实现
type WithdrawalOrderMySQLRepository struct {
	db *gorm.DB
}

// NewWithdrawalOrderRepository 创建提现订单仓储
func NewWithdrawalOrderRepository(db *gorm.DB) domain.WithdrawalOrderRepository {
	_ = db.AutoMigrate(&WithdrawalOrderModel{})
	return &WithdrawalOrderMySQLRepository{db: db}
}

func (r *WithdrawalOrderMySQLRepository) getDB(ctx context.Context) *gorm.DB {
	if tx := contextx.GetTx(ctx); tx != nil {
		if gormTx, ok := tx.(*gorm.DB); ok {
			return gormTx
		}
	}
	return r.db.WithContext(ctx)
}

func (r *WithdrawalOrderMySQLRepository) Save(ctx context.Context, w *domain.WithdrawalOrder) error {
	model := r.toModel(w)
	return r.getDB(ctx).Create(model).Error
}

func (r *WithdrawalOrderMySQLRepository) FindByID(ctx context.Context, id uint) (*domain.WithdrawalOrder, error) {
	var model WithdrawalOrderModel
	if err := r.getDB(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomain(&model), nil
}

func (r *WithdrawalOrderMySQLRepository) FindByWithdrawalNo(ctx context.Context, withdrawalNo string) (*domain.WithdrawalOrder, error) {
	var model WithdrawalOrderModel
	if err := r.getDB(ctx).Where("withdrawal_no = ?", withdrawalNo).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomain(&model), nil
}

func (r *WithdrawalOrderMySQLRepository) FindByUserID(ctx context.Context, userID string, offset, limit int) ([]*domain.WithdrawalOrder, int64, error) {
	var models []WithdrawalOrderModel
	var total int64
	db := r.getDB(ctx).Model(&WithdrawalOrderModel{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	result := make([]*domain.WithdrawalOrder, len(models))
	for i, m := range models {
		result[i] = r.toDomain(&m)
	}
	return result, total, nil
}

func (r *WithdrawalOrderMySQLRepository) FindByAccountID(ctx context.Context, accountID string, offset, limit int) ([]*domain.WithdrawalOrder, int64, error) {
	var models []WithdrawalOrderModel
	var total int64
	db := r.getDB(ctx).Model(&WithdrawalOrderModel{}).Where("account_id = ?", accountID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	result := make([]*domain.WithdrawalOrder, len(models))
	for i, m := range models {
		result[i] = r.toDomain(&m)
	}
	return result, total, nil
}

func (r *WithdrawalOrderMySQLRepository) FindPendingForAudit(ctx context.Context, offset, limit int) ([]*domain.WithdrawalOrder, int64, error) {
	var models []WithdrawalOrderModel
	var total int64
	db := r.getDB(ctx).Model(&WithdrawalOrderModel{}).Where("status = ?", string(domain.WithdrawalStatusAuditing))
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at ASC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	result := make([]*domain.WithdrawalOrder, len(models))
	for i, m := range models {
		result[i] = r.toDomain(&m)
	}
	return result, total, nil
}

func (r *WithdrawalOrderMySQLRepository) Update(ctx context.Context, w *domain.WithdrawalOrder) error {
	model := r.toModel(w)
	return r.getDB(ctx).Model(&WithdrawalOrderModel{}).Where("id = ?", w.ID).Updates(model).Error
}

func (r *WithdrawalOrderMySQLRepository) WithTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

func (r *WithdrawalOrderMySQLRepository) toModel(w *domain.WithdrawalOrder) *WithdrawalOrderModel {
	return &WithdrawalOrderModel{
		Model:            gorm.Model{ID: w.ID, CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt},
		WithdrawalNo:     w.WithdrawalNo,
		UserID:           w.UserID,
		AccountID:        w.AccountID,
		Amount:           w.Amount,
		Fee:              w.Fee,
		NetAmount:        w.NetAmount,
		Currency:         w.Currency,
		BankAccountNo:    w.BankAccountNo,
		BankName:         w.BankName,
		BankHolder:       w.BankHolder,
		Status:           string(w.Status),
		AuditRemark:      w.AuditRemark,
		AuditedBy:        w.AuditedBy,
		AuditedAt:        w.AuditedAt,
		GatewayReference: w.GatewayReference,
		FailureReason:    w.FailureReason,
		CompletedAt:      w.CompletedAt,
	}
}

func (r *WithdrawalOrderMySQLRepository) toDomain(m *WithdrawalOrderModel) *domain.WithdrawalOrder {
	w := &domain.WithdrawalOrder{
		ID:               m.Model.ID,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
		WithdrawalNo:     m.WithdrawalNo,
		UserID:           m.UserID,
		AccountID:        m.AccountID,
		Amount:           m.Amount,
		Fee:              m.Fee,
		NetAmount:        m.NetAmount,
		Currency:         m.Currency,
		BankAccountNo:    m.BankAccountNo,
		BankName:         m.BankName,
		BankHolder:       m.BankHolder,
		Status:           domain.WithdrawalStatus(m.Status),
		AuditRemark:      m.AuditRemark,
		AuditedBy:        m.AuditedBy,
		AuditedAt:        m.AuditedAt,
		GatewayReference: m.GatewayReference,
		FailureReason:    m.FailureReason,
		CompletedAt:      m.CompletedAt,
	}
	w.InitFSM()
	return w
}
