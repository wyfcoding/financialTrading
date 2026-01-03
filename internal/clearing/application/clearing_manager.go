package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	accountv1 "github.com/wyfcoding/financialtrading/goapi/account/v1"
	clearingv1 "github.com/wyfcoding/financialtrading/goapi/clearing/v1"
	positionv1 "github.com/wyfcoding/financialtrading/goapi/position/v1"
	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
	"github.com/wyfcoding/pkg/dtm"
	"github.com/wyfcoding/pkg/idgen"
)

// ClearingManager 处理所有清算相关的写入操作（Commands）。
type ClearingManager struct {
	settlementRepo  domain.SettlementRepository
	eodRepo         domain.EODClearingRepository
	marginRepo      domain.MarginRequirementRepository
	accountCli      accountv1.AccountServiceClient
	positionCli     positionv1.PositionServiceClient
	logger          *slog.Logger
	dtmServer       string
	accountSvcURL   string // Account 服务 gRPC 地址
	positionSvcURL  string // Position 服务 gRPC 地址
	clearingSvcURL  string // 本服务 gRPC 地址 (用于 Saga 回调)
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

func (m *ClearingManager) SetPositionClient(cli positionv1.PositionServiceClient, svcURL string) {
	m.positionCli = cli
	m.positionSvcURL = svcURL
}

func (m *ClearingManager) SetDTMServer(addr string) {
	m.dtmServer = addr
}

func (m *ClearingManager) SetSvcURL(url string) {
	m.clearingSvcURL = url
}

// SagaMarkSettlementCompleted Saga 正向: 确认结算成功
func (m *ClearingManager) SagaMarkSettlementCompleted(ctx context.Context, settlementID string) error {
	settlement, err := m.settlementRepo.Get(ctx, settlementID)
	if err != nil || settlement == nil {
		return fmt.Errorf("settlement not found: %s", settlementID)
	}
	settlement.Status = domain.SettlementStatusCompleted
	return m.settlementRepo.Save(ctx, settlement)
}

// SagaMarkSettlementFailed Saga 补偿: 标记结算失败
func (m *ClearingManager) SagaMarkSettlementFailed(ctx context.Context, settlementID string, reason string) error {
	settlement, err := m.settlementRepo.Get(ctx, settlementID)
	if err != nil || settlement == nil {
		return fmt.Errorf("settlement not found: %s", settlementID)
	}
	settlement.Status = domain.SettlementStatusFailed
	return m.settlementRepo.Save(ctx, settlement)
}

// SettleTrade 清算单笔交易 (gRPC 外部调用入口)
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

// ProcessTradeExecution 处理来自消息队列的成交事件 (生产级全闭环 Saga)
func (m *ClearingManager) ProcessTradeExecution(ctx context.Context, event map[string]any) error {
	tradeID := event["trade_id"].(string)
	buyUserID := event["buy_user_id"].(string)
	sellUserID := event["sell_user_id"].(string)
	symbol := event["symbol"].(string)
	quantityStr := event["quantity"].(string)
	priceStr := event["price"].(string)

	m.logger.Info("orchestrating full closed-loop saga settlement", "trade_id", tradeID, "symbol", symbol)

	// 1. 幂等性预检 (若已完成则跳过)
	existing, _ := m.settlementRepo.GetByTrade(ctx, tradeID)
	if existing != nil && existing.Status == domain.SettlementStatusCompleted {
		return nil
	}

	// 2. 本地初始化结算记录 (状态：PENDING)
	// 此操作必须先于 Saga 提交，确保状态同步桩有据可循
	gid := fmt.Sprintf("SAGA-SETTLE-%s", tradeID)
	if existing == nil {
		qty, _ := decimal.NewFromString(quantityStr)
		prc, _ := decimal.NewFromString(priceStr)
		err := m.settlementRepo.Save(ctx, &domain.Settlement{
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
		if err != nil {
			return fmt.Errorf("failed to init local settlement record: %w", err)
		}
	}

	// 3. 构建闭环 Saga 事务
	saga := dtm.NewSaga(ctx, m.dtmServer, gid)

	accountSvc := m.accountSvcURL + "/api.account.v1.AccountService"
	positionSvc := m.positionSvcURL + "/api.position.v1.PositionService"
	clearingSvc := m.clearingSvcURL + "/api.clearing.v1.ClearingService"

	statusReq := &clearingv1.SagaSettlementRequest{
		SettlementId: gid,
		TradeId:      tradeID,
	}

	// [关键优化] 步骤 0: 状态跟踪桩
	// 正向为空，补偿为 MarkFailed。
	// 一旦后面任何一步失败，DTM 会自动调用此补偿，将 PENDING 改为 FAILED。
	saga.Add(
		"", 
		clearingSvc+"/SagaMarkSettlementFailed",
		statusReq,
	)

	// 步骤 1 & 2: 资金对冲
	saga.Add(accountSvc+"/SagaDeductFrozen", accountSvc+"/SagaRefundFrozen", &accountv1.SagaAccountRequest{
		UserId: buyUserID, Currency: "USDT", Amount: m.calcAmount(quantityStr, priceStr),
		TradeId: tradeID,
	})
	saga.Add(accountSvc+"/SagaAddBalance", accountSvc+"/SagaSubBalance", &accountv1.SagaAccountRequest{
		UserId: sellUserID, Currency: "USDT", Amount: m.calcAmount(quantityStr, priceStr),
		TradeId: tradeID,
	})

	// 步骤 3: 卖方扣除冻结持仓 (Base Asset, e.g., BTC)
	saga.Add(
		positionSvc+"/SagaDeductFrozen",
		positionSvc+"/SagaRefundFrozen",
		&positionv1.SagaPositionRequest{
			UserId:   sellUserID,
			Symbol:   symbol,
			Quantity: quantityStr,
			Price:    priceStr, // 传递成交价用于盈亏计算
			TradeId:  tradeID,
		},
	)
	// 步骤 4: 买方增加持仓 (Base Asset, e.g., BTC)
	saga.Add(
		positionSvc+"/SagaAddPosition",
		positionSvc+"/SagaSubPosition",
		&positionv1.SagaPositionRequest{
			UserId:   buyUserID,
			Symbol:   symbol,
			Quantity: quantityStr,
			Price:    priceStr, // 传递成交价
			TradeId:  tradeID,
		},
	)

	// [关键优化] 步骤 5: 成功终结桩
	// 只有当上述所有资金、资产对冲都成功后，才会执行此正向操作，将 PENDING 改为 COMPLETED。
	saga.Add(
		clearingSvc+"/SagaMarkSettlementCompleted",
		"",
		statusReq,
	)

	// 4. 提交 Saga
	if err := saga.Submit(); err != nil {
		m.logger.Error("failed to submit full-loop saga settlement", "gid", gid, "error", err)
		return err
	}

	m.logger.Info("full-loop saga settlement submitted successfully", "gid", gid)
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
		return &domain.MarginRequirement{
			Symbol:           symbol,
			BaseMarginRate:   decimal.NewFromFloat(0.1),
			VolatilityFactor: decimal.Zero,
		}, nil
	}
	return margin, nil
}
