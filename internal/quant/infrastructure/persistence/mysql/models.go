package mysql

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/quant/domain"
	"gorm.io/gorm"
)

// StrategyModel 策略数据库模型
type StrategyModel struct {
	gorm.Model
	ID          string `gorm:"column:id;type:varchar(32);primaryKey"`
	Name        string `gorm:"column:name;type:varchar(100);not null"`
	Description string `gorm:"column:description;type:text"`
	Script      string `gorm:"column:script;type:text"`
	Status      string `gorm:"column:status;type:varchar(20);default:'ACTIVE'"`
}

func (StrategyModel) TableName() string { return "strategies" }

// BacktestResultModel 回测结果数据库模型
type BacktestResultModel struct {
	gorm.Model
	ID            string `gorm:"column:id;type:varchar(32);primaryKey"`
	StrategyID    string `gorm:"column:strategy_id;type:varchar(32);index;not null"`
	Symbol        string `gorm:"column:symbol;type:varchar(32);not null"`
	StartTime     int64  `gorm:"column:start_time;type:bigint"`
	EndTime       int64  `gorm:"column:end_time;type:bigint"`
	TotalReturn   string `gorm:"column:total_return;type:decimal(32,18)"`
	MaxDrawdown   string `gorm:"column:max_drawdown;type:decimal(32,18)"`
	SharpeRatio   string `gorm:"column:sharpe_ratio;type:decimal(32,18)"`
	TotalTrades   int    `gorm:"column:total_trades;type:int"`
	WinningTrades int    `gorm:"column:winning_trades;type:int"`
	Status        string `gorm:"column:status;type:varchar(20);default:'RUNNING'"`
}

func (BacktestResultModel) TableName() string { return "backtest_results" }

// SignalModel 信号数据库模型
type SignalModel struct {
	gorm.Model
	StrategyID string    `gorm:"column:strategy_id;type:varchar(32);index"`
	Symbol     string    `gorm:"column:symbol;type:varchar(20);index;not null"`
	Indicator  string    `gorm:"column:indicator;type:varchar(10);not null"`
	Period     int       `gorm:"column:period;not null"`
	Value      float64   `gorm:"column:value;type:decimal(20,8)"`
	Confidence float64   `gorm:"column:confidence;type:decimal(10,6)"`
	Timestamp  time.Time `gorm:"column:timestamp;index"`
}

func (SignalModel) TableName() string { return "signals" }

// mapping helpers

func toStrategyModel(s *domain.Strategy) *StrategyModel {
	if s == nil {
		return nil
	}
	return &StrategyModel{
		Model: gorm.Model{
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
		},
		ID:          s.ID,
		Name:        s.Name,
		Description: s.Description,
		Script:      s.Script,
		Status:      string(s.Status),
	}
}

func toStrategy(m *StrategyModel) *domain.Strategy {
	if m == nil {
		return nil
	}
	return &domain.Strategy{
		ID:          m.ID,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		Name:        m.Name,
		Description: m.Description,
		Script:      m.Script,
		Status:      domain.StrategyStatus(m.Status),
	}
}

func toBacktestResultModel(r *domain.BacktestResult) *BacktestResultModel {
	if r == nil {
		return nil
	}
	return &BacktestResultModel{
		Model: gorm.Model{
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
		},
		ID:            r.ID,
		StrategyID:    r.StrategyID,
		Symbol:        r.Symbol,
		StartTime:     r.StartTime,
		EndTime:       r.EndTime,
		TotalReturn:   r.TotalReturn.String(),
		MaxDrawdown:   r.MaxDrawdown.String(),
		SharpeRatio:   r.SharpeRatio.String(),
		TotalTrades:   r.TotalTrades,
		WinningTrades: r.WinningTrades,
		Status:        string(r.Status),
	}
}

func toBacktestResult(m *BacktestResultModel) (*domain.BacktestResult, error) {
	if m == nil {
		return nil, nil
	}
	totalReturn, err := decimal.NewFromString(m.TotalReturn)
	if err != nil {
		return nil, err
	}
	maxDrawdown, err := decimal.NewFromString(m.MaxDrawdown)
	if err != nil {
		return nil, err
	}
	sharpeRatio, err := decimal.NewFromString(m.SharpeRatio)
	if err != nil {
		return nil, err
	}

	return &domain.BacktestResult{
		ID:            m.ID,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		StrategyID:    m.StrategyID,
		Symbol:        m.Symbol,
		StartTime:     m.StartTime,
		EndTime:       m.EndTime,
		TotalReturn:   totalReturn,
		MaxDrawdown:   maxDrawdown,
		SharpeRatio:   sharpeRatio,
		TotalTrades:   m.TotalTrades,
		WinningTrades: m.WinningTrades,
		Status:        domain.BacktestStatus(m.Status),
	}, nil
}

func toSignalModel(signal *domain.Signal) *SignalModel {
	if signal == nil {
		return nil
	}
	return &SignalModel{
		Model: gorm.Model{
			ID:        signal.ID,
			CreatedAt: signal.CreatedAt,
			UpdatedAt: signal.UpdatedAt,
		},
		StrategyID: signal.StrategyID,
		Symbol:     signal.Symbol,
		Indicator:  string(signal.Indicator),
		Period:     signal.Period,
		Value:      signal.Value,
		Confidence: signal.Confidence,
		Timestamp:  signal.Timestamp,
	}
}

func toSignal(m *SignalModel) *domain.Signal {
	if m == nil {
		return nil
	}
	return &domain.Signal{
		ID:         m.ID,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
		StrategyID: m.StrategyID,
		Symbol:     m.Symbol,
		Indicator:  domain.IndicatorType(m.Indicator),
		Period:     m.Period,
		Value:      m.Value,
		Confidence: m.Confidence,
		Timestamp:  m.Timestamp,
	}
}
