package mysql

import (
	"time"

	"github.com/wyfcoding/financialtrading/internal/order/domain"
)

// OrderModel MySQL 订单表映射
type OrderModel struct {
	ID             uint      `gorm:"primaryKey;autoIncrement"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
	OrderID        string    `gorm:"column:order_id;type:varchar(36);uniqueIndex;not null"`
	UserID         string    `gorm:"column:user_id;type:varchar(50);index;not null"`
	Symbol         string    `gorm:"column:symbol;type:varchar(20);index;not null"`
	Side           string    `gorm:"column:side;type:varchar(10);not null"`
	Type           string    `gorm:"column:type;type:varchar(20);not null"`
	Price          float64   `gorm:"column:price;type:decimal(20,8)"`
	StopPrice      float64   `gorm:"column:stop_price;type:decimal(20,8)"`
	Quantity       float64   `gorm:"column:quantity;type:decimal(20,8);not null"`
	FilledQuantity float64   `gorm:"column:filled_quantity;type:decimal(20,8);default:0"`
	AveragePrice   float64   `gorm:"column:average_price;type:decimal(20,8);default:0"`
	Status         string    `gorm:"column:status;type:varchar(20);index;not null"`
	TimeInForce    string    `gorm:"column:tif;type:varchar(10);default:'GTC'"`
	ParentOrderID  string    `gorm:"column:parent_id;type:varchar(36);index"`
	IsOCO          bool      `gorm:"column:is_oco"`
}

func (OrderModel) TableName() string { return "orders" }

// EventModel 事件存储表
type EventModel struct {
	ID          uint      `gorm:"primaryKey;autoIncrement"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
	AggregateID string    `gorm:"column:aggregate_id;type:varchar(36);index;not null"`
	EventType   string    `gorm:"column:event_type;type:varchar(50);not null"`
	Payload     string    `gorm:"column:payload;type:json;not null"`
	OccurredAt  int64     `gorm:"column:occurred_at;not null"`
}

func (EventModel) TableName() string { return "order_events" }

// mapping helpers

func toOrderModel(o *domain.Order) *OrderModel {
	if o == nil {
		return nil
	}
	return &OrderModel{
		ID:             o.ID,
		CreatedAt:      o.CreatedAt,
		UpdatedAt:      o.UpdatedAt,
		OrderID:        o.OrderID,
		UserID:         o.UserID,
		Symbol:         o.Symbol,
		Side:           string(o.Side),
		Type:           string(o.Type),
		Price:          o.Price,
		StopPrice:      o.StopPrice,
		Quantity:       o.Quantity,
		FilledQuantity: o.FilledQuantity,
		AveragePrice:   o.AveragePrice,
		Status:         string(o.Status),
		TimeInForce:    string(o.TimeInForce),
		ParentOrderID:  o.ParentOrderID,
		IsOCO:          o.IsOCO,
	}
}

func toOrder(m *OrderModel) *domain.Order {
	if m == nil {
		return nil
	}
	return &domain.Order{
		ID:             m.ID,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
		OrderID:        m.OrderID,
		UserID:         m.UserID,
		Symbol:         m.Symbol,
		Side:           domain.OrderSide(m.Side),
		Type:           domain.OrderType(m.Type),
		Price:          m.Price,
		StopPrice:      m.StopPrice,
		Quantity:       m.Quantity,
		FilledQuantity: m.FilledQuantity,
		AveragePrice:   m.AveragePrice,
		Status:         domain.OrderStatus(m.Status),
		TimeInForce:    domain.TimeInForce(m.TimeInForce),
		ParentOrderID:  m.ParentOrderID,
		IsOCO:          m.IsOCO,
	}
}
