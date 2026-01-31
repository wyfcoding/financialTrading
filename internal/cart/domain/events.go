package domain

import "time"

// CartCreatedEvent 购物车创建事件
type CartCreatedEvent struct {
	CartID    uint      `json:"cart_id"`
	UserID    string    `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
}

// CartItemAddedEvent 购物车添加商品事件
type CartItemAddedEvent struct {
	CartID    uint      `json:"cart_id"`
	UserID    string    `json:"user_id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}

// CartItemRemovedEvent 购物车移除商品事件
type CartItemRemovedEvent struct {
	CartID    uint      `json:"cart_id"`
	UserID    string    `json:"user_id"`
	ProductID string    `json:"product_id"`
	Timestamp time.Time `json:"timestamp"`
}

// CartClearedEvent 购物车清空事件
type CartClearedEvent struct {
	CartID    uint      `json:"cart_id"`
	UserID    string    `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
}
