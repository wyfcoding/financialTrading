package application

type SignalDTO struct {
	Symbol    string  `json:"symbol"`
	Indicator string  `json:"indicator"`
	Period    int     `json:"period"`
	Value     float64 `json:"value"`
	Timestamp int64   `json:"timestamp"`
}
