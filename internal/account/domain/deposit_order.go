// 生成摘要：充值和提现订单聚合根，包含状态机流程。
// 迁移自 ecommerce/payment，现在属于 financialTrading/account 服务。
// 假设：充值通过外部网关回调确认；提现需要人工审核。
package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/fsm"
	"github.com/wyfcoding/pkg/idgen"
)

// DepositStatus 充值订单状态
type DepositStatus string

const (
	DepositStatusPending   DepositStatus = "PENDING"   // 等待支付
	DepositStatusConfirmed DepositStatus = "CONFIRMED" // 网关已确认
	DepositStatusCompleted DepositStatus = "COMPLETED" // 账户已入账
	DepositStatusFailed    DepositStatus = "FAILED"    // 失败
	DepositStatusCancelled DepositStatus = "CANCELLED" // 已取消
)

// WithdrawalStatus 提现订单状态
type WithdrawalStatus string

const (
	WithdrawalStatusPending    WithdrawalStatus = "PENDING"    // 用户提交
	WithdrawalStatusAuditing   WithdrawalStatus = "AUDITING"   // 审核中
	WithdrawalStatusApproved   WithdrawalStatus = "APPROVED"   // 审核通过
	WithdrawalStatusProcessing WithdrawalStatus = "PROCESSING" // 处理中 (网关执行)
	WithdrawalStatusCompleted  WithdrawalStatus = "COMPLETED"  // 已完成
	WithdrawalStatusRejected   WithdrawalStatus = "REJECTED"   // 已拒绝
	WithdrawalStatusFailed     WithdrawalStatus = "FAILED"     // 失败
)

// GatewayType 网关类型
type GatewayType string

const (
	GatewayTypeAlipay GatewayType = "alipay"
	GatewayTypeWechat GatewayType = "wechat"
	GatewayTypeBank   GatewayType = "bank"
	GatewayTypeCrypto GatewayType = "crypto"
)

// DepositOrder 充值订单聚合根
type DepositOrder struct {
	ID            uint            `json:"id"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	DepositNo     string          `json:"deposit_no"`
	UserID        string          `json:"user_id"`
	AccountID     string          `json:"account_id"`
	Amount        decimal.Decimal `json:"amount"`
	Currency      string          `json:"currency"`
	GatewayType   GatewayType     `json:"gateway_type"`
	Status        DepositStatus   `json:"status"`
	TransactionID string          `json:"transaction_id"`
	ThirdPartyNo  string          `json:"third_party_no"`
	PaymentURL    string          `json:"payment_url"`
	FailureReason string          `json:"failure_reason"`
	ConfirmedAt   *time.Time      `json:"confirmed_at"`
	CompletedAt   *time.Time      `json:"completed_at"`
	fsm           *fsm.Machine[string, string]
}

func (DepositOrder) TableName() string {
	return "deposit_orders"
}

// NewDepositOrder 创建充值订单
func NewDepositOrder(userID, accountID string, amount decimal.Decimal, currency string, gatewayType GatewayType, idGenerator idgen.Generator) *DepositOrder {
	depositNo := fmt.Sprintf("DEP%d", idGenerator.Generate())
	d := &DepositOrder{
		DepositNo:   depositNo,
		UserID:      userID,
		AccountID:   accountID,
		Amount:      amount,
		Currency:    currency,
		GatewayType: gatewayType,
		Status:      DepositStatusPending,
	}
	d.initFSM()
	return d
}

func (d *DepositOrder) initFSM() {
	m := fsm.NewMachine[string, string](string(d.Status))
	m.AddTransition(string(DepositStatusPending), "CONFIRM", string(DepositStatusConfirmed))
	m.AddTransition(string(DepositStatusConfirmed), "COMPLETE", string(DepositStatusCompleted))
	m.AddTransition(string(DepositStatusPending), "FAIL", string(DepositStatusFailed))
	m.AddTransition(string(DepositStatusConfirmed), "FAIL", string(DepositStatusFailed))
	m.AddTransition(string(DepositStatusPending), "CANCEL", string(DepositStatusCancelled))
	d.fsm = m
}

// InitFSM 确保状态机已初始化
func (d *DepositOrder) InitFSM() {
	if d.fsm == nil {
		d.initFSM()
	}
}

// Confirm 网关确认充值
func (d *DepositOrder) Confirm(ctx context.Context, transactionID, thirdPartyNo string) error {
	d.InitFSM()
	if err := d.fsm.Trigger(ctx, "CONFIRM"); err != nil {
		return err
	}
	d.Status = DepositStatusConfirmed
	d.TransactionID = transactionID
	d.ThirdPartyNo = thirdPartyNo
	now := time.Now()
	d.ConfirmedAt = &now
	return nil
}

// Complete 账户入账完成
func (d *DepositOrder) Complete(ctx context.Context) error {
	d.InitFSM()
	if err := d.fsm.Trigger(ctx, "COMPLETE"); err != nil {
		return err
	}
	d.Status = DepositStatusCompleted
	now := time.Now()
	d.CompletedAt = &now
	return nil
}

// Fail 充值失败
func (d *DepositOrder) Fail(ctx context.Context, reason string) error {
	d.InitFSM()
	if err := d.fsm.Trigger(ctx, "FAIL"); err != nil {
		return err
	}
	d.Status = DepositStatusFailed
	d.FailureReason = reason
	return nil
}

// Cancel 取消充值
func (d *DepositOrder) Cancel(ctx context.Context) error {
	d.InitFSM()
	if err := d.fsm.Trigger(ctx, "CANCEL"); err != nil {
		return err
	}
	d.Status = DepositStatusCancelled
	return nil
}

// WithdrawalOrder 提现订单聚合根
type WithdrawalOrder struct {
	ID               uint             `json:"id"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	WithdrawalNo     string           `json:"withdrawal_no"`
	UserID           string           `json:"user_id"`
	AccountID        string           `json:"account_id"`
	Amount           decimal.Decimal  `json:"amount"`
	Fee              decimal.Decimal  `json:"fee"`
	NetAmount        decimal.Decimal  `json:"net_amount"`
	Currency         string           `json:"currency"`
	BankAccountNo    string           `json:"bank_account_no"`
	BankName         string           `json:"bank_name"`
	BankHolder       string           `json:"bank_holder"`
	Status           WithdrawalStatus `json:"status"`
	AuditRemark      string           `json:"audit_remark"`
	AuditedBy        string           `json:"audited_by"`
	AuditedAt        *time.Time       `json:"audited_at"`
	GatewayReference string           `json:"gateway_reference"`
	FailureReason    string           `json:"failure_reason"`
	CompletedAt      *time.Time       `json:"completed_at"`
	fsm              *fsm.Machine[string, string]
}

func (WithdrawalOrder) TableName() string {
	return "withdrawal_orders"
}

// NewWithdrawalOrder 创建提现订单
func NewWithdrawalOrder(userID, accountID string, amount, fee decimal.Decimal, currency, bankAccountNo, bankName, bankHolder string, idGenerator idgen.Generator) *WithdrawalOrder {
	withdrawalNo := fmt.Sprintf("WD%d", idGenerator.Generate())
	w := &WithdrawalOrder{
		WithdrawalNo:  withdrawalNo,
		UserID:        userID,
		AccountID:     accountID,
		Amount:        amount,
		Fee:           fee,
		NetAmount:     amount.Sub(fee),
		Currency:      currency,
		BankAccountNo: bankAccountNo,
		BankName:      bankName,
		BankHolder:    bankHolder,
		Status:        WithdrawalStatusPending,
	}
	w.initFSM()
	return w
}

func (w *WithdrawalOrder) initFSM() {
	m := fsm.NewMachine[string, string](string(w.Status))
	m.AddTransition(string(WithdrawalStatusPending), "AUDIT", string(WithdrawalStatusAuditing))
	m.AddTransition(string(WithdrawalStatusAuditing), "APPROVE", string(WithdrawalStatusApproved))
	m.AddTransition(string(WithdrawalStatusAuditing), "REJECT", string(WithdrawalStatusRejected))
	m.AddTransition(string(WithdrawalStatusApproved), "PROCESS", string(WithdrawalStatusProcessing))
	m.AddTransition(string(WithdrawalStatusProcessing), "COMPLETE", string(WithdrawalStatusCompleted))
	m.AddTransition(string(WithdrawalStatusProcessing), "FAIL", string(WithdrawalStatusFailed))
	w.fsm = m
}

// InitFSM 确保状态机已初始化
func (w *WithdrawalOrder) InitFSM() {
	if w.fsm == nil {
		w.initFSM()
	}
}

// StartAudit 开始审核
func (w *WithdrawalOrder) StartAudit(ctx context.Context) error {
	w.InitFSM()
	if err := w.fsm.Trigger(ctx, "AUDIT"); err != nil {
		return err
	}
	w.Status = WithdrawalStatusAuditing
	return nil
}

// Approve 审核通过
func (w *WithdrawalOrder) Approve(ctx context.Context, auditor, remark string) error {
	w.InitFSM()
	if err := w.fsm.Trigger(ctx, "APPROVE"); err != nil {
		return err
	}
	w.Status = WithdrawalStatusApproved
	w.AuditedBy = auditor
	w.AuditRemark = remark
	now := time.Now()
	w.AuditedAt = &now
	return nil
}

// Reject 审核拒绝
func (w *WithdrawalOrder) Reject(ctx context.Context, auditor, remark string) error {
	w.InitFSM()
	if err := w.fsm.Trigger(ctx, "REJECT"); err != nil {
		return err
	}
	w.Status = WithdrawalStatusRejected
	w.AuditedBy = auditor
	w.AuditRemark = remark
	now := time.Now()
	w.AuditedAt = &now
	return nil
}

// StartProcessing 开始处理 (调用网关)
func (w *WithdrawalOrder) StartProcessing(ctx context.Context) error {
	w.InitFSM()
	if err := w.fsm.Trigger(ctx, "PROCESS"); err != nil {
		return err
	}
	w.Status = WithdrawalStatusProcessing
	return nil
}

// Complete 提现完成
func (w *WithdrawalOrder) Complete(ctx context.Context, gatewayRef string) error {
	w.InitFSM()
	if err := w.fsm.Trigger(ctx, "COMPLETE"); err != nil {
		return err
	}
	w.Status = WithdrawalStatusCompleted
	w.GatewayReference = gatewayRef
	now := time.Now()
	w.CompletedAt = &now
	return nil
}

// Fail 提现失败
func (w *WithdrawalOrder) Fail(ctx context.Context, reason string) error {
	w.InitFSM()
	if err := w.fsm.Trigger(ctx, "FAIL"); err != nil {
		return err
	}
	w.Status = WithdrawalStatusFailed
	w.FailureReason = reason
	return nil
}
