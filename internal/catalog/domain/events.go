package domain

import "time"

// ProductCreatedEvent 商品创建事件
type ProductCreatedEvent struct {
	ProductID uint      `json:"product_id"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	Stock     int       `json:"stock"`
	Category  string    `json:"category"`
	Timestamp time.Time `json:"timestamp"`
}

// ProductUpdatedEvent 商品更新事件
type ProductUpdatedEvent struct {
	ProductID uint      `json:"product_id"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	Stock     int       `json:"stock"`
	Category  string    `json:"category"`
	Timestamp time.Time `json:"timestamp"`
}

// ProductStockChangedEvent 商品库存变更事件
type ProductStockChangedEvent struct {
	ProductID uint      `json:"product_id"`
	OldStock  int       `json:"old_stock"`
	NewStock  int       `json:"new_stock"`
	Timestamp time.Time `json:"timestamp"`
}
