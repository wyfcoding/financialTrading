package mysql

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
	"gorm.io/gorm"
)

// TradeModel MySQL 成交表映射
type TradeModel struct {
	gorm.Model
	TradeID          string          `gorm:"column:trade_id;type:varchar(32);uniqueIndex;not null;comment:成交ID"`
	OrderID          string          `gorm:"column:order_id;type:varchar(32);index;not null;comment:订单ID"`
	UserID           string          `gorm:"column:user_id;type:varchar(32);index;not null;comment:用户ID"`
	Symbol           string          `gorm:"column:symbol;type:varchar(20);not null;comment:标的"`
	Side             string          `gorm:"column:side;type:varchar(10);not null;comment:方向"`
	ExecutedPrice    decimal.Decimal `gorm:"column:price;type:decimal(32,18);not null;comment:成交价"`
	ExecutedQuantity decimal.Decimal `gorm:"column:quantity;type:decimal(32,18);not null;comment:成交量"`
	ExecutedAt       time.Time       `gorm:"column:executed_at;not null;comment:成交时间"`
	Status           string          `gorm:"column:status;type:varchar(20);default:'EXECUTED';comment:状态"`
}

func (TradeModel) TableName() string {
	return "trades"
}

// AlgoOrderModel MySQL 算法订单表映射
type AlgoOrderModel struct {
	gorm.Model
	AlgoID            string          `gorm:"column:algo_id;type:varchar(64);uniqueIndex;not null;comment:算法订单ID"`
	UserID            string          `gorm:"column:user_id;type:varchar(64);not null;comment:用户ID"`
	Symbol            string          `gorm:"column:symbol;type:varchar(20);not null;comment:标的"`
	Side              string          `gorm:"column:side;type:varchar(10);not null;comment:方向"`
	TotalQuantity     decimal.Decimal `gorm:"column:total_quantity;type:decimal(20,8);not null;comment:总量"`
	ExecutedQuantity  decimal.Decimal `gorm:"column:executed_qty;type:decimal(20,8);default:0;comment:已成交量"`
	ParticipationRate decimal.Decimal `gorm:"column:participation_rate;type:decimal(10,4);default:0;comment:参与率"`
	AlgoType          string          `gorm:"column:algo_type;type:varchar(20);not null;comment:算法类型"`
	StartTime         time.Time       `gorm:"column:start_time;not null;comment:开始时间"`
	EndTime           time.Time       `gorm:"column:end_time;not null;comment:结束时间"`
	Status            string          `gorm:"column:status;type:varchar(20);default:'PENDING';comment:状态"`
	StrategyParams    string          `gorm:"column:strategy_params;type:text;comment:策略参数"`
}

func (AlgoOrderModel) TableName() string {
	return "algo_orders"
}

// EventPO 事件存储表
type EventPO struct {
	gorm.Model
	AggregateID string `gorm:"column:aggregate_id;type:varchar(64);index;not null"`
	EventType   string `gorm:"column:event_type;type:varchar(50);not null"`
	Payload     string `gorm:"column:payload;type:json;not null"`
	OccurredAt  int64  `gorm:"column:occurred_at;not null"`
}

func (EventPO) TableName() string {
	return "execution_events"
}

func toTradeModel(t *domain.Trade) *TradeModel {
	if t == nil {
		return nil
	}
	return &TradeModel{
		Model: gorm.Model{
			ID:        t.ID,
			CreatedAt: t.CreatedAt,
			UpdatedAt: t.UpdatedAt,
		},
		TradeID:          t.TradeID,
		OrderID:          t.OrderID,
		UserID:           t.UserID,
		Symbol:           t.Symbol,
		Side:             string(t.Side),
		ExecutedPrice:    t.ExecutedPrice,
		ExecutedQuantity: t.ExecutedQuantity,
		ExecutedAt:       t.ExecutedAt,
		Status:           t.Status,
	}
}

func toTrade(model *TradeModel) *domain.Trade {
	if model == nil {
		return nil
	}
	t := &domain.Trade{
		ID:               model.ID,
		CreatedAt:        model.CreatedAt,
		UpdatedAt:        model.UpdatedAt,
		TradeID:          model.TradeID,
		OrderID:          model.OrderID,
		UserID:           model.UserID,
		Symbol:           model.Symbol,
		Side:             domain.TradeSide(model.Side),
		ExecutedPrice:    model.ExecutedPrice,
		ExecutedQuantity: model.ExecutedQuantity,
		ExecutedAt:       model.ExecutedAt,
		Status:           model.Status,
	}
	t.SetID(t.TradeID)
	return t
}

func toAlgoOrderModel(o *domain.AlgoOrder) *AlgoOrderModel {
	if o == nil {
		return nil
	}
	return &AlgoOrderModel{
		Model: gorm.Model{
			ID:        o.ID,
			CreatedAt: o.CreatedAt,
			UpdatedAt: o.UpdatedAt,
		},
		AlgoID:            o.AlgoID,
		UserID:            o.UserID,
		Symbol:            o.Symbol,
		Side:              string(o.Side),
		TotalQuantity:     o.TotalQuantity,
		ExecutedQuantity:  o.ExecutedQuantity,
		ParticipationRate: o.ParticipationRate,
		AlgoType:          string(o.AlgoType),
		StartTime:         o.StartTime,
		EndTime:           o.EndTime,
		Status:            o.Status,
		StrategyParams:    o.StrategyParams,
	}
}

func toAlgoOrder(model *AlgoOrderModel) *domain.AlgoOrder {
	if model == nil {
		return nil
	}
	a := &domain.AlgoOrder{
		ID:                model.ID,
		CreatedAt:         model.CreatedAt,
		UpdatedAt:         model.UpdatedAt,
		AlgoID:            model.AlgoID,
		UserID:            model.UserID,
		Symbol:            model.Symbol,
		Side:              domain.TradeSide(model.Side),
		TotalQuantity:     model.TotalQuantity,
		ExecutedQuantity:  model.ExecutedQuantity,
		ParticipationRate: model.ParticipationRate,
		AlgoType:          domain.AlgoType(model.AlgoType),
		StartTime:         model.StartTime,
		EndTime:           model.EndTime,
		Status:            model.Status,
		StrategyParams:    model.StrategyParams,
	}
	a.SetID(a.AlgoID)
	return a
}
