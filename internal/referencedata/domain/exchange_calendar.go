// 生成摘要：从 calendar 合并到 referencedata 域。交易日历属参考数据子聚合。
package domain

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrExchangeNotFound 交易所 MIC 未找到。
	ErrExchangeNotFound = errors.New("exchange MIC not found")
	// ErrYearNotGenerated 日历年份数据未生成。
	ErrYearNotGenerated = errors.New("calendar year data not generated")
)

// MarketSession 交易时段（如 Pre-Market, Regular, After-Market）。
type MarketSession struct {
	SessionName string        `json:"session_name"`
	StartTime   time.Duration `json:"start_time"`
	EndTime     time.Duration `json:"end_time"`
}

// TradingDay 日历的单个交易天配置。
type TradingDay struct {
	Date         time.Time       `json:"date"`
	IsTradingDay bool            `json:"is_trading_day"`
	IsHalfDay    bool            `json:"is_half_day"`
	HolidayName  string          `json:"holiday_name"`
	Sessions     []MarketSession `json:"sessions"`
}

// ExchangeCalendar 交易所整体交易日历聚合根。
// 维护全球各国交易所时区、法定节假日以及半日交易的时间窗口。
// 用于期权行权到期日计算以及 T+1/T+2 交收日对齐。
type ExchangeCalendar struct {
	MIC             string                 `json:"mic"`
	Name            string                 `json:"name"`
	CountryCode     string                 `json:"country"`
	TimeZoneName    string                 `json:"time_zone"`
	DefaultSessions []MarketSession        `json:"default_sessions"`
	Days            map[string]*TradingDay `json:"days"`
}

// IsMarketOpen 实时推算目前时点的交易是否开放。
func (c *ExchangeCalendar) IsMarketOpen(givenTime time.Time) (bool, *MarketSession, error) {
	loc, err := time.LoadLocation(c.TimeZoneName)
	if err != nil {
		return false, nil, err
	}
	localTime := givenTime.In(loc)
	dateKey := localTime.Format("2006-01-02")
	day, exists := c.Days[dateKey]
	if !exists {
		return false, nil, ErrYearNotGenerated
	}
	if !day.IsTradingDay {
		return false, nil, nil
	}
	timeOfDay := time.Duration(localTime.Hour())*time.Hour +
		time.Duration(localTime.Minute())*time.Minute +
		time.Duration(localTime.Second())*time.Second
	sessions := c.DefaultSessions
	if len(day.Sessions) > 0 {
		sessions = day.Sessions
	}
	for _, session := range sessions {
		if timeOfDay >= session.StartTime && timeOfDay <= session.EndTime {
			return true, &session, nil
		}
	}
	return false, nil, nil
}

// AddWorkingDays 推算证券 T+N 交收日（跨周末节假日）。
func (c *ExchangeCalendar) AddWorkingDays(start time.Time, n int) (time.Time, error) {
	loc, _ := time.LoadLocation(c.TimeZoneName)
	t := start.In(loc)
	daysAdded := 0
	for daysAdded < n {
		t = t.AddDate(0, 0, 1)
		dateKey := t.Format("2006-01-02")
		day, exists := c.Days[dateKey]
		if !exists {
			return start, ErrYearNotGenerated
		}
		if day.IsTradingDay {
			daysAdded++
		}
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc), nil
}

// CalendarRepository 交易日历仓储。
type CalendarRepository interface {
	LoadCalendar(ctx context.Context, mic string, year int) (*ExchangeCalendar, error)
	GetSettlementDate(ctx context.Context, mic string, tradeDate time.Time, tPlus int) (time.Time, error)
}
