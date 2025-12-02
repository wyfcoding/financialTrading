package domain

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// IndicatorService 指标计算服务
// 提供常用技术指标的计算方法
type IndicatorService struct{}

// NewIndicatorService 创建指标计算服务实例
func NewIndicatorService() *IndicatorService {
	return &IndicatorService{}
}

// CalculateRSI 计算相对强弱指数 (Relative Strength Index)
// prices: 价格序列，按时间升序排列（最新的在最后）
// period: 计算周期，通常为 14
// 返回: RSI 值 (0-100)
func (s *IndicatorService) CalculateRSI(prices []decimal.Decimal, period int) (decimal.Decimal, error) {
	if len(prices) < period+1 {
		return decimal.Zero, fmt.Errorf("not enough data points for RSI calculation")
	}

	if period <= 0 {
		return decimal.Zero, fmt.Errorf("invalid period")
	}

	gains := decimal.Zero
	losses := decimal.Zero

	// 计算初始平均涨跌幅
	for i := 1; i <= period; i++ {
		change := prices[i].Sub(prices[i-1])
		if change.GreaterThan(decimal.Zero) {
			gains = gains.Add(change)
		} else {
			losses = losses.Add(change.Abs())
		}
	}

	avgGain := gains.Div(decimal.NewFromInt(int64(period)))
	avgLoss := losses.Div(decimal.NewFromInt(int64(period)))

	// 平滑计算后续数据
	for i := period + 1; i < len(prices); i++ {
		change := prices[i].Sub(prices[i-1])
		currentGain := decimal.Zero
		currentLoss := decimal.Zero

		if change.GreaterThan(decimal.Zero) {
			currentGain = change
		} else {
			currentLoss = change.Abs()
		}

		// Wilder 平滑法
		// 平均涨幅 = ((上一次平均涨幅 * (周期 - 1)) + 当前涨幅) / 周期
		avgGain = avgGain.Mul(decimal.NewFromInt(int64(period - 1))).Add(currentGain).Div(decimal.NewFromInt(int64(period)))
		avgLoss = avgLoss.Mul(decimal.NewFromInt(int64(period - 1))).Add(currentLoss).Div(decimal.NewFromInt(int64(period)))
	}

	if avgLoss.IsZero() {
		return decimal.NewFromInt(100), nil
	}

	rs := avgGain.Div(avgLoss)
	rsi := decimal.NewFromInt(100).Sub(decimal.NewFromInt(100).Div(decimal.NewFromInt(1).Add(rs)))

	return rsi, nil
}

// CalculateMACD 计算移动平均收敛散度 (Moving Average Convergence Divergence)
// prices: 价格序列
// fastPeriod: 快线周期 (通常 12)
// slowPeriod: 慢线周期 (通常 26)
// signalPeriod: 信号线周期 (通常 9)
// 返回: macdLine, signalLine, histogram
func (s *IndicatorService) CalculateMACD(prices []decimal.Decimal, fastPeriod, slowPeriod, signalPeriod int) (decimal.Decimal, decimal.Decimal, decimal.Decimal, error) {
	if len(prices) < slowPeriod+signalPeriod {
		return decimal.Zero, decimal.Zero, decimal.Zero, fmt.Errorf("not enough data points for MACD calculation")
	}

	// 注意：这里简化了 EMA 计算，实际 MACD 需要基于 EMA 序列计算信号线 (Signal Line)
	// 为了准确计算，我们需要计算出 EMA 序列，而不仅仅是最后一个点的 EMA

	// 重新实现：计算 EMA 序列
	fastEMAs := CalculateEMASeries(prices, fastPeriod)
	slowEMAs := CalculateEMASeries(prices, slowPeriod)

	// MACD 线 = 快线 EMA - 慢线 EMA
	// 我们需要 MACD 线的序列来计算信号线
	// 序列长度取决于较短的 slowEMAs
	// slowEMAs 的起始点比 fastEMAs 晚 (slowPeriod - fastPeriod)

	// 对齐序列
	// fastEMAs 长度: len(prices)
	// slowEMAs 长度: len(prices) (前 slowPeriod-1 个为 0 或无效)

	macdLineSeries := make([]decimal.Decimal, len(prices))
	for i := 0; i < len(prices); i++ {
		if i < slowPeriod-1 {
			continue
		}
		macdLineSeries[i] = fastEMAs[i].Sub(slowEMAs[i])
	}

	// 信号线 = EMA(MACD 线, 信号线周期)
	// 注意：MACD 线前 slowPeriod-1 个无效，计算信号线时要跳过
	validMACDSeries := macdLineSeries[slowPeriod-1:]
	signalLineSeries := CalculateEMASeries(validMACDSeries, signalPeriod)

	// 取最后一个值
	lastIndex := len(prices) - 1
	macdLine := macdLineSeries[lastIndex]
	// signalLineSeries 的长度是 len(prices) - (slowPeriod - 1)
	// 对应的最后一个值是 signalLineSeries[len(signalLineSeries)-1]
	signalLine := signalLineSeries[len(signalLineSeries)-1]

	histogram := macdLine.Sub(signalLine)

	return macdLine, signalLine, histogram, nil
}

// CalculateEMA 计算指数移动平均 (Exponential Moving Average) - 仅返回最后一个值
func CalculateEMA(prices []decimal.Decimal, period int) decimal.Decimal {
	series := CalculateEMASeries(prices, period)
	if len(series) == 0 {
		return decimal.Zero
	}
	return series[len(series)-1]
}

// CalculateEMASeries 计算 EMA 序列
func CalculateEMASeries(prices []decimal.Decimal, period int) []decimal.Decimal {
	if len(prices) == 0 {
		return nil
	}

	emaSeries := make([]decimal.Decimal, len(prices))
	k := decimal.NewFromFloat(2.0 / float64(period+1))

	// 初始 EMA 通常用 SMA 代替，或者直接用第一个价格
	// 这里简单处理：第一个价格作为初始 EMA
	emaSeries[0] = prices[0]

	for i := 1; i < len(prices); i++ {
		// EMA = Price(t) * k + EMA(y) * (1 - k)
		emaSeries[i] = prices[i].Mul(k).Add(emaSeries[i-1].Mul(decimal.NewFromInt(1).Sub(k)))
	}

	return emaSeries
}
