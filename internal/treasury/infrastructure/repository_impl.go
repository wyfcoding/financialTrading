// Package infrastructure 资金服务基础设施层实现
// 生成摘要：
// 1) 完整实现 domain 层定义的所有仓储接口
// 2) 包含 GormAccountRepository (保持原有逻辑)
// 3) 新增 GormCashPoolRepository、GormLiquidityForecastRepository、GormTransferInstructionRepository
// 4) 统一处理事务上下文 (contextx.GetTx)
package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/wyfcoding/financialtrading/internal/treasury/domain"
	"github.com/wyfcoding/pkg/contextx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// baseRepository 基础仓储，提供事务支持
type baseRepository struct {
	db *gorm.DB
}

func (r *baseRepository) getDB(ctx context.Context) *gorm.DB {
	// 尝试从 Context 中获取事务句柄
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok && tx != nil {
		return tx
	}
	return r.db
}

// TransactionManager 事务管理器
type TransactionManager struct {
	db *gorm.DB
}

func NewTransactionManager(db *gorm.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

// Transaction 开启一个新事务
func (tm *TransactionManager) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return tm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(contextx.WithTx(ctx, tx))
	})
}

// --- Account Repository ---

type GormAccountRepository struct {
	baseRepository
}

func NewGormAccountRepository(db *gorm.DB) domain.AccountRepository {
	return &GormAccountRepository{baseRepository{db: db}}
}

func (r *GormAccountRepository) Save(ctx context.Context, account *domain.Account) error {
	return r.getDB(ctx).WithContext(ctx).Save(account).Error
}

func (r *GormAccountRepository) GetByID(ctx context.Context, id uint64) (*domain.Account, error) {
	var account domain.Account
	if err := r.getDB(ctx).WithContext(ctx).First(&account, id).Error; err != nil {
		return nil, fmt.Errorf("account not found: %w", err)
	}
	return &account, nil
}

func (r *GormAccountRepository) GetByOwner(ctx context.Context, ownerID uint64, accType domain.AccountType, currency domain.Currency) (*domain.Account, error) {
	var account domain.Account
	err := r.getDB(ctx).WithContext(ctx).Where("owner_id = ? AND type = ? AND currency = ?", ownerID, accType, currency).First(&account).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (r *GormAccountRepository) GetWithLock(ctx context.Context, id uint64) (*domain.Account, error) {
	var account domain.Account
	if err := r.getDB(ctx).WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&account, id).Error; err != nil {
		return nil, fmt.Errorf("account lock failed: %w", err)
	}
	return &account, nil
}

// --- Transaction Repository ---

type GormTransactionRepository struct {
	baseRepository
}

func NewGormTransactionRepository(db *gorm.DB) domain.TransactionRepository {
	return &GormTransactionRepository{baseRepository{db: db}}
}

func (r *GormTransactionRepository) Save(ctx context.Context, tx *domain.Transaction) error {
	return r.getDB(ctx).WithContext(ctx).Create(tx).Error
}

func (r *GormTransactionRepository) List(ctx context.Context, accountID uint, txType *domain.TransactionType, limit, offset int) ([]*domain.Transaction, int64, error) {
	query := r.getDB(ctx).WithContext(ctx).Model(&domain.Transaction{}).Where("account_id = ?", accountID)
	if txType != nil {
		query = query.Where("type = ?", *txType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var txs []*domain.Transaction
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&txs).Error; err != nil {
		return nil, 0, err
	}
	return txs, total, nil
}

// --- Cash Pool Repository ---

type GormCashPoolRepository struct {
	baseRepository
}

func NewGormCashPoolRepository(db *gorm.DB) domain.CashPoolRepository {
	return &GormCashPoolRepository{baseRepository{db: db}}
}

func (r *GormCashPoolRepository) Save(ctx context.Context, pool *domain.CashPool) error {
	// 级联保存 Accounts
	return r.getDB(ctx).WithContext(ctx).Session(&gorm.Session{FullSaveAssociations: true}).Save(pool).Error
}

func (r *GormCashPoolRepository) GetByID(ctx context.Context, id uint64) (*domain.CashPool, error) {
	var pool domain.CashPool
	if err := r.getDB(ctx).WithContext(ctx).Preload("Accounts").First(&pool, id).Error; err != nil {
		return nil, err
	}
	return &pool, nil
}

func (r *GormCashPoolRepository) ListAll(ctx context.Context) ([]*domain.CashPool, error) {
	var pools []*domain.CashPool
	if err := r.getDB(ctx).WithContext(ctx).Find(&pools).Error; err != nil {
		return nil, err
	}
	return pools, nil
}

// --- Liquidity Forecast Repository ---

type GormLiquidityForecastRepository struct {
	baseRepository
}

func NewGormLiquidityForecastRepository(db *gorm.DB) domain.LiquidityForecastRepository {
	return &GormLiquidityForecastRepository{baseRepository{db: db}}
}

func (r *GormLiquidityForecastRepository) Save(ctx context.Context, forecast *domain.LiquidityForecast) error {
	return r.getDB(ctx).WithContext(ctx).Save(forecast).Error
}

func (r *GormLiquidityForecastRepository) ListByPoolAndDateRange(ctx context.Context, poolID uint64, start, end time.Time) ([]*domain.LiquidityForecast, error) {
	var forecasts []*domain.LiquidityForecast
	err := r.getDB(ctx).WithContext(ctx).
		Where("pool_id = ? AND date >= ? AND date <= ?", poolID, start, end).
		Order("date ASC").
		Find(&forecasts).Error
	if err != nil {
		return nil, err
	}
	return forecasts, nil
}

// --- Transfer Instruction Repository ---

type GormTransferInstructionRepository struct {
	baseRepository
}

func NewGormTransferInstructionRepository(db *gorm.DB) domain.TransferInstructionRepository {
	return &GormTransferInstructionRepository{baseRepository{db: db}}
}

func (r *GormTransferInstructionRepository) Save(ctx context.Context, instruction *domain.TransferInstruction) error {
	return r.getDB(ctx).WithContext(ctx).Save(instruction).Error
}

func (r *GormTransferInstructionRepository) GetByID(ctx context.Context, id string) (*domain.TransferInstruction, error) {
	var instruction domain.TransferInstruction
	if err := r.getDB(ctx).WithContext(ctx).Where("instruction_id = ?", id).First(&instruction).Error; err != nil {
		return nil, err
	}
	return &instruction, nil
}

func (r *GormTransferInstructionRepository) ListPending(ctx context.Context, limit int) ([]*domain.TransferInstruction, error) {
	var ins []*domain.TransferInstruction
	err := r.getDB(ctx).WithContext(ctx).
		Where("status = ?", domain.InstructionStatusApproved). // Approved awaiting execution
		Limit(limit).
		Find(&ins).Error
	if err != nil {
		return nil, err
	}
	return ins, nil
}
