// 包 了仓储接口的具体实现。
// 这一层负责与具体的数据存储（此处为数据库）进行交互。
package repository

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialTrading/internal/execution/domain"
	"github.com/wyfcoding/pkg/logging"
	"gorm.io/gorm"
)

// ExecutionModel 是执行记录的数据库模型 (GORM Model)。
// 它直接映射到数据库中的 `executions` 表。
// 注意：该模型与领域实体 (domain.Execution) 是分离的，这是 DDD 的一个重要实践。
type ExecutionModel struct {
	gorm.Model
	ExecutionID      string `gorm:"column:execution_id;type:varchar(50);uniqueIndex;not null;comment:执行ID"`
	OrderID          string `gorm:"column:order_id;type:varchar(50);index;not null;comment:订单ID"`
	UserID           string `gorm:"column:user_id;type:varchar(50);index;not null;comment:用户ID"`
	Symbol           string `gorm:"column:symbol;type:varchar(50);index;not null;comment:交易对"`
	Side             string `gorm:"column:side;type:varchar(10);not null;comment:买卖方向"`
	ExecutedPrice    string `gorm:"column:executed_price;type:decimal(20,8);not null;comment:成交价格"`
	ExecutedQuantity string `gorm:"column:executed_quantity;type:decimal(20,8);not null;comment:成交数量"`
	Status           string `gorm:"column:status;type:varchar(20);index;not null;comment:执行状态"`
}

// TableName 显式指定 GORM 应使用的表名。
func (ExecutionModel) TableName() string {
	return "executions"
}

// ExecutionRepositoryImpl 是 ExecutionRepository 接口的 GORM 实现。
type ExecutionRepositoryImpl struct {
	db *gorm.DB // 依赖注入的数据库连接实例
}

// NewExecutionRepository 是 ExecutionRepositoryImpl 的构造函数。
func NewExecutionRepository(database *gorm.DB) domain.ExecutionRepository {
	return &ExecutionRepositoryImpl{
		db: database,
	}
}

// Save 实现了保存执行记录的接口。
// 它将领域实体转换为数据库模型，然后执行数据库插入操作。
func (er *ExecutionRepositoryImpl) Save(ctx context.Context, execution *domain.Execution) error {
	// 将领域实体 (domain.Execution) 转换为数据库模型 (ExecutionModel)
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

	// 使用 GORM 的 Create 方法将记录插入数据库
	if err := er.db.WithContext(ctx).Create(model).Error; err != nil {
		logging.Error(ctx, "Failed to save execution",
			"execution_id", execution.ExecutionID,
			"error", err,
		)
		return fmt.Errorf("failed to save execution: %w", err)
	}

	// 回填 GORM 生成的 ID 和时间戳到领域实体中
	execution.Model = model.Model
	return nil
}

// Get 实现了根据 ID 获取执行记录的接口。
func (er *ExecutionRepositoryImpl) Get(ctx context.Context, executionID string) (*domain.Execution, error) {
	var model ExecutionModel

	// 使用 GORM 的 First 方法查询记录
	if err := er.db.WithContext(ctx).Where("execution_id = ?", executionID).First(&model).Error; err != nil {
		// 如果 GORM 返回 "record not found"，则按约定返回 (nil, nil)
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logging.Error(ctx, "Failed to get execution",
			"execution_id", executionID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	// 将查询到的数据库模型转换为领域实体
	return er.modelToDomain(&model), nil
}

// GetByOrder 实现了获取订单所有执行记录的接口。
func (er *ExecutionRepositoryImpl) GetByOrder(ctx context.Context, orderID string) ([]*domain.Execution, error) {
	var models []ExecutionModel

	if err := er.db.WithContext(ctx).Where("order_id = ?", orderID).Order("created_at DESC").Find(&models).Error; err != nil {
		logging.Error(ctx, "Failed to get executions by order",
			"order_id", orderID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get executions by order: %w", err)
	}

	// 批量将数据库模型转换为领域实体
	executions := make([]*domain.Execution, 0, len(models))
	for i := range models {
		executions = append(executions, er.modelToDomain(&models[i]))
	}

	return executions, nil
}

// GetByUser 实现了分页获取用户执行历史的接口。
func (er *ExecutionRepositoryImpl) GetByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Execution, int64, error) {
	var models []ExecutionModel
	var total int64

	query := er.db.WithContext(ctx).Where("user_id = ?", userID)

	// 先执行 Count 查询获取总记录数
	if err := query.Model(&ExecutionModel{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count executions: %w", err)
	}

	// 再执行分页查询获取数据
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&models).Error; err != nil {
		logging.Error(ctx, "Failed to get executions by user",
			"user_id", userID,
			"error", err,
		)
		return nil, 0, fmt.Errorf("failed to get executions by user: %w", err)
	}

	executions := make([]*domain.Execution, 0, len(models))
	for i := range models {
		executions = append(executions, er.modelToDomain(&models[i]))
	}

	return executions, total, nil
}

// modelToDomain 是一个辅助函数，用于将数据库模型 (ExecutionModel) 转换为领域实体 (domain.Execution)。
// 这是保持领域层纯粹性的关键一步，避免基础设施层的细节泄漏到领域层。
func (er *ExecutionRepositoryImpl) modelToDomain(model *ExecutionModel) *domain.Execution {
	// 从字符串安全地转换回 decimal.Decimal 类型，忽略错误以简化，但在生产中应处理
	price, _ := decimal.NewFromString(model.ExecutedPrice)
	quantity, _ := decimal.NewFromString(model.ExecutedQuantity)

	return &domain.Execution{
		Model:            model.Model,
		ExecutionID:      model.ExecutionID,
		OrderID:          model.OrderID,
		UserID:           model.UserID,
		Symbol:           model.Symbol,
		Side:             domain.OrderSide(model.Side),
		ExecutedPrice:    price,
		ExecutedQuantity: quantity,
		Status:           domain.ExecutionStatus(model.Status),
		CreatedAt:        model.CreatedAt,
		UpdatedAt:        model.UpdatedAt,
	}
}
