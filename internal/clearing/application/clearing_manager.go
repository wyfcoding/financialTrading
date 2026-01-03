package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	accountv1 "github.com/wyfcoding/financialtrading/goapi/account/v1"
	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
	"github.com/wyfcoding/pkg/dtm"
	"github.com/wyfcoding/pkg/idgen"
)

// ClearingManager 处理所有清算相关的写入操作（Commands）。
type ClearingManager struct {
	settlementRepo domain.SettlementRepository
	eodRepo        domain.EODClearingRepository
	marginRepo     domain.MarginRequirementRepository
	accountCli     accountv1.AccountServiceClient
	logger         *slog.Logger
	dtmServer      string
	accountSvcURL  string // Account 服务 gRPC 地址
}

// NewClearingManager 构造函数。
func NewClearingManager(
	settlementRepo domain.SettlementRepository,
	eodRepo domain.EODClearingRepository,
	marginRepo domain.MarginRequirementRepository,
	logger *slog.Logger,
) *ClearingManager {
	return &ClearingManager{
		settlementRepo: settlementRepo,
		eodRepo:        eodRepo,
		marginRepo:     marginRepo,
		logger:         logger.With("module", "clearing_manager"),
	}
}

func (m *ClearingManager) SetAccountClient(cli accountv1.AccountServiceClient, svcURL string) {
	m.accountCli = cli
	m.accountSvcURL = svcURL
}

func (m *ClearingManager) SetDTMServer(addr string) {
	m.dtmServer = addr
}

// SettleTrade 清算单笔交易 (gRPC 外部调用入口，已改为内部重定向到分布式事务逻辑)
func (m *ClearingManager) SettleTrade(ctx context.Context, req *SettleTradeRequest) (string, error) {
	event := map[string]any{
		"trade_id":     req.TradeID,
		"buy_user_id":  req.BuyUserID,
		"sell_user_id": req.SellUserID,
		"symbol":       req.Symbol,
		"quantity":     req.Quantity,
		"price":        req.Price,
	}
	err := m.ProcessTradeExecution(ctx, event)
	return "", err
}

// ProcessTradeExecution 处理来自消息队列的成交事件 (分布式事务核心编排)
func (m *ClearingManager) ProcessTradeExecution(ctx context.Context, event map[string]any) error {
	tradeID := event["trade_id"].(string)
	buyUserID := event["buy_user_id"].(string)
	sellUserID := event["sell_user_id"].(string)
	symbol := event["symbol"].(string)
	quantityStr := event["quantity"].(string)
	priceStr := event["price"].(string)

	m.logger.Info("orchestrating saga settlement for trade", "trade_id", tradeID, "symbol", symbol)

	// 1. 幂等性预检
	existing, _ := m.settlementRepo.GetByTrade(ctx, tradeID)
	if existing != nil && existing.Status == domain.SettlementStatusCompleted {
		m.logger.Info("trade already settled", "trade_id", tradeID)
		return nil
	}

	// 2. 初始化 DTM Saga 事务
	gid := fmt.Sprintf("SAGA-SETTLE-%s", tradeID)
	saga := dtm.NewSaga(ctx, m.dtmServer, gid)

	accountSvc := m.accountSvcURL + "/api.account.v1.AccountService"

	// 步骤 A: 买方扣除冻结资金 (USDT)
	buyReq := &accountv1.SagaAccountRequest{
		UserId:   buyUserID,
		Currency: "USDT",
		Amount:   m.calcAmount(quantityStr, priceStr),
		TradeId:  tradeID,
	}
	saga.Add(
		accountSvc+"/SagaDeductFrozen",
		accountSvc+"/SagaRefundFrozen",
		buyReq,
	)

	// 步骤 B: 卖方增加余额 (USDT)
	sellReq := &accountv1.SagaAccountRequest{
		UserId:   sellUserID,
		Currency: "USDT",
		Amount:   m.calcAmount(quantityStr, priceStr),
		TradeId:  tradeID,
	}
	saga.Add(
		accountSvc+"/SagaAddBalance",
		accountSvc+"/SagaSubBalance",
		sellReq,
	)

	// 3. 提交 Saga 事务
	if err := saga.Submit(); err != nil {
		m.logger.Error("failed to submit saga settlement", "gid", gid, "error", err)
		return fmt.Errorf("saga submission failed: %w", err)
	}

	// 4. 同步更新本地状态 (最终一致性：此处即使失败，Saga 也会在后台完成)
	// 在生产环境中，通常会有一个专门的消息监听器监听 Saga 完成事件来更新此状态
	m.logger.Info("saga settlement submitted", "gid", gid)
	
	// 为了即时反馈，我们先记录为 PENDING，后续由 Saga 驱动或对账修正
	qty, _ := decimal.NewFromString(quantityStr)
	prc, _ := decimal.NewFromString(priceStr)
	_ = m.settlementRepo.Save(ctx, &domain.Settlement{
		SettlementID:   gid,
		TradeID:        tradeID,
		BuyUserID:      buyUserID,
		SellUserID:     sellUserID,
		Symbol:         symbol,
		Quantity:       qty,
		Price:          prc,
		Status:         domain.SettlementStatusPending,
		SettlementTime: time.Now(),
	})

	return nil
}

func (m *ClearingManager) calcAmount(q, p string) string {
	qd, _ := decimal.NewFromString(q)
	pd, _ := decimal.NewFromString(p)
	return qd.Mul(pd).String()
}

// ExecuteEODClearing 执行日终清算
func (m *ClearingManager) ExecuteEODClearing(ctx context.Context, clearingDate string) (string, error) {
	clearingID := fmt.Sprintf("EOD-%d", idgen.GenID())

	clearing := &domain.EODClearing{
		ClearingID:    clearingID,
		ClearingDate:  clearingDate,
		Status:        domain.ClearingStatusProcessing,
		StartTime:     time.Now(),
		TradesSettled: 0,
		TotalTrades:   0,
	}

	if err := m.eodRepo.Save(ctx, clearing); err != nil {
		return "", err
	}

	return clearingID, nil
}

// GetMarginRequirement 获取指定品种的实时保证金要求
func (m *ClearingManager) GetMarginRequirement(ctx context.Context, symbol string) (*domain.MarginRequirement, error) {
	margin, err := m.marginRepo.GetBySymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}
	if margin == nil {
		// 默认兜底：10% 基础保证金率，0% 波动率加成
		return &domain.MarginRequirement{
			Symbol:           symbol,
			BaseMarginRate:   decimal.NewFromFloat(0.1),
			VolatilityFactor: decimal.Zero,
		}, nil
	}
	return margin, nil
}
