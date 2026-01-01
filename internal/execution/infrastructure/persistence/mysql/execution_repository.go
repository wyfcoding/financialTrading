// Package mysql 提供了执行仓储接口的 MySQL GORM 实现。
package mysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
	"github.com/wyfcoding/pkg/logging"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ExecutionModel 是执行记录的数据库模型。
type ExecutionModel struct {
	gorm.Model
	ExecutionID      string `gorm:"column:execution_id;type:varchar(32);uniqueIndex;not null"`
	OrderID          string `gorm:"column:order_id;type:varchar(32);index;not null"`
	UserID           string `gorm:"column:user_id;type:varchar(32);index;not null"`
	Symbol           string `gorm:"column:symbol;type:varchar(20);not null"`
	Side             string `gorm:"column:side;type:varchar(10);not null"`
	ExecutedPrice    string `gorm:"column:executed_price;type:decimal(32,18);not null"`
	ExecutedQuantity string `gorm:"column:executed_quantity;type:decimal(32,18);not null"`
	Status           string `gorm:"column:status;type:varchar(20);index;not null"`
}

// TableName 指定表名
func (ExecutionModel) TableName() string {
	return "executions"
}

// AlgoOrderModel 是算法订单的数据库模型。
type AlgoOrderModel struct {
	gorm.Model
	AlgoID            string    `gorm:"column:algo_id;type:varchar(36);uniqueIndex;not null"`
	UserID            string    `gorm:"column:user_id;type:varchar(32);index;not null"`
	Symbol            string    `gorm:"column:symbol;type:varchar(20);not null"`
	Side              string    `gorm:"column:side;type:varchar(10);not null"`
	TotalQuantity     string    `gorm:"column:total_quantity;type:decimal(32,18);not null"`
	ExecutedQuantity  string    `gorm:"column:executed_quantity;type:decimal(32,18);not null"`
	AlgoType          string    `gorm:"column:algo_type;type:varchar(20);not null"`
	StartTime         time.Time `gorm:"column:start_time"`
	EndTime           time.Time `gorm:"column:end_time"`
	ParticipationRate string    `gorm:"column:participation_rate;type:decimal(10,4)"`
	Status            string    `gorm:"column:status;type:varchar(20);index;not null"`
}

func (AlgoOrderModel) TableName() string {
	return "algo_orders"
}

// executionRepositoryImpl 是 domain.ExecutionRepository 接口的 GORM 实现。
type executionRepositoryImpl struct {
	db *gorm.DB
}

// NewExecutionRepository 创建执行仓储实例
func NewExecutionRepository(db *gorm.DB) domain.ExecutionRepository {
	return &executionRepositoryImpl{db: db}
}

// Save 实现 domain.ExecutionRepository.Save
func (r *executionRepositoryImpl) Save(ctx context.Context, execution *domain.Execution) error {
	model := &ExecutionModel{
		Model:            execution.Model,
		ExecutionID:      execution.ExecutionID,
		OrderID:          execution.OrderID,
		UserID:           execution.UserID,
		Symbol:           execution.Symbol,
		Side:             string(execution.Side),
		ExecutedPrice:    execution.ExecutedPrice.String(),
		ExecutedQuantity: execution.ExecutedQuantity.String(),
		Status:           string(execution.Status),
	}

	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "execution_id"}},
		UpdateAll: true,
	}).Create(model).Error
	if err != nil {
		logging.Error(ctx, "execution_repository.Save failed", "execution_id", execution.ExecutionID, "error", err)
		return fmt.Errorf("failed to save execution: %w", err)
	}

	execution.Model = model.Model
	return nil
}

// Get 实现 domain.ExecutionRepository.Get
func (r *executionRepositoryImpl) Get(ctx context.Context, executionID string) (*domain.Execution, error) {
	var model ExecutionModel
	if err := r.db.WithContext(ctx).Where("execution_id = ?", executionID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logging.Error(ctx, "execution_repository.Get failed", "execution_id", executionID, "error", err)
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}
	return r.toDomain(&model), nil
}

// GetByOrder 实现 domain.ExecutionRepository.GetByOrder
func (r *executionRepositoryImpl) GetByOrder(ctx context.Context, orderID string) ([]*domain.Execution, error) {
	var models []ExecutionModel
	if err := r.db.WithContext(ctx).Where("order_id = ?", orderID).Order("created_at asc").Find(&models).Error; err != nil {
		logging.Error(ctx, "execution_repository.GetByOrder failed", "order_id", orderID, "error", err)
		return nil, fmt.Errorf("failed to get executions by order: %w", err)
	}

	res := make([]*domain.Execution, len(models))
	for i, m := range models {
		res[i] = r.toDomain(&m)
	}
	return res, nil
}

// GetByUser 实现 domain.ExecutionRepository.GetByUser
func (r *executionRepositoryImpl) GetByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Execution, int64, error) {
	var models []ExecutionModel
	var total int64
	db := r.db.WithContext(ctx).Model(&ExecutionModel{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at desc").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		logging.Error(ctx, "execution_repository.GetByUser failed", "user_id", userID, "error", err)
		return nil, 0, fmt.Errorf("failed to get execution history: %w", err)
	}

	res := make([]*domain.Execution, len(models))
	for i, m := range models {
		res[i] = r.toDomain(&m)
	}
	return res, total, nil
}

func (r *executionRepositoryImpl) toDomain(m *ExecutionModel) *domain.Execution {
	p, err := decimal.NewFromString(m.ExecutedPrice)
	if err != nil {
		p = decimal.Zero
	}
	q, err := decimal.NewFromString(m.ExecutedQuantity)
	if err != nil {
		q = decimal.Zero
	}
	return &domain.Execution{
		Model:            m.Model,
		ExecutionID:      m.ExecutionID,
		OrderID:          m.OrderID,
		UserID:           m.UserID,
		Symbol:           m.Symbol,
		Side:             domain.OrderSide(m.Side),
		ExecutedPrice:    p,
		ExecutedQuantity: q,
		Status:           domain.ExecutionStatus(m.Status),
	}
}

// SaveAlgoOrder 实现 domain.ExecutionRepository.SaveAlgoOrder
func (r *executionRepositoryImpl) SaveAlgoOrder(ctx context.Context, algoOrder *domain.AlgoOrder) error {
	model := &AlgoOrderModel{
		Model:             algoOrder.Model,
		AlgoID:            algoOrder.AlgoID,
		UserID:            algoOrder.UserID,
		Symbol:            algoOrder.Symbol,
		Side:              string(algoOrder.Side),
		TotalQuantity:     algoOrder.TotalQuantity.String(),
		ExecutedQuantity:  algoOrder.ExecutedQuantity.String(),
		AlgoType:          string(algoOrder.AlgoType),
		StartTime:         algoOrder.StartTime,
		EndTime:           algoOrder.EndTime,
		ParticipationRate: algoOrder.ParticipationRate.String(),
		Status:            string(algoOrder.Status),
	}

	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "algo_id"}},
		UpdateAll: true,
	}).Create(model).Error
	if err != nil {
		logging.Error(ctx, "execution_repository.SaveAlgoOrder failed", "algo_id", algoOrder.AlgoID, "error", err)
		return fmt.Errorf("failed to save algo order: %w", err)
	}

	algoOrder.Model = model.Model
	return nil
}

// GetAlgoOrder 实现 domain.ExecutionRepository.GetAlgoOrder
func (r *executionRepositoryImpl) GetAlgoOrder(ctx context.Context, algoID string) (*domain.AlgoOrder, error) {
	var model AlgoOrderModel
	if err := r.db.WithContext(ctx).Where("algo_id = ?", algoID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.algoToDomain(&model), nil
}

// ListAlgoOrders 实现 domain.ExecutionRepository.ListAlgoOrders
func (r *executionRepositoryImpl) ListAlgoOrders(ctx context.Context, userID string, status domain.ExecutionStatus) ([]*domain.AlgoOrder, error) {
	var models []AlgoOrderModel
	db := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if status != "" {
		db = db.Where("status = ?", string(status))
	}

	if err := db.Find(&models).Error; err != nil {
		return nil, err
	}

	res := make([]*domain.AlgoOrder, len(models))
	for i, m := range models {
		res[i] = r.algoToDomain(&m)
	}
	return res, nil
}

func (r *executionRepositoryImpl) algoToDomain(m *AlgoOrderModel) *domain.AlgoOrder {
	total, _ := decimal.NewFromString(m.TotalQuantity)
	executed, _ := decimal.NewFromString(m.ExecutedQuantity)
	participation, _ := decimal.NewFromString(m.ParticipationRate)

	return &domain.AlgoOrder{
		Model:             m.Model,
		AlgoID:            m.AlgoID,
		UserID:            m.UserID,
		Symbol:            m.Symbol,
		Side:              domain.OrderSide(m.Side),
		TotalQuantity:     total,
		ExecutedQuantity:  executed,
		AlgoType:          domain.AlgoType(m.AlgoType),
		StartTime:         m.StartTime,
		EndTime:           m.EndTime,
		ParticipationRate: participation,
		Status:            domain.ExecutionStatus(m.Status),
	}
}
