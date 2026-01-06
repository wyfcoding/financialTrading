// 包 domain 账户服务的领域模型
package domain

import (
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Account 账户实体，代表用户的核心资金池。
// 聚合根属性：负责管理余额的原子变更与并发版本控制。
type Account struct {
	gorm.Model
	AccountID        string          `gorm:"column:account_id;type:varchar(32);uniqueIndex;not null;comment:账户业务唯一ID" json:"account_id"`
	UserID           string          `gorm:"column:user_id;type:varchar(32);index;not null;comment:所属用户唯一ID" json:"user_id"`
	AccountType      string          `gorm:"column:account_type;type:varchar(20);not null;comment:账户类型(SPOT/MARGIN)" json:"account_type"`
	Currency         string          `gorm:"column:currency;type:varchar(10);not null;comment:结算币种(如USDT/BTC)" json:"currency"`
	Balance          decimal.Decimal `gorm:"column:balance;type:decimal(32,18);default:0;not null;comment:账户总余额" json:"balance"`
	AvailableBalance decimal.Decimal `gorm:"column:available_balance;type:decimal(32,18);default:0;not null;comment:当前可用余额" json:"available_balance"`
	FrozenBalance    decimal.Decimal `gorm:"column:frozen_balance;type:decimal(32,18);default:0;not null;comment:冻结中的保证金" json:"frozen_balance"`
	Version          int64           `gorm:"column:version;default:0;not null;comment:乐观锁控制版本号" json:"version"`
}

// Transaction 封装了每一笔资金变动的流水记录。
type Transaction struct {
	gorm.Model
	TransactionID string          `gorm:"column:transaction_id;type:varchar(32);uniqueIndex;not null;comment:资金流水唯一ID" json:"transaction_id"`
	AccountID     string          `gorm:"column:account_id;type:varchar(32);index;not null;comment:关联账户业务ID" json:"account_id"`
	Type          string          `gorm:"column:type;type:varchar(20);not null;comment:变动类型(DEPOSIT/WITHDRAW/TRADE)" json:"type"`
	Amount        decimal.Decimal `gorm:"column:amount;type:decimal(32,18);not null;comment:变动涉及金额" json:"amount"`
	Status        string          `gorm:"column:status;type:varchar(20);not null;comment:流水最终状态" json:"status"`
}

// End of domain file
