package application

import (
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/order/domain"
)

// PlaceOrderCommand 下单命令
// Side/Type/TimeInForce 使用字符串便于接口层输入
// StopPrice/Parent/IsOCO 兼容复杂订单
// Remain fields are basic for limit/market orders
//
// NOTE: fields are validated in command service.
type PlaceOrderCommand struct {
	UserID        string
	Symbol        string
	Side          string
	Type          string
	Price         float64
	StopPrice     float64
	Quantity      float64
	TimeInForce   string
	ParentOrderID string
	IsOCO         bool
}

// CancelOrderCommand 取消订单命令

type CancelOrderCommand struct {
	OrderID string
	UserID  string
	Reason  string
}

// OrderDTO API/Query 输出结构

type OrderDTO struct {
	OrderID        string `json:"order_id"`
	UserID         string `json:"user_id"`
	Symbol         string `json:"symbol"`
	Side           string `json:"side"`
	OrderType      string `json:"order_type"`
	Price          string `json:"price"`
	Quantity       string `json:"quantity"`
	FilledQuantity string `json:"filled_quantity"`
	AveragePrice   string `json:"average_price"`
	Status         string `json:"status"`
	TimeInForce    string `json:"time_in_force"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
	Remark         string `json:"remark"`
}

func toOrderDTO(o *domain.Order) *OrderDTO {
	if o == nil {
		return nil
	}
	return &OrderDTO{
		OrderID:        o.OrderID,
		UserID:         o.UserID,
		Symbol:         o.Symbol,
		Side:           string(o.Side),
		OrderType:      string(o.Type),
		Price:          decimal.NewFromFloat(o.Price).String(),
		Quantity:       decimal.NewFromFloat(o.Quantity).String(),
		FilledQuantity: decimal.NewFromFloat(o.FilledQuantity).String(),
		AveragePrice:   decimal.NewFromFloat(o.AveragePrice).String(),
		Status:         string(o.Status),
		TimeInForce:    string(o.TimeInForce),
		CreatedAt:      o.CreatedAt.Unix(),
		UpdatedAt:      o.UpdatedAt.Unix(),
		Remark:         "",
	}
}

func toOrderDTOs(orders []*domain.Order) []*OrderDTO {
	dtos := make([]*OrderDTO, 0, len(orders))
	for _, o := range orders {
		dtos = append(dtos, toOrderDTO(o))
	}
	return dtos
}

// CreateOrderRequest 兼容 HTTP 接口使用

type CreateOrderRequest struct {
	UserID          string
	Symbol          string
	Side            string
	OrderType       string
	Price           string
	Quantity        string
	TimeInForce     string
	StopPrice       string
	ClientOrderID   string
	TakeProfitPrice string
	StopLossPrice   string
	IsOCO           bool
	LinkedOrderID   string
}
