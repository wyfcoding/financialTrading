package application

// PositionDTO 持仓 DTO
type PositionDTO struct {
	PositionID    string `json:"position_id"`
	UserID        string `json:"user_id"`
	Symbol        string `json:"symbol"`
	Side          string `json:"side"`
	Quantity      string `json:"quantity"`
	EntryPrice    string `json:"entry_price"`
	CurrentPrice  string `json:"current_price,omitempty"`
	UnrealizedPnL string `json:"unrealized_pnl,omitempty"`
	RealizedPnL   string `json:"realized_pnl"`
	OpenedAt      int64  `json:"opened_at"`
	ClosedAt      *int64 `json:"closed_at,omitempty"`
	Status        string `json:"status"`
}
