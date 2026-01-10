// Package application 提供了清算模块的业务逻辑编排。
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
	settlementRepo domain.SettlementRepository        // 结算记录仓储
	eodRepo        domain.EODClearingRepository       // 日终清算仓储
	marginRepo     domain.MarginRequirementRepository // 保证金规则仓储
	accountCli     accountv1.AccountServiceClient     // 账户服务客户端
	positionCli    positionv1.PositionServiceClient   // 持仓服务客户端
	logger         *slog.Logger                       // 结构化日志记录器
	dtmServer      string                             // DTM 服务端地址
	accountSvcURL  string                             // Account 服务 gRPC 基地址
	positionSvcURL string                             // Position 服务 gRPC 基地址
	clearingSvcURL string                             // Clearing 服务 gRPC 基地址 (回调使用)
}

// NewClearingManager 构造一个新的清算管理器实例。
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

// SetAccountClient 注入账户服务客户端及其访问路径。
func (m *ClearingManager) SetAccountClient(cli accountv1.AccountServiceClient, svcURL string) {
	m.accountCli = cli
	m.accountSvcURL = svcURL
}

// SetPositionClient 注入持仓服务客户端及其访问路径。
func (m *ClearingManager) SetPositionClient(cli positionv1.PositionServiceClient, svcURL string) {
	m.positionCli = cli
	m.positionSvcURL = svcURL
}

// SetDTMServer 配置分布式事务协调器地址。
func (m *ClearingManager) SetDTMServer(addr string) {
	m.dtmServer = addr
}

// SetSvcURL 配置当前服务的基地址，用于 Saga 回调注册。
func (m *ClearingManager) SetSvcURL(url string) {
	m.clearingSvcURL = url
}

// SagaMarkSettlementCompleted Saga 正向阶段：确认结算流程已合规终结。
func (m *ClearingManager) SagaMarkSettlementCompleted(ctx context.Context, settlementID string) error {
	settlement, err := m.settlementRepo.Get(ctx, settlementID)
	if err != nil || settlement == nil {
		return fmt.Errorf("settlement record not found: %s", settlementID)
	}
	settlement.Status = domain.SettlementStatusCompleted
	return m.settlementRepo.Save(ctx, settlement)
}

// SagaMarkSettlementFailed Saga 补偿阶段：将处于中间态的结算标记为失败。
func (m *ClearingManager) SagaMarkSettlementFailed(ctx context.Context, settlementID string, _ string) error {
	settlement, err := m.settlementRepo.Get(ctx, settlementID)
	if err != nil || settlement == nil {
		return fmt.Errorf("settlement record not found: %s", settlementID)
	}
	settlement.Status = domain.SettlementStatusFailed
	return m.settlementRepo.Save(ctx, settlement)
}

// SettleTrade 清算单笔交易 (gRPC 外部调用入口)。
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

// ProcessTradeExecution 处理来自成交引擎的异步成交事件，执行分布式 Saga 清算。
// 架构逻辑：
// 1. 本地幂等初始化记录。
// 2. 编排 Saga：对冲资金（买卖双方）-> 对冲持仓（买卖双方）-> 终结状态。
func (m *ClearingManager) ProcessTradeExecution(ctx context.Context, event map[string]any) error {
	tradeID, _ := event["trade_id"].(string)
	buyUserID, _ := event["buy_user_id"].(string)
	sellUserID, _ := event["sell_user_id"].(string)
	symbol, _ := event["symbol"].(string)
	quantityStr, _ := event["quantity"].(string)
	priceStr, _ := event["price"].(string)

	m.logger.InfoContext(ctx, "orchestrating trade settlement saga", "trade_id", tradeID, "symbol", symbol)

	existing, _ := m.settlementRepo.GetByTrade(ctx, tradeID)
	if existing != nil && existing.Status == domain.SettlementStatusCompleted {
		return nil
	}

	gid := fmt.Sprintf("SAGA-SETTLE-%s", tradeID)
	if existing == nil {
		qty, err := decimal.NewFromString(quantityStr)
		if err != nil {
			return fmt.Errorf("invalid qty: %w", err)
		}
		prc, err := decimal.NewFromString(priceStr)
		if err != nil {
			return fmt.Errorf("invalid prc: %w", err)
		}
		err = m.settlementRepo.Save(ctx, &domain.Settlement{
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
			return fmt.Errorf("failed to init local settlement: %w", err)
		}
	}

	saga := dtm.NewSaga(ctx, m.dtmServer, gid)
	accountSvc := m.accountSvcURL + "/api.account.v1.AccountService"
	positionSvc := m.positionSvcURL + "/api.position.v1.PositionService"
	clearingSvc := m.clearingSvcURL + "/api.clearing.v1.ClearingService"

	statusReq := &clearingv1.SagaSettlementRequest{SettlementId: gid, TradeId: tradeID}

	// 注册 Saga 步骤
	saga.Add("", clearingSvc+"/SagaMarkSettlementFailed", statusReq)

	amount, err := m.calcAmount(quantityStr, priceStr)
	if err != nil {
		return err
	}

	// 资金流转
	saga.Add(accountSvc+"/SagaDeductFrozen", accountSvc+"/SagaRefundFrozen", &accountv1.SagaAccountRequest{
		UserId: buyUserID, Currency: "USDT", Amount: amount, TradeId: tradeID,
	})
	saga.Add(accountSvc+"/SagaAddBalance", accountSvc+"/SagaSubBalance", &accountv1.SagaAccountRequest{
		UserId: sellUserID, Currency: "USDT", Amount: amount, TradeId: tradeID,
	})

	// 资产流转
	saga.Add(positionSvc+"/SagaDeductFrozen", positionSvc+"/SagaRefundFrozen", &positionv1.SagaPositionRequest{
		UserId: sellUserID, Symbol: symbol, Quantity: quantityStr, Price: priceStr, TradeId: tradeID,
	})
	saga.Add(positionSvc+"/SagaAddPosition", positionSvc+"/SagaSubPosition", &positionv1.SagaPositionRequest{
		UserId: buyUserID, Symbol: symbol, Quantity: quantityStr, Price: priceStr, TradeId: tradeID,
	})

	saga.Add(clearingSvc+"/SagaMarkSettlementCompleted", "", statusReq)

	if err := saga.Submit(); err != nil {
		m.logger.ErrorContext(ctx, "failed to submit settlement saga", "gid", gid, "error", err)
		return err
	}

	m.logger.InfoContext(ctx, "settlement saga submitted successfully", "gid", gid)
	return nil
}

func (m *ClearingManager) calcAmount(q, p string) (string, error) {
	qd, err := decimal.NewFromString(q)
	if err != nil {
		return "", err
	}
	pd, err := decimal.NewFromString(p)
	if err != nil {
		return "", err
	}
	return qd.Mul(pd).String(), nil
}

// ExecuteEODClearing 启动日终清算流程。
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
		m.logger.ErrorContext(ctx, "failed to initiate EOD clearing", "date", clearingDate, "error", err)
		return "", err
	}

	m.logger.InfoContext(ctx, "EOD clearing initiated", "clearing_id", clearingID, "date", clearingDate)
	return clearingID, nil
}

// GetMarginRequirement 实时获取并计算品种保证金率。
func (m *ClearingManager) GetMarginRequirement(ctx context.Context, symbol string) (*domain.MarginRequirement, error) {
	margin, err := m.marginRepo.GetBySymbol(ctx, symbol)
	if err != nil {
		m.logger.ErrorContext(ctx, "failed to fetch margin requirement", "symbol", symbol, "error", err)
		return nil, err
	}
	if margin == nil {
		return &domain.MarginRequirement{
			Symbol:           symbol,
			BaseMarginRate:   decimal.NewFromFloat(0.1),
			VolatilityFactor: decimal.Zero,
		}, nil
	}

	m.logger.DebugContext(ctx, "margin requirement fetched", "symbol", symbol, "rate", margin.BaseMarginRate.String())
	return margin, nil
}
