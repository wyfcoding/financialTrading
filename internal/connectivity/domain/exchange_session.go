// 生成摘要：从 marketaccess 合并到 connectivity 域。DMA 市场接入属连接层子聚合。
package domain

import (
	"time"

	"gorm.io/gorm"
)

// ExchangeSessionState 交易所接入通道状态。
type ExchangeSessionState string

const (
	// SessionConnected 已连接。
	SessionConnected ExchangeSessionState = "CONNECTED"
	// SessionDisconnected 已断开。
	SessionDisconnected ExchangeSessionState = "DISCONNECTED"
	// SessionHalted 交易所熔断。
	SessionHalted ExchangeSessionState = "HALTED"
)

// ExchangeSession 交易所接入通道聚合。
// 承担与底层物理交易所的网络会话保活、报文封包以及通道流控。
type ExchangeSession struct {
	gorm.Model
	// ExchangeID 交易所标识，全局唯一。
	ExchangeID string `gorm:"type:varchar(32);uniqueIndex" json:"exchange_id"`
	// Vendor 接入方式：FIX, REST, WEBSOCKET。
	Vendor string `gorm:"type:varchar(32);comment:如 FIX, REST, WEBSOCKET" json:"vendor"`
	// Endpoint 接入端点地址。
	Endpoint string `gorm:"type:varchar(255)" json:"endpoint"`
	// RateLimitPerSec 物理通道限流。
	RateLimitPerSec int32 `gorm:"default:100;comment:物理通道限流" json:"rate_limit_per_sec"`
	// State 通道状态。
	State ExchangeSessionState `gorm:"type:varchar(16)" json:"state"`
	// LastHeartbeat 最近心跳时间。
	LastHeartbeat time.Time `json:"last_heartbeat"`
}
