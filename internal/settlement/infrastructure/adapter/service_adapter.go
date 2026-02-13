package adapter

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/settlement/domain"
)

type CustodianAdapter struct {
	custodyClient CustodyClient
	logger        *slog.Logger
}

type CustodyClient interface {
	TransferSecurity(ctx context.Context, fromAccount, toAccount, symbol string, quantity int64) error
	TransferCash(ctx context.Context, fromAccount, toAccount string, amount int64, currency string) error
	GetBalance(ctx context.Context, accountID, currency string) (int64, error)
	GetPosition(ctx context.Context, accountID, symbol string) (int64, error)
	Freeze(ctx context.Context, accountID string, amount int64, currency string) error
	Unfreeze(ctx context.Context, accountID string, amount int64, currency string) error
}

func NewCustodianAdapter(client CustodyClient, logger *slog.Logger) domain.CustodianService {
	return &CustodianAdapter{
		custodyClient: client,
		logger:        logger,
	}
}

func (a *CustodianAdapter) TransferSecurity(ctx context.Context, fromAccount, toAccount, symbol string, quantity decimal.Decimal) error {
	a.logger.Info("transferring security",
		"from", fromAccount,
		"to", toAccount,
		"symbol", symbol,
		"quantity", quantity,
	)

	qty := quantity.IntPart()
	if err := a.custodyClient.TransferSecurity(ctx, fromAccount, toAccount, symbol, qty); err != nil {
		a.logger.Error("security transfer failed", "error", err)
		return fmt.Errorf("custody transfer security: %w", err)
	}

	a.logger.Info("security transfer completed", "symbol", symbol, "quantity", qty)
	return nil
}

func (a *CustodianAdapter) TransferCash(ctx context.Context, fromAccount, toAccount string, amount decimal.Decimal, currency string) error {
	a.logger.Info("transferring cash",
		"from", fromAccount,
		"to", toAccount,
		"amount", amount,
		"currency", currency,
	)

	amt := amount.IntPart()
	if err := a.custodyClient.TransferCash(ctx, fromAccount, toAccount, amt, currency); err != nil {
		a.logger.Error("cash transfer failed", "error", err)
		return fmt.Errorf("custody transfer cash: %w", err)
	}

	a.logger.Info("cash transfer completed", "amount", amt, "currency", currency)
	return nil
}

func (a *CustodianAdapter) GetAccountBalance(ctx context.Context, accountID, currency string) (decimal.Decimal, error) {
	balance, err := a.custodyClient.GetBalance(ctx, accountID, currency)
	if err != nil {
		return decimal.Zero, fmt.Errorf("get balance: %w", err)
	}
	return decimal.NewFromInt(balance), nil
}

func (a *CustodianAdapter) GetSecurityPosition(ctx context.Context, accountID, symbol string) (decimal.Decimal, error) {
	position, err := a.custodyClient.GetPosition(ctx, accountID, symbol)
	if err != nil {
		return decimal.Zero, fmt.Errorf("get position: %w", err)
	}
	return decimal.NewFromInt(position), nil
}

func (a *CustodianAdapter) FreezeAccount(ctx context.Context, accountID string, amount decimal.Decimal, currency string) error {
	amt := amount.IntPart()
	if err := a.custodyClient.Freeze(ctx, accountID, amt, currency); err != nil {
		return fmt.Errorf("freeze account: %w", err)
	}
	return nil
}

func (a *CustodianAdapter) UnfreezeAccount(ctx context.Context, accountID string, amount decimal.Decimal, currency string) error {
	amt := amount.IntPart()
	if err := a.custodyClient.Unfreeze(ctx, accountID, amt, currency); err != nil {
		return fmt.Errorf("unfreeze account: %w", err)
	}
	return nil
}

type CCPAdapter struct {
	ccpClient CCPClient
	logger    *slog.Logger
}

type CCPClient interface {
	RegisterTrade(ctx context.Context, tradeID, symbol string, quantity, price int64, buyerAccount, sellerAccount string) error
	CalculateMargin(ctx context.Context, accountID string) (int64, error)
}

func NewCCPAdapter(client CCPClient, logger *slog.Logger) domain.CCPService {
	return &CCPAdapter{
		ccpClient: client,
		logger:    logger,
	}
}

func (a *CCPAdapter) RegisterTrade(ctx context.Context, instruction *domain.SettlementInstruction) error {
	a.logger.Info("registering trade with CCP",
		"instruction_id", instruction.InstructionID,
		"trade_id", instruction.TradeID,
	)

	err := a.ccpClient.RegisterTrade(
		ctx,
		instruction.TradeID,
		instruction.Symbol,
		instruction.Quantity.IntPart(),
		instruction.Price.IntPart(),
		instruction.BuyerAccountID,
		instruction.SellerAccountID,
	)
	if err != nil {
		a.logger.Error("CCP trade registration failed", "error", err)
		return fmt.Errorf("ccp register trade: %w", err)
	}

	a.logger.Info("CCP trade registered", "trade_id", instruction.TradeID)
	return nil
}

func (a *CCPAdapter) CalculateMargin(ctx context.Context, accountID string) (decimal.Decimal, error) {
	margin, err := a.ccpClient.CalculateMargin(ctx, accountID)
	if err != nil {
		return decimal.Zero, fmt.Errorf("calculate margin: %w", err)
	}
	return decimal.NewFromInt(margin), nil
}

type NotificationAdapter struct {
	notifyClient NotificationClient
	logger       *slog.Logger
}

type NotificationClient interface {
	SendNotification(ctx context.Context, accountID, eventType, message string, data map[string]string) error
}

func NewNotificationAdapter(client NotificationClient, logger *slog.Logger) domain.NotificationService {
	return &NotificationAdapter{
		notifyClient: client,
		logger:       logger,
	}
}

func (a *NotificationAdapter) NotifySettlementCreated(ctx context.Context, accountID, instructionID string) error {
	a.logger.Info("sending settlement created notification", "account_id", accountID, "instruction_id", instructionID)

	return a.notifyClient.SendNotification(ctx, accountID, "SETTLEMENT_CREATED",
		"Settlement instruction created",
		map[string]string{"instruction_id": instructionID},
	)
}

func (a *NotificationAdapter) NotifySettlementCompleted(ctx context.Context, accountID, instructionID string) error {
	a.logger.Info("sending settlement completed notification", "account_id", accountID, "instruction_id", instructionID)

	return a.notifyClient.SendNotification(ctx, accountID, "SETTLEMENT_COMPLETED",
		"Settlement instruction completed successfully",
		map[string]string{"instruction_id": instructionID},
	)
}

func (a *NotificationAdapter) NotifySettlementFailed(ctx context.Context, accountID, instructionID, reason string) error {
	a.logger.Info("sending settlement failed notification", "account_id", accountID, "instruction_id", instructionID)

	return a.notifyClient.SendNotification(ctx, accountID, "SETTLEMENT_FAILED",
		fmt.Sprintf("Settlement failed: %s", reason),
		map[string]string{
			"instruction_id": instructionID,
			"reason":         reason,
		},
	)
}
