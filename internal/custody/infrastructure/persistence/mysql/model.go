package mysql

import (
	"time"

	"gorm.io/gorm"
)

type AssetVaultModel struct {
	gorm.Model
	VaultID string `gorm:"column:vault_id;type:varchar(64);uniqueIndex;not null"`
	Type    string `gorm:"column:type;type:varchar(16);not null"`
	UserID  uint64 `gorm:"column:user_id;index"`
	Symbol  string `gorm:"column:symbol;type:varchar(32);index;not null"`
	Balance int64  `gorm:"column:balance;not null"`
	Locked  int64  `gorm:"column:locked;not null;default:0"`
}

func (AssetVaultModel) TableName() string { return "asset_vaults" }

type CustodyTransferModel struct {
	gorm.Model
	TransferID string    `gorm:"column:transfer_id;type:varchar(64);uniqueIndex;not null"`
	FromVault  string    `gorm:"column:from_vault;type:varchar(64);not null"`
	ToVault    string    `gorm:"column:to_vault;type:varchar(64);not null"`
	Symbol     string    `gorm:"column:symbol;type:varchar(32);not null"`
	Amount     int64     `gorm:"column:amount;not null"`
	Reason     string    `gorm:"column:reason;type:varchar(255)"`
	Timestamp  time.Time `gorm:"column:timestamp;autoCreateTime"`
}

func (CustodyTransferModel) TableName() string { return "custody_transfers" }

type CorpActionModel struct {
	gorm.Model
	ActionID   string    `gorm:"column:action_id;type:varchar(64);uniqueIndex;not null"`
	Symbol     string    `gorm:"column:symbol;type:varchar(32);index;not null"`
	Type       string    `gorm:"column:type;type:varchar(16);not null"`
	Ratio      float64   `gorm:"column:ratio;type:decimal(18,8);not null"`
	RecordDate time.Time `gorm:"column:record_date;not null"`
	ExDate     time.Time `gorm:"column:ex_date;not null"`
	PayDate    time.Time `gorm:"column:pay_date;not null"`
	Status     string    `gorm:"column:status;type:varchar(16);not null"`
}

func (CorpActionModel) TableName() string { return "corp_actions" }

type CorpActionExecutionModel struct {
	gorm.Model
	ExecutionID string    `gorm:"column:execution_id;type:varchar(64);uniqueIndex;not null"`
	ActionID    string    `gorm:"column:action_id;type:varchar(64);index;not null"`
	UserID      uint64    `gorm:"column:user_id;index;not null"`
	OldPosition int64     `gorm:"column:old_position;not null"`
	NewPosition int64     `gorm:"column:new_position;not null"`
	ChangeAmt   int64     `gorm:"column:change_amt;not null"`
	ExecutedAt  time.Time `gorm:"column:executed_at;autoCreateTime"`
}

func (CorpActionExecutionModel) TableName() string { return "corp_action_executions" }
