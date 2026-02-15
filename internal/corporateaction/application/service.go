// Package application 公司行动应用服务
package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/corporateaction/domain"
	"github.com/wyfcoding/pkg/idgen"
)

// PositionService 持仓服务接口 (External Dependency)
type PositionService interface {
	GetPosition(ctx context.Context, accountID, symbol string, date time.Time) (decimal.Decimal, error)
	ListHolders(ctx context.Context, symbol string, date time.Time) ([]string, error)
}

// CashService 资金服务接口 (External Dependency)
type CashService interface {
	Deposit(ctx context.Context, accountID string, amount decimal.Decimal, currency string, refID string) error
}

// SecurityService 证券服务接口 (External Dependency)
type SecurityService interface {
	AdjustPosition(ctx context.Context, accountID, symbol string, delta decimal.Decimal, refID string) error
}

type CorporateActionService struct {
	actionRepo      domain.ActionRepository
	entitlementRepo domain.EntitlementRepository
	positionSvc     PositionService
	cashSvc         CashService
	securitySvc     SecurityService
	logger          *slog.Logger
}

func NewCorporateActionService(
	actionRepo domain.ActionRepository,
	entitlementRepo domain.EntitlementRepository,
	positionSvc PositionService,
	cashSvc CashService,
	securitySvc SecurityService,
	logger *slog.Logger,
) *CorporateActionService {
	return &CorporateActionService{
		actionRepo:      actionRepo,
		entitlementRepo: entitlementRepo,
		positionSvc:     positionSvc,
		cashSvc:         cashSvc,
		securitySvc:     securitySvc,
		logger:          logger.With("module", "corporate_action_service"),
	}
}

// AnnounceAction 发布公司行动
func (s *CorporateActionService) AnnounceAction(ctx context.Context, cmd CreateActionCmd) (string, error) {
	eventID := fmt.Sprintf("CA%s", idgen.GenIDString())
	action := domain.NewCorporateAction(eventID, cmd.Symbol, cmd.Type)
	
	action.AnnouncementDate = time.Now()
	action.ExDate = cmd.ExDate
	action.RecordDate = cmd.RecordDate
	action.PaymentDate = cmd.PaymentDate
	action.RatioNumerator = cmd.RatioNumerator
	action.RatioDenominator = cmd.RatioDenominator
	action.Currency = cmd.Currency
	action.Description = cmd.Description

	if err := s.actionRepo.Save(ctx, action); err != nil {
		return "", err
	}
	return eventID, nil
}

// CalculateEntitlements 计算权益 (通常在Record Date后执行)
func (s *CorporateActionService) CalculateEntitlements(ctx context.Context, eventID string) error {
	s.logger.InfoContext(ctx, "calculating entitlements", "event_id", eventID)
	
	action, err := s.actionRepo.GetByEventID(ctx, eventID)
	if err != nil {
		return err
	}

	// 获取所有持仓用户
	holders, err := s.positionSvc.ListHolders(ctx, action.Symbol, action.RecordDate)
	if err != nil {
		return err
	}

	for _, holderID := range holders {
		qty, err := s.positionSvc.GetPosition(ctx, holderID, action.Symbol, action.RecordDate)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to get position", "holder", holderID, "error", err)
			continue
		}

		ent, err := action.CalculateEntitlement(holderID, qty)
		if err != nil {
			s.logger.WarnContext(ctx, "failed to calc entitlement", "holder", holderID, "error", err)
			continue
		}
		
		ent.EntitlementID = fmt.Sprintf("ENT%s", idgen.GenIDString())
		if err := s.entitlementRepo.Save(ctx, ent); err != nil {
			s.logger.ErrorContext(ctx, "failed to save entitlement", "holder", holderID, "error", err)
		}
	}

	action.Status = domain.ActionStatusProcessed
	return s.actionRepo.Save(ctx, action)
}

// ProcessPayments 执行支付 (在Payment Date执行)
func (s *CorporateActionService) ProcessPayments(ctx context.Context, eventID string) error {
	s.logger.InfoContext(ctx, "processing payments", "event_id", eventID)

	action, err := s.actionRepo.GetByEventID(ctx, eventID)
	if err != nil {
		return err
	}

	entitlements, err := s.entitlementRepo.ListByActionID(ctx, action.ID)
	if err != nil {
		return err
	}

	for _, ent := range entitlements {
		if ent.Status == "PAID" {
			continue
		}

		// 支付现金
		if ent.PayoutCash.IsPositive() {
			err := s.cashSvc.Deposit(ctx, ent.AccountID, ent.PayoutCash, action.Currency, ent.EntitlementID)
			if err != nil {
				s.logger.ErrorContext(ctx, "cash payment failed", "ent_id", ent.EntitlementID, "error", err)
				ent.Status = "FAILED"
				_ = s.entitlementRepo.Save(ctx, ent)
				continue
			}
		}

		// 支付股票 (送股/拆股)
		if ent.PayoutStock.IsPositive() {
			err := s.securitySvc.AdjustPosition(ctx, ent.AccountID, ent.StockSymbol, ent.PayoutStock, ent.EntitlementID)
			if err != nil {
				s.logger.ErrorContext(ctx, "stock payment failed", "ent_id", ent.EntitlementID, "error", err)
				ent.Status = "FAILED"
				_ = s.entitlementRepo.Save(ctx, ent)
				continue
			}
		}

		ent.Status = "PAID"
		_ = s.entitlementRepo.Save(ctx, ent)
	}

	action.Status = domain.ActionStatusCompleted
	return s.actionRepo.Save(ctx, action)
}

type CreateActionCmd struct {
	Symbol           string
	Type             domain.ActionType
	ExDate           time.Time
	RecordDate       time.Time
	PaymentDate      time.Time
	RatioNumerator   decimal.Decimal
	RatioDenominator decimal.Decimal
	Currency         string
	Description      string
}
