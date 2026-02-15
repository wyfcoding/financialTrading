// Package domain 资金调拨领域模型
// 生成摘要：
// 1) 定义 TransferInstruction 实体，代表资金调拨指令
// 2) 支持审批流状态机管理
package domain

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type InstructionStatus string

const (
	InstructionStatusDraft     InstructionStatus = "DRAFT"
	InstructionStatusPending   InstructionStatus = "PENDING_APPROVAL"
	InstructionStatusApproved  InstructionStatus = "APPROVED"
	InstructionStatusRejected  InstructionStatus = "REJECTED"
	InstructionStatusExecuting InstructionStatus = "EXECUTING"
	InstructionStatusCompleted InstructionStatus = "COMPLETED"
	InstructionStatusFailed    InstructionStatus = "FAILED"
)

// TransferInstruction 资金调拨指令
type TransferInstruction struct {
	gorm.Model
	InstructionID   string            `gorm:"column:instruction_id;type:varchar(64);uniqueIndex;not null"`
	FromAccountID   uint64            `gorm:"column:from_account_id;not null"`
	ToAccountID     uint64            `gorm:"column:to_account_id;not null"`
	Amount          decimal.Decimal   `gorm:"column:amount;type:decimal(20,4);not null"`
	Currency        string            `gorm:"column:currency;type:char(3);not null"`
	RequestDate     time.Time         `gorm:"column:request_date;not null"`
	ExecutionDate   *time.Time        `gorm:"column:execution_date"`
	Status          InstructionStatus `gorm:"column:status;type:varchar(32);not null;default:'DRAFT'"`
	Purpose         string            `gorm:"column:purpose;type:varchar(255)"`
	ApproverID      string            `gorm:"column:approver_id;type:varchar(64)"`
	RejectionReason string            `gorm:"column:rejection_reason;type:varchar(255)"`
	TransactionRef  string            `gorm:"column:transaction_ref;type:varchar(64)"` // 关联的底层交易流水号
}

func (TransferInstruction) TableName() string { return "treasury_transfer_instructions" }

// Approve 审批通过
func (t *TransferInstruction) Approve(approverID string) error {
	if t.Status != InstructionStatusPending {
		return errors.New("instruction is not pending approval")
	}
	t.Status = InstructionStatusApproved
	t.ApproverID = approverID
	return nil
}

// Reject 审批拒绝
func (t *TransferInstruction) Reject(approverID, reason string) error {
	if t.Status != InstructionStatusPending {
		return errors.New("instruction is not pending approval")
	}
	t.Status = InstructionStatusRejected
	t.ApproverID = approverID
	t.RejectionReason = reason
	return nil
}

// Execute 开始执行
func (t *TransferInstruction) Execute() error {
	if t.Status != InstructionStatusApproved {
		return errors.New("instruction is not approved")
	}
	t.Status = InstructionStatusExecuting
	now := time.Now()
	t.ExecutionDate = &now
	return nil
}

// Complete 完成执行
func (t *TransferInstruction) Complete(txRef string) {
	t.Status = InstructionStatusCompleted
	t.TransactionRef = txRef
}

// Fail 执行失败
func (t *TransferInstruction) Fail(reason string) {
	t.Status = InstructionStatusFailed
	t.RejectionReason = reason // 复用字段存储失败原因
}
