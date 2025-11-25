// Package repository 包含仓储实现
package repository

import (
	"fmt"

	"github.com/fynnwu/FinancialTrading/internal/execution/domain"
	"github.com/fynnwu/FinancialTrading/pkg/db"
	"github.com/fynnwu/FinancialTrading/pkg/logger"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ExecutionModel 执行记录数据库模型
type ExecutionModel struct {
	gorm.Model
	// 执行 ID
	ExecutionID string `gorm:"column:execution_id;type:varchar(50);uniqueIndex;not null" json:"execution_id"`
	// 订单 ID
	OrderID string `gorm:"column:order_id;type:varchar(50);index;not null" json:"order_id"`
	// 用户 ID
	UserID string `gorm:"column:user_id;type:varchar(50);index;not null" json:"user_id"`
	// 交易对
	Symbol string `gorm:"column:symbol;type:varchar(50);index;not null" json:"symbol"`
	// 买卖方向
	Side string `gorm:"column:side;type:varchar(10);not null" json:"side"`
	// 执行价格
	ExecutedPrice string `gorm:"column:executed_price;type:decimal(20,8);not null" json:"executed_price"`
	// 执行数量
	ExecutedQuantity string `gorm:"column:executed_quantity;type:decimal(20,8);not null" json:"executed_quantity"`
	// 执行状态
	Status string `gorm:"column:status;type:varchar(20);index;not null" json:"status"`
}

// TableName 指定表名
func (ExecutionModel) TableName() string {
	return "executions"
}

// ExecutionRepositoryImpl 执行记录仓储实现
type ExecutionRepositoryImpl struct {
	db *db.DB
}

// NewExecutionRepository 创建执行记录仓储
func NewExecutionRepository(database *db.DB) domain.ExecutionRepository {
	return &ExecutionRepositoryImpl{
		db: database,
	}
}

// Save 保存执行记录
func (er *ExecutionRepositoryImpl) Save(execution *domain.Execution) error {
	model := &ExecutionModel{
		ExecutionID:      execution.ExecutionID,
		OrderID:          execution.OrderID,
		UserID:           execution.UserID,
		Symbol:           execution.Symbol,
		Side:             execution.Side,
		ExecutedPrice:    execution.ExecutedPrice.String(),
		ExecutedQuantity: execution.ExecutedQuantity.String(),
		Status:           string(execution.Status),
	}

	if err := er.db.Create(model).Error; err != nil {
		logger.Error("Failed to save execution",
			zap.String("execution_id", execution.ExecutionID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save execution: %w", err)
	}

	return nil
}

// Get 获取执行记录
func (er *ExecutionRepositoryImpl) Get(executionID string) (*domain.Execution, error) {
	var model ExecutionModel

	if err := er.db.Where("execution_id = ?", executionID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logger.Error("Failed to get execution",
			zap.String("execution_id", executionID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	return er.modelToDomain(&model), nil
}

// GetByOrder 获取订单执行历史
func (er *ExecutionRepositoryImpl) GetByOrder(orderID string) ([]*domain.Execution, error) {
	var models []ExecutionModel

	if err := er.db.Where("order_id = ?", orderID).Order("created_at DESC").Find(&models).Error; err != nil {
		logger.Error("Failed to get executions by order",
			zap.String("order_id", orderID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get executions by order: %w", err)
	}

	executions := make([]*domain.Execution, 0, len(models))
	for _, model := range models {
		executions = append(executions, er.modelToDomain(&model))
	}

	return executions, nil
}

// GetByUser 获取用户执行历史
func (er *ExecutionRepositoryImpl) GetByUser(userID string, limit, offset int) ([]*domain.Execution, int64, error) {
	var models []ExecutionModel
	var total int64

	query := er.db.Where("user_id = ?", userID)

	if err := query.Model(&ExecutionModel{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count executions: %w", err)
	}

	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&models).Error; err != nil {
		logger.Error("Failed to get executions by user",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return nil, 0, fmt.Errorf("failed to get executions by user: %w", err)
	}

	executions := make([]*domain.Execution, 0, len(models))
	for _, model := range models {
		executions = append(executions, er.modelToDomain(&model))
	}

	return executions, total, nil
}

// modelToDomain 将数据库模型转换为领域对象
func (er *ExecutionRepositoryImpl) modelToDomain(model *ExecutionModel) *domain.Execution {
	price, _ := decimal.NewFromString(model.ExecutedPrice)
	quantity, _ := decimal.NewFromString(model.ExecutedQuantity)

	return &domain.Execution{
		ExecutionID:      model.ExecutionID,
		OrderID:          model.OrderID,
		UserID:           model.UserID,
		Symbol:           model.Symbol,
		Side:             model.Side,
		ExecutedPrice:    price,
		ExecutedQuantity: quantity,
		Status:           domain.ExecutionStatus(model.Status),
		CreatedAt:        model.CreatedAt,
		UpdatedAt:        model.UpdatedAt,
	}
}
