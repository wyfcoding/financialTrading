// 包 domain 账户服务的领域模型
package domain

import (
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Account 账户实体
// 代表用户的资金账户，包含余额和状态信息
type Account struct {
	gorm.Model
	// 账户 ID (业务主键)，全局唯一
	AccountID string `gorm:"column:account_id;type:varchar(32);uniqueIndex;not null" json:"account_id"`
	// 用户 ID，关联的用户
	UserID string `gorm:"column:user_id;type:varchar(32);index;not null" json:"user_id"`
	// 账户类型（SPOT: 现货, MARGIN: 杠杆, FUTURES: 合约）
	AccountType string `gorm:"column:account_type;type:varchar(20);not null" json:"account_type"`
	// 货币（如 USD, BTC, ETH）
	Currency string `gorm:"column:currency;type:varchar(10);not null" json:"currency"`
	// 总余额 = 可用余额 + 冻结余额
	Balance decimal.Decimal `gorm:"column:balance;type:decimal(32,18);default:0;not null" json:"balance"`
	// 可用余额
	AvailableBalance decimal.Decimal `gorm:"column:available_balance;type:decimal(32,18);default:0;not null" json:"available_balance"`
	// 冻结余额
	FrozenBalance decimal.Decimal `gorm:"column:frozen_balance;type:decimal(32,18);default:0;not null" json:"frozen_balance"`
	// 版本号 (用于乐观锁并发控制)
	Version int64 `gorm:"column:version;default:0;not null" json:"version"`
}

// Transaction 交易记录
type Transaction struct {
	gorm.Model
	// 交易 ID (业务主键)
	TransactionID string `gorm:"column:transaction_id;type:varchar(32);uniqueIndex;not null" json:"transaction_id"`
	// 账户 ID
	AccountID string `gorm:"column:account_id;type:varchar(32);index;not null" json:"account_id"`
	// 交易类型（DEPOSIT, WITHDRAW, TRADE, FEE）
	Type string `gorm:"column:type;type:varchar(20);not null" json:"type"`
	// 金额
	Amount decimal.Decimal `gorm:"column:amount;type:decimal(32,18);not null" json:"amount"`
	// 状态
	Status string `gorm:"column:status;type:varchar(20);not null" json:"status"`
}

// End of domain file
