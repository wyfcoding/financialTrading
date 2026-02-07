package application

// SubmitOrderCommand 提交订单命令 DTO

type SubmitOrderCommand struct {
	OrderID                string `json:"order_id"`
	Symbol                 string `json:"symbol"`
	Side                   string `json:"side"` // "buy" or "sell"
	Price                  string `json:"price"`
	Quantity               string `json:"quantity"`
	UserID                 string `json:"user_id"`
	IsIceberg              bool   `json:"is_iceberg"`
	IcebergDisplayQuantity string `json:"iceberg_display_quantity"`
	PostOnly               bool   `json:"post_only"`
}

// OrderBookDTO 订单簿 DTO

type OrderBookDTO struct {
	Symbol    string     `json:"symbol"`
	Bids      []LevelDTO `json:"bids"`
	Asks      []LevelDTO `json:"asks"`
	Timestamp int64      `json:"timestamp"`
}

// LevelDTO 订单簿档位 DTO

type LevelDTO struct {
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
}

// TradeDTO 成交 DTO

type TradeDTO struct {
	TradeID      string `json:"trade_id"`
	MakerOrderID string `json:"maker_order_id"`
	TakerOrderID string `json:"taker_order_id"`
	Symbol       string `json:"symbol"`
	Price        string `json:"price"`
	Quantity     string `json:"quantity"`
	Timestamp    int64  `json:"timestamp"`
}
