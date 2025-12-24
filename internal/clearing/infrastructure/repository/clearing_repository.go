// 包 了仓储接口的具体实现。
// 这一层负责与具体的数据存储（如数据库、缓存）进行交互，实现了领域层定义的仓储接口。
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
	"github.com/wyfcoding/pkg/logging"
	"gorm.io/gorm"
)

// SettlementModel 是清算记录的数据库模型 (GORM Model)。
// 它直接映射到数据库中的 `settlements` 表，用于数据的持久化。
// 与领域实体 (domain.Settlement) 分离，使得数据库结构的变化不会直接影响核心领域逻辑。
type SettlementModel struct {
	gorm.Model
	SettlementID   string `gorm:"column:settlement_id;type:varchar(50);uniqueIndex;not null"`
	TradeID        string `gorm:"column:trade_id;type:varchar(50);index;not null"`
	BuyUserID      string `gorm:"column:buy_user_id;type:varchar(50);index;not null"`
	SellUserID     string `gorm:"column:sell_user_id;type:varchar(50);index;not null"`
	Symbol         string `gorm:"column:symbol;type:varchar(50);not null"`
	Quantity       string `gorm:"column:quantity;type:decimal(20,8);not null"` // 数据库中存字符串，保证精度
	Price          string `gorm:"column:price;type:decimal(20,8);not null"`    // 数据库中存字符串，保证精度
	Status         string `gorm:"column:status;type:varchar(20);index;not null"`
	SettlementTime int64  `gorm:"column:settlement_time;type:bigint;not null"` // 存 Unix 时间戳
}

// TableName 显式指定 GORM 应使用的表名。
func (SettlementModel) TableName() string {
	return "settlements"
}

// SettlementRepositoryImpl 是 SettlementRepository 接口的 GORM 实现。
type SettlementRepositoryImpl struct {
	db *gorm.DB // 依赖注入数据库连接实例
}

// NewSettlementRepository 是 SettlementRepositoryImpl 的构造函数。
func NewSettlementRepository(database *gorm.DB) domain.SettlementRepository {
	return &SettlementRepositoryImpl{
		db: database,
	}
}

// Save 实现了保存清算记录的接口。
// 它将领域实体转换为数据库模型，然后执行数据库插入操作。
func (sr *SettlementRepositoryImpl) Save(ctx context.Context, settlement *domain.Settlement) error {
	// 将领域实体 (domain.Settlement) 转换为数据库模型 (SettlementModel)
	model := &SettlementModel{
		Model:          settlement.Model,
		SettlementID:   settlement.SettlementID,
		TradeID:        settlement.TradeID,
		BuyUserID:      settlement.BuyUserID,
		SellUserID:     settlement.SellUserID,
		Symbol:         settlement.Symbol,
		Quantity:       settlement.Quantity.String(),
		Price:          settlement.Price.String(),
		Status:         settlement.Status,
		SettlementTime: settlement.SettlementTime.Unix(),
	}

	// 使用 GORM 的 Create 方法将记录插入数据库
	if err := sr.db.WithContext(ctx).Create(model).Error; err != nil {
		logging.Error(ctx, "Failed to save settlement",
			"settlement_id", settlement.SettlementID,
			"error", err,
		)
		return fmt.Errorf("failed to save settlement: %w", err)
	}

	// 回填 GORM 生成的 ID 和时间戳到领域实体中
	settlement.Model = model.Model
	return nil
}

// Get 实现了根据 ID 获取清算记录的接口。
func (sr *SettlementRepositoryImpl) Get(ctx context.Context, settlementID string) (*domain.Settlement, error) {
	var model SettlementModel

	// 使用 GORM 的 First 方法查询记录
	if err := sr.db.WithContext(ctx).Where("settlement_id = ?", settlementID).First(&model).Error; err != nil {
		// 如果记录未找到，返回 nil, nil，符合业务预期
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logging.Error(ctx, "Failed to get settlement",
			"settlement_id", settlementID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get settlement: %w", err)
	}

	// 将查询到的数据库模型转换为领域实体
	return sr.modelToDomain(&model), nil
}

// GetByUser 实现了分页获取用户清算历史的接口。
func (sr *SettlementRepositoryImpl) GetByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Settlement, int64, error) {
	var models []SettlementModel
	var total int64

	query := sr.db.WithContext(ctx).Where("buy_user_id = ? OR sell_user_id = ?", userID, userID)

	// 先计算总数
	if err := query.Model(&SettlementModel{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count settlements: %w", err)
	}

	// 再执行分页查询
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&models).Error; err != nil {
		logging.Error(ctx, "Failed to get settlements by user",
			"user_id", userID,
			"error", err,
		)
		return nil, 0, fmt.Errorf("failed to get settlements by user: %w", err)
	}

	// 批量将数据库模型转换为领域实体
	settlements := make([]*domain.Settlement, 0, len(models))
	for i := range models {
		settlements = append(settlements, sr.modelToDomain(&models[i]))
	}

	return settlements, total, nil
}

// GetByTrade 实现了根据交易 ID 获取清算记录的接口。
func (sr *SettlementRepositoryImpl) GetByTrade(ctx context.Context, tradeID string) (*domain.Settlement, error) {
	var model SettlementModel

	if err := sr.db.WithContext(ctx).Where("trade_id = ?", tradeID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logging.Error(ctx, "Failed to get settlement by trade",
			"trade_id", tradeID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get settlement by trade: %w", err)
	}

	return sr.modelToDomain(&model), nil
}

// modelToDomain 是一个辅助函数，用于将数据库模型 (SettlementModel) 转换为领域实体 (domain.Settlement)。
// 这是保持领域层纯粹性的关键。
func (sr *SettlementRepositoryImpl) modelToDomain(model *SettlementModel) *domain.Settlement {
	// 从字符串安全地转换回 decimal.Decimal 类型
	quantity, _ := decimal.NewFromString(model.Quantity)
	price, _ := decimal.NewFromString(model.Price)

	return &domain.Settlement{
		Model:          model.Model,
		SettlementID:   model.SettlementID,
		TradeID:        model.TradeID,
		BuyUserID:      model.BuyUserID,
		SellUserID:     model.SellUserID,
		Symbol:         model.Symbol,
		Quantity:       quantity,
		Price:          price,
		Status:         model.Status,
		SettlementTime: time.Unix(model.SettlementTime, 0),
		CreatedAt:      model.CreatedAt,
	}
}

// EODClearingModel 是日终清算的数据库模型。
type EODClearingModel struct {
	gorm.Model
	ClearingID    string `gorm:"column:clearing_id;type:varchar(50);uniqueIndex;not null"`
	ClearingDate  string `gorm:"column:clearing_date;type:varchar(20);index;not null"`
	Status        string `gorm:"column:status;type:varchar(20);index;not null"`
	StartTime     int64  `gorm:"column:start_time;type:bigint;not null"`
	EndTime       *int64 `gorm:"column:end_time;type:bigint"` // 使用指针以支持 NULL 值
	TradesSettled int64  `gorm:"column:trades_settled;type:bigint;not null"`
	TotalTrades   int64  `gorm:"column:total_trades;type:bigint;not null"`
}

// 指定表名。
func (EODClearingModel) TableName() string {
	return "eod_clearings"
}

// EODClearingRepositoryImpl 是 EODClearingRepository 接口的 GORM 实现。
type EODClearingRepositoryImpl struct {
	db *gorm.DB
}

// NewEODClearingRepository 是 EODClearingRepositoryImpl 的构造函数。
func NewEODClearingRepository(database *gorm.DB) domain.EODClearingRepository {
	return &EODClearingRepositoryImpl{
		db: database,
	}
}

// Save 实现了保存日终清算任务的接口。
func (ecr *EODClearingRepositoryImpl) Save(ctx context.Context, clearing *domain.EODClearing) error {
	// 领域实体到数据库模型的转换
	model := &EODClearingModel{
		Model:         clearing.Model,
		ClearingID:    clearing.ClearingID,
		ClearingDate:  clearing.ClearingDate,
		Status:        clearing.Status,
		StartTime:     clearing.StartTime.Unix(),
		TradesSettled: clearing.TradesSettled,
		TotalTrades:   clearing.TotalTrades,
	}

	if clearing.EndTime != nil {
		endTime := clearing.EndTime.Unix()
		model.EndTime = &endTime
	}

	if err := ecr.db.WithContext(ctx).Create(model).Error; err != nil {
		logging.Error(ctx, "Failed to save EOD clearing",
			"clearing_id", clearing.ClearingID,
			"error", err,
		)
		return fmt.Errorf("failed to save EOD clearing: %w", err)
	}

	clearing.Model = model.Model
	return nil
}

// Get 实现了根据 ID 获取日终清算任务的接口。
func (ecr *EODClearingRepositoryImpl) Get(ctx context.Context, clearingID string) (*domain.EODClearing, error) {
	var model EODClearingModel

	if err := ecr.db.WithContext(ctx).Where("clearing_id = ?", clearingID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logging.Error(ctx, "Failed to get EOD clearing",
			"clearing_id", clearingID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get EOD clearing: %w", err)
	}

	return ecr.modelToDomain(&model), nil
}

// GetLatest 实现了获取最新一次日终清算任务的接口。
func (ecr *EODClearingRepositoryImpl) GetLatest(ctx context.Context) (*domain.EODClearing, error) {
	var model EODClearingModel

	if err := ecr.db.WithContext(ctx).Order("created_at DESC").First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logging.Error(ctx, "Failed to get latest EOD clearing", "error", err)
		return nil, fmt.Errorf("failed to get latest EOD clearing: %w", err)
	}

	return ecr.modelToDomain(&model), nil
}

// Update 实现了更新日终清算任务的接口。
func (ecr *EODClearingRepositoryImpl) Update(ctx context.Context, clearing *domain.EODClearing) error {
	// 使用 map 构建更新字段，只会更新非零值字段，更安全。
	updates := map[string]any{
		"status":         clearing.Status,
		"trades_settled": clearing.TradesSettled,
		"total_trades":   clearing.TotalTrades,
	}

	if clearing.EndTime != nil {
		updates["end_time"] = clearing.EndTime.Unix()
	}

	if err := ecr.db.WithContext(ctx).Model(&EODClearingModel{}).Where("clearing_id = ?", clearing.ClearingID).Updates(updates).Error; err != nil {
		logging.Error(ctx, "Failed to update EOD clearing",
			"clearing_id", clearing.ClearingID,
			"error", err,
		)
		return fmt.Errorf("failed to update EOD clearing: %w", err)
	}

	return nil
}

// modelToDomain 是一个辅助函数，用于将 EODClearingModel 转换为 domain.EODClearing。
func (ecr *EODClearingRepositoryImpl) modelToDomain(model *EODClearingModel) *domain.EODClearing {
	var endTime *time.Time
	if model.EndTime != nil {
		t := time.Unix(*model.EndTime, 0)
		endTime = &t
	}

	return &domain.EODClearing{
		Model:         model.Model,
		ClearingID:    model.ClearingID,
		ClearingDate:  model.ClearingDate,
		Status:        model.Status,
		StartTime:     time.Unix(model.StartTime, 0),
		EndTime:       endTime,
		TradesSettled: model.TradesSettled,
		TotalTrades:   model.TotalTrades,
	}
}
