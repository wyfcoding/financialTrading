// Package application 包含清算服务的用例逻辑(Use Cases)。
// 这一层负责编排领域对象和仓储，以完成具体的业务功能。
package application

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialTrading/internal/clearing/domain"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/logging"
)

// SettleTradeRequest 是清算交易请求的数据传输对象 (DTO - Data Transfer Object)。
// 它用于从接口层（如 gRPC handler）向应用层传递参数，与领域模型解耦。
type SettleTradeRequest struct {
	TradeID    string // 交易 ID
	BuyUserID  string // 买方用户 ID
	SellUserID string // 卖方用户 ID
	Symbol     string // 交易对符号
	Quantity   string // 数量 (使用字符串以保持精度)
	Price      string // 价格 (使用字符串以保持精度)
}

// ClearingApplicationService 是清算应用服务。
// 它封装了清算相关的所有业务用例，是业务逻辑的核心协调者。
// 它通过依赖注入的方式引用了领域层的仓储接口。
type ClearingApplicationService struct {
	settlementRepo domain.SettlementRepository  // 结算仓储接口，用于持久化结算信息
	eodRepo        domain.EODClearingRepository // 日终清算仓储接口，用于持久化日终清算任务
}

// NewClearingApplicationService 是 ClearingApplicationService 的构造函数。
//
// @param settlementRepo 实现了 SettlementRepository 接口的实例。
// @param eodRepo 实现了 EODClearingRepository 接口的实例。
// @return *ClearingApplicationService 返回一个新的清算应用服务实例。
func NewClearingApplicationService(
	settlementRepo domain.SettlementRepository,
	eodRepo domain.EODClearingRepository,
) *ClearingApplicationService {
	// 初始化雪花ID生成器，传入特定的节点ID（此处为7）。
	// 在分布式系统中，每个服务实例应有唯一的节点ID。
	return &ClearingApplicationService{
		settlementRepo: settlementRepo,
		eodRepo:        eodRepo,
	}
}

// SettleTrade 是清算单笔交易的业务用例。
//
// @param ctx context.Context 用于传递请求上下文，例如 Trace ID。
// @param req *SettleTradeRequest 包含清算所需的数据。
// @return error 如果处理过程中发生错误，则返回错误信息。
func (cas *ClearingApplicationService) SettleTrade(ctx context.Context, req *SettleTradeRequest) (string, error) {
	// 1. 输入参数校验
	if req.TradeID == "" || req.BuyUserID == "" || req.SellUserID == "" {
		return "", fmt.Errorf("invalid request parameters: trade_id, buy_user_id, and sell_user_id are required")
	}

	// 2. 数据转换和校验
	// 使用 decimal 包处理高精度的货币计算，避免浮点数误差。
	quantity, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		return "", fmt.Errorf("invalid quantity format: %w", err)
	}

	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		return "", fmt.Errorf("invalid price format: %w", err)
	}

	// 3. 生成唯一的清算ID
	settlementID := fmt.Sprintf("SETTLE-%d", idgen.GenID())

	// 4. 创建领域对象
	// 这是业务逻辑的核心部分，在这里可以添加更复杂的领域规则，
	// 例如检查账户余额、更新头寸、计算手续费等。
	// 当前实现为简化版，直接创建清算完成的记录。
	settlement := &domain.Settlement{
		SettlementID:   settlementID,
		TradeID:        req.TradeID,
		BuyUserID:      req.BuyUserID,
		SellUserID:     req.SellUserID,
		Symbol:         req.Symbol,
		Quantity:       quantity,
		Price:          price,
		Status:         domain.SettlementStatusCompleted, // 假设立即完成
		SettlementTime: time.Now(),
		CreatedAt:      time.Now(),
	}

	// 5. 持久化领域对象
	// 调用仓储接口将清算记录保存到数据库。
	// 仓储的具体实现（如 GORM）对应用层是透明的。
	if err := cas.settlementRepo.Save(ctx, settlement); err != nil {
		logging.Error(ctx, "Failed to save settlement",
			"settlement_id", settlementID,
			"trade_id", req.TradeID,
			"error", err,
		)
		return "", fmt.Errorf("failed to save settlement record: %w", err)
	}

	logging.Info(ctx, "Trade settled successfully",
		"settlement_id", settlementID,
		"trade_id", req.TradeID,
	)

	return settlementID, nil
}

// ExecuteEODClearing 是执行日终清算的业务用例。
//
// @param ctx context.Context 请求上下文。
// @param clearingDate string 需要执行清算的日期。
// @return error 如果处理过程中发生错误，则返回错误信息。
func (cas *ClearingApplicationService) ExecuteEODClearing(ctx context.Context, clearingDate string) (string, error) {
	// 1. 生成唯一的日终清算任务ID
	clearingID := fmt.Sprintf("EOD-%d", idgen.GenID())

	// 2. 创建日终清算领域对象
	// 在实际应用中，这里会触发一个长时间运行的后台任务。
	// 任务会查询当天所有未结算的交易，并分批进行处理。
	clearing := &domain.EODClearing{
		ClearingID:    clearingID,
		ClearingDate:  clearingDate,
		Status:        domain.ClearingStatusProcessing, // 设置初始状态为处理中
		StartTime:     time.Now(),
		TradesSettled: 0,
		TotalTrades:   0, // 总交易数可以在任务开始时查询得到
	}

	// 3. 持久化任务状态
	if err := cas.eodRepo.Save(ctx, clearing); err != nil {
		logging.Error(ctx, "Failed to save EOD clearing task",
			"clearing_id", clearingID,
			"error", err,
		)
		return "", fmt.Errorf("failed to save EOD clearing task: %w", err)
	}

	// 4. 触发异步处理（示例）
	// 在真实场景中，可以在这里向消息队列（如 Kafka）发送一个事件，
	// 由后台的 worker 来消费并执行实际的清算逻辑。
	// go cas.processEOD(ctx, clearing)
	logging.Info(ctx, "EOD clearing task started",
		"clearing_id", clearingID,
		"clearing_date", clearingDate,
	)

	return clearingID, nil
}

// GetClearingStatus 是获取日终清算任务状态的业务用例。
//
// @param ctx context.Context 请求上下文。
// @param clearingID string 要查询的清算任务ID。
// @return *domain.EODClearing 返回清算任务的详细信息。
// @return error 如果查询失败，则返回错误。
func (cas *ClearingApplicationService) GetClearingStatus(ctx context.Context, clearingID string) (*domain.EODClearing, error) {
	// 通过仓储接口从数据源获取清算任务的状态。
	clearing, err := cas.eodRepo.Get(ctx, clearingID)
	if err != nil {
		logging.Error(ctx, "Failed to get clearing status",
			"clearing_id", clearingID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get clearing status: %w", err)
	}

	return clearing, nil
}
