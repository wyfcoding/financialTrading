package application

type CreateOrderRequest struct {
	UserID        string
	Symbol        string
	Side          string
	OrderType     string
	Price         string
	Quantity      string
	TimeInForce   string
	StopPrice     string
	ClientOrderID string

	// Bracket support
	TakeProfitPrice string
	StopLossPrice   string

	// OCO support
	IsOCO         bool
	LinkedOrderID string
}

type OrderDTO struct {
	OrderID        string `json:"order_id"`
	UserID         string `json:"user_id"`
	Symbol         string `json:"symbol"`
	Side           string `json:"side"`
	OrderType      string `json:"order_type"`
	Price          string `json:"price"`
	Quantity       string `json:"quantity"`
	FilledQuantity string `json:"filled_quantity"`
	Status         string `json:"status"`
	TimeInForce    string `json:"time_in_force"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
	Remark         string `json:"remark"`
}
