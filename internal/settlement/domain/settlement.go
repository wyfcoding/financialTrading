// 变更说明：完善结算服务领域模型，增加 DVP 券款对付、多币种结算、结算失败处理等完整功能
package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// SettlementStatus 结算状态
type SettlementStatus int8

const (
	SettlementStatusPending    SettlementStatus = 1 // 待结算
	SettlementStatusNetting    SettlementStatus = 2 // 净额清算中
	SettlementStatusCleared    SettlementStatus = 3 // 已清算
	SettlementStatusProcessing SettlementStatus = 4 // 处理中
	SettlementStatusSettled    SettlementStatus = 5 // 已交收
	SettlementStatusFailed     SettlementStatus = 6 // 失败
	SettlementStatusCancelled  SettlementStatus = 7 // 已取消
)

func (s SettlementStatus) String() string {
	switch s {
	case SettlementStatusPending:
		return "PENDING"
	case SettlementStatusNetting:
		return "NETTING"
	case SettlementStatusCleared:
		return "CLEARED"
	case SettlementStatusProcessing:
		return "PROCESSING"
	case SettlementStatusSettled:
		return "SETTLED"
	case SettlementStatusFailed:
		return "FAILED"
	case SettlementStatusCancelled:
		return "CANCELLED"
	default:
		return "UNKNOWN"
	}
}

// SettlementType 结算类型
type SettlementType int8

const (
	SettlementTypeDVP   SettlementType = 1 // 券款对付
	SettlementTypeFOP   SettlementType = 2 // 只付券
	SettlementTypeRVP   SettlementType = 3 // 只收款
	SettlementTypeFree  SettlementType = 4 // 免费交付
)

func (t SettlementType) String() string {
	switch t {
	case SettlementTypeDVP:
		return "DVP"
	case SettlementTypeFOP:
		return "FOP"
	case SettlementTypeRVP:
		return "RVP"
	case SettlementTypeFree:
		return "FREE"
	default:
		return "UNKNOWN"
	}
}

// SettlementInstruction 结算指令聚合根
type SettlementInstruction struct {
	gorm.Model
	InstructionID    string           `gorm:"column:instruction_id;type:varchar(64);uniqueIndex;not null" json:"instruction_id"`
	TradeID          string           `gorm:"column:trade_id;type:varchar(64);index;not null" json:"trade_id"`
	OrderID          string           `gorm:"column:order_id;type:varchar(64);index" json:"order_id"`
	Symbol           string           `gorm:"column:symbol;type:varchar(32);not null" json:"symbol"`
	SecurityType     string           `gorm:"column:security_type;type:varchar(32)" json:"security_type"`
	Quantity         decimal.Decimal  `gorm:"column:quantity;type:decimal(20,4);not null" json:"quantity"`
	Price            decimal.Decimal  `gorm:"column:price;type:decimal(18,8);not null" json:"price"`
	Amount           decimal.Decimal  `gorm:"column:amount;type:decimal(20,2);not null" json:"amount"`
	Currency         string           `gorm:"column:currency;type:varchar(3);not null" json:"currency"`
	SettlementType   SettlementType   `gorm:"column:settlement_type;type:tinyint;not null;default:1" json:"settlement_type"`
	
	BuyerAccountID   string           `gorm:"column:buyer_account_id;type:varchar(64);index;not null" json:"buyer_account_id"`
	BuyerCustodian   string           `gorm:"column:buyer_custodian;type:varchar(64)" json:"buyer_custodian"`
	BuyerSettleAcct  string           `gorm:"column:buyer_settle_account;type:varchar(64)" json:"buyer_settle_account"`
	
	SellerAccountID  string           `gorm:"column:seller_account_id;type:varchar(64);index;not null" json:"seller_account_id"`
	SellerCustodian  string           `gorm:"column:seller_custodian;type:varchar(64)" json:"seller_custodian"`
	SellerSettleAcct string           `gorm:"column:seller_settle_account;type:varchar(64)" json:"seller_settle_account"`
	
	TradeDate        time.Time        `gorm:"column:trade_date;not null" json:"trade_date"`
	SettlementDate   time.Time        `gorm:"column:settlement_date;index;not null" json:"settlement_date"`
	ValueDate        time.Time        `gorm:"column:value_date" json:"value_date"`
	Status           SettlementStatus `gorm:"column:status;type:tinyint;not null;default:1" json:"status"`
	FailReason       string           `gorm:"column:fail_reason;type:varchar(512)" json:"fail_reason"`
	RetryCount       int              `gorm:"column:retry_count;default:0" json:"retry_count"`
	MaxRetry         int              `gorm:"column:max_retry;default:3" json:"max_retry"`
	
	CCPFlag          bool             `gorm:"column:ccp_flag;default:false" json:"ccp_flag"`
	CCPAccount       string           `gorm:"column:ccp_account;type:varchar(64)" json:"ccp_account"`
	
	NettingID        string           `gorm:"column:netting_id;type:varchar(64);index" json:"netting_id"`
	BatchID          string           `gorm:"column:batch_id;type:varchar(64);index" json:"batch_id"`
	
	FeeAmount        decimal.Decimal  `gorm:"column:fee_amount;type:decimal(20,2)" json:"fee_amount"`
	FeeCurrency      string           `gorm:"column:fee_currency;type:varchar(3)" json:"fee_currency"`
	TaxAmount        decimal.Decimal  `gorm:"column:tax_amount;type:decimal(20,2)" json:"tax_amount"`
	
	ConfirmedAt      *time.Time       `gorm:"column:confirmed_at" json:"confirmed_at"`
	SettledAt        *time.Time       `gorm:"column:settled_at" json:"settled_at"`
	
	Events           []SettlementEvent `gorm:"foreignKey:InstructionID;references:InstructionID" json:"events"`
}

// TableName 表名
func (SettlementInstruction) TableName() string {
	return "settlement_instructions"
}

// SettlementEvent 结算事件
type SettlementEvent struct {
	gorm.Model
	InstructionID string    `gorm:"column:instruction_id;type:varchar(64);index;not null" json:"instruction_id"`
	EventType     string    `gorm:"column:event_type;type:varchar(32);not null" json:"event_type"`
	Description   string    `gorm:"column:description;type:varchar(255)" json:"description"`
	Status        string    `gorm:"column:status;type:varchar(32)" json:"status"`
	OccurredAt    time.Time `gorm:"column:occurred_at;not null" json:"occurred_at"`
}

// TableName 表名
func (SettlementEvent) TableName() string {
	return "settlement_events"
}

// NettingResult 净额清算结果
type NettingResult struct {
	gorm.Model
	NettingID      string          `gorm:"column:netting_id;type:varchar(64);uniqueIndex;not null" json:"netting_id"`
	AccountID      string          `gorm:"column:account_id;type:varchar(64);index;not null" json:"account_id"`
	Currency       string          `gorm:"column:currency;type:varchar(3);not null" json:"currency"`
	Symbol         string          `gorm:"column:symbol;type:varchar(32)" json:"symbol"`
	GrossAmount    decimal.Decimal `gorm:"column:gross_amount;type:decimal(20,2)" json:"gross_amount"`
	NetAmount      decimal.Decimal `gorm:"column:net_amount;type:decimal(20,2)" json:"net_amount"`
	NetQuantity    decimal.Decimal `gorm:"column:net_quantity;type:decimal(20,4)" json:"net_quantity"`
	InstructionIDs string          `gorm:"column:instruction_ids;type:text" json:"instruction_ids"`
	Status         string          `gorm:"column:status;type:varchar(32)" json:"status"`
	CreatedAt      time.Time       `gorm:"column:created_at" json:"created_at"`
}

// TableName 表名
func (NettingResult) TableName() string {
	return "netting_results"
}

// SettlementBatch 结算批次
type SettlementBatch struct {
	gorm.Model
	BatchID        string          `gorm:"column:batch_id;type:varchar(64);uniqueIndex;not null" json:"batch_id"`
	SettlementDate time.Time       `gorm:"column:settlement_date;not null" json:"settlement_date"`
	Currency       string          `gorm:"column:currency;type:varchar(3)" json:"currency"`
	TotalCount     int             `gorm:"column:total_count" json:"total_count"`
	TotalAmount    decimal.Decimal `gorm:"column:total_amount;type:decimal(20,2)" json:"total_amount"`
	SuccessCount   int             `gorm:"column:success_count" json:"success_count"`
	FailedCount    int             `gorm:"column:failed_count" json:"failed_count"`
	Status         string          `gorm:"column:status;type:varchar(32)" json:"status"`
	StartedAt      *time.Time      `gorm:"column:started_at" json:"started_at"`
	CompletedAt    *time.Time      `gorm:"column:completed_at" json:"completed_at"`
}

// TableName 表名
func (SettlementBatch) TableName() string {
	return "settlement_batches"
}

// FXRate 汇率
type FXRate struct {
	gorm.Model
	FromCurrency string          `gorm:"column:from_currency;type:varchar(3);not null" json:"from_currency"`
	ToCurrency   string          `gorm:"column:to_currency;type:varchar(3);not null" json:"to_currency"`
	Rate         decimal.Decimal `gorm:"column:rate;type:decimal(18,8);not null" json:"rate"`
	BidRate      decimal.Decimal `gorm:"column:bid_rate;type:decimal(18,8)" json:"bid_rate"`
	AskRate      decimal.Decimal `gorm:"column:ask_rate;type:decimal(18,8)" json:"ask_rate"`
	Source       string          `gorm:"column:source;type:varchar(32)" json:"source"`
	EffectiveAt  time.Time       `gorm:"column:effective_at" json:"effective_at"`
	ExpiresAt    *time.Time      `gorm:"column:expires_at" json:"expires_at"`
}

// TableName 表名
func (FXRate) TableName() string {
	return "fx_rates"
}

// NewSettlementInstruction 创建结算指令
func NewSettlementInstruction(
	tradeID, symbol string,
	quantity, price decimal.Decimal,
	currency string,
	buyerAccountID, sellerAccountID string,
	settlementDays int,
) *SettlementInstruction {
	now := time.Now()
	tradeDate := now.Truncate(24 * time.Hour)
	settlementDate := tradeDate.AddDate(0, 0, settlementDays)
	amount := quantity.Mul(price)

	return &SettlementInstruction{
		InstructionID:  fmt.Sprintf("SI%d%s", now.UnixNano(), tradeID[:8]),
		TradeID:        tradeID,
		Symbol:         symbol,
		Quantity:       quantity,
		Price:          price,
		Amount:         amount,
		Currency:       currency,
		SettlementType: SettlementTypeDVP,
		BuyerAccountID: buyerAccountID,
		SellerAccountID: sellerAccountID,
		TradeDate:      tradeDate,
		SettlementDate: settlementDate,
		Status:         SettlementStatusPending,
		MaxRetry:       3,
		Events:         []SettlementEvent{},
	}
}

// SetCustodian 设置托管商
func (s *SettlementInstruction) SetCustodian(buyerCustodian, buyerSettleAcct, sellerCustodian, sellerSettleAcct string) {
	s.BuyerCustodian = buyerCustodian
	s.BuyerSettleAcct = buyerSettleAcct
	s.SellerCustodian = sellerCustodian
	s.SellerSettleAcct = sellerSettleAcct
}

// SetCCP 设置中央对手方
func (s *SettlementInstruction) SetCCP(ccpAccount string) {
	s.CCPFlag = true
	s.CCPAccount = ccpAccount
}

// StartNetting 开始净额清算
func (s *SettlementInstruction) StartNetting(nettingID string) error {
	if s.Status != SettlementStatusPending {
		return errors.New("invalid status for netting")
	}
	s.Status = SettlementStatusNetting
	s.NettingID = nettingID
	s.addEvent("NETTING_STARTED", "开始净额清算", "PROCESSING")
	return nil
}

// CompleteNetting 完成净额清算
func (s *SettlementInstruction) CompleteNetting() error {
	if s.Status != SettlementStatusNetting {
		return errors.New("invalid status for complete netting")
	}
	s.Status = SettlementStatusCleared
	s.addEvent("NETTING_COMPLETED", "净额清算完成", "SUCCESS")
	return nil
}

// StartProcessing 开始处理
func (s *SettlementInstruction) StartProcessing(batchID string) error {
	if s.Status != SettlementStatusCleared && s.Status != SettlementStatusPending {
		return errors.New("invalid status for processing")
	}
	s.Status = SettlementStatusProcessing
	s.BatchID = batchID
	s.addEvent("PROCESSING_STARTED", "开始结算处理", "PROCESSING")
	return nil
}

// Confirm 确认结算
func (s *SettlementInstruction) Confirm() error {
	now := time.Now()
	s.ConfirmedAt = &now
	s.addEvent("CONFIRMED", "结算确认", "SUCCESS")
	return nil
}

// Settle 完成交收
func (s *SettlementInstruction) Settle() error {
	if s.Status != SettlementStatusProcessing {
		return errors.New("invalid status for settle")
	}
	now := time.Now()
	s.Status = SettlementStatusSettled
	s.SettledAt = &now
	s.addEvent("SETTLED", "结算完成", "SUCCESS")
	return nil
}

// Fail 结算失败
func (s *SettlementInstruction) Fail(reason string) error {
	s.Status = SettlementStatusFailed
	s.FailReason = reason
	s.addEvent("FAILED", reason, "FAILED")
	return nil
}

// Retry 重试
func (s *SettlementInstruction) Retry() error {
	if s.RetryCount >= s.MaxRetry {
		return errors.New("max retry count exceeded")
	}
	s.RetryCount++
	s.Status = SettlementStatusPending
	s.FailReason = ""
	s.addEvent("RETRY", fmt.Sprintf("重试结算 (第%d次)", s.RetryCount), "PENDING")
	return nil
}

// Cancel 取消结算
func (s *SettlementInstruction) Cancel(reason string) error {
	if s.Status == SettlementStatusSettled {
		return errors.New("cannot cancel settled instruction")
	}
	s.Status = SettlementStatusCancelled
	s.FailReason = reason
	s.addEvent("CANCELLED", reason, "CANCELLED")
	return nil
}

// CanRetry 是否可以重试
func (s *SettlementInstruction) CanRetry() bool {
	return s.Status == SettlementStatusFailed && s.RetryCount < s.MaxRetry
}

// IsSettled 是否已结算
func (s *SettlementInstruction) IsSettled() bool {
	return s.Status == SettlementStatusSettled
}

// IsFailed 是否失败
func (s *SettlementInstruction) IsFailed() bool {
	return s.Status == SettlementStatusFailed
}

// addEvent 添加事件
func (s *SettlementInstruction) addEvent(eventType, description, status string) {
	s.Events = append(s.Events, SettlementEvent{
		InstructionID: s.InstructionID,
		EventType:     eventType,
		Description:   description,
		Status:        status,
		OccurredAt:    time.Now(),
	})
}

// SettlementRepository 结算仓储接口
type SettlementRepository interface {
	Save(ctx context.Context, instruction *SettlementInstruction) error
	Update(ctx context.Context, instruction *SettlementInstruction) error
	Get(ctx context.Context, instructionID string) (*SettlementInstruction, error)
	GetByTradeID(ctx context.Context, tradeID string) (*SettlementInstruction, error)
	FindPendingByDate(ctx context.Context, date time.Time, limit int) ([]*SettlementInstruction, error)
	FindPendingByAccount(ctx context.Context, accountID string, limit int) ([]*SettlementInstruction, error)
	UpdateStatus(ctx context.Context, instructionID string, status SettlementStatus, reason string) error
	WithTx(ctx context.Context, fn func(txCtx context.Context) error) error
}

// NettingRepository 净额清算仓储接口
type NettingRepository interface {
	Save(ctx context.Context, result *NettingResult) error
	Get(ctx context.Context, nettingID string) (*NettingResult, error)
	GetByAccountAndCurrency(ctx context.Context, accountID, currency string) (*NettingResult, error)
}

// BatchRepository 批次仓储接口
type BatchRepository interface {
	Save(ctx context.Context, batch *SettlementBatch) error
	Get(ctx context.Context, batchID string) (*SettlementBatch, error)
	GetByDate(ctx context.Context, date time.Time) (*SettlementBatch, error)
}

// FXRateRepository 汇率仓储接口
type FXRateRepository interface {
	GetRate(ctx context.Context, fromCurrency, toCurrency string) (*FXRate, error)
	SaveRate(ctx context.Context, rate *FXRate) error
}

// SettlementReadRepository 结算读模型仓储接口
type SettlementReadRepository interface {
	Save(ctx context.Context, instruction *SettlementInstruction) error
	Get(ctx context.Context, instructionID string) (*SettlementInstruction, error)
	Delete(ctx context.Context, instructionID string) error
}

// CustodianService 托管服务接口
type CustodianService interface {
	TransferSecurity(ctx context.Context, fromAccount, toAccount, symbol string, quantity decimal.Decimal) error
	TransferCash(ctx context.Context, fromAccount, toAccount string, amount decimal.Decimal, currency string) error
	GetAccountBalance(ctx context.Context, accountID, currency string) (decimal.Decimal, error)
	GetSecurityPosition(ctx context.Context, accountID, symbol string) (decimal.Decimal, error)
	FreezeAccount(ctx context.Context, accountID string, amount decimal.Decimal, currency string) error
	UnfreezeAccount(ctx context.Context, accountID string, amount decimal.Decimal, currency string) error
}

// CCPService 中央对手方服务接口
type CCPService interface {
	RegisterTrade(ctx context.Context, instruction *SettlementInstruction) error
	CalculateMargin(ctx context.Context, accountID string) (decimal.Decimal, error)
}

// NotificationService 通知服务接口
type NotificationService interface {
	NotifySettlementCreated(ctx context.Context, accountID, instructionID string) error
	NotifySettlementCompleted(ctx context.Context, accountID, instructionID string) error
	NotifySettlementFailed(ctx context.Context, accountID, instructionID, reason string) error
}

// SettlementDomainService 结算领域服务
type SettlementDomainService struct {
	custodianSvc CustodianService
	ccpSvc       CCPService
	notification NotificationService
}

// NewSettlementDomainService 创建结算领域服务
func NewSettlementDomainService(
	custodianSvc CustodianService,
	ccpSvc CCPService,
	notification NotificationService,
) *SettlementDomainService {
	return &SettlementDomainService{
		custodianSvc: custodianSvc,
		ccpSvc:       ccpSvc,
		notification: notification,
	}
}

// ExecuteDVP 执行DVP结算
func (s *SettlementDomainService) ExecuteDVP(ctx context.Context, instruction *SettlementInstruction) error {
	if instruction.SettlementType != SettlementTypeDVP {
		return errors.New("not a DVP instruction")
	}

	if s.custodianSvc == nil {
		return nil
	}

	sellerAccount := instruction.SellerAccountID
	buyerAccount := instruction.BuyerAccountID
	if instruction.CCPFlag && instruction.CCPAccount != "" {
		sellerAccount = instruction.CCPAccount
		buyerAccount = instruction.CCPAccount
	}

	if err := s.custodianSvc.TransferSecurity(ctx, sellerAccount, buyerAccount, instruction.Symbol, instruction.Quantity); err != nil {
		return fmt.Errorf("security transfer failed: %w", err)
	}

	if err := s.custodianSvc.TransferCash(ctx, buyerAccount, sellerAccount, instruction.Amount, instruction.Currency); err != nil {
		_ = s.custodianSvc.TransferSecurity(ctx, buyerAccount, sellerAccount, instruction.Symbol, instruction.Quantity)
		return fmt.Errorf("cash transfer failed: %w", err)
	}

	return nil
}

// ValidateBalance 验证余额
func (s *SettlementDomainService) ValidateBalance(ctx context.Context, instruction *SettlementInstruction) error {
	if s.custodianSvc == nil {
		return nil
	}

	cashBalance, err := s.custodianSvc.GetAccountBalance(ctx, instruction.BuyerAccountID, instruction.Currency)
	if err != nil {
		return fmt.Errorf("failed to get buyer cash balance: %w", err)
	}
	if cashBalance.LessThan(instruction.Amount) {
		return errors.New("insufficient cash balance")
	}

	securityPosition, err := s.custodianSvc.GetSecurityPosition(ctx, instruction.SellerAccountID, instruction.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get seller security position: %w", err)
	}
	if securityPosition.LessThan(instruction.Quantity) {
		return errors.New("insufficient security position")
	}

	return nil
}

// NotifyCompletion 通知结算完成
func (s *SettlementDomainService) NotifyCompletion(ctx context.Context, instruction *SettlementInstruction) error {
	if s.notification == nil {
		return nil
	}
	_ = s.notification.NotifySettlementCompleted(ctx, instruction.BuyerAccountID, instruction.InstructionID)
	_ = s.notification.NotifySettlementCompleted(ctx, instruction.SellerAccountID, instruction.InstructionID)
	return nil
}
