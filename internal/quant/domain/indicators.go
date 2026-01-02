package domain

import (
	"fmt"
	"math"

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

	// 计算完整的 EMA 序列以保证精度
	fastEMAs := CalculateEMASeries(prices, fastPeriod)
	slowEMAs := CalculateEMASeries(prices, slowPeriod)

	macdLineSeries := make([]decimal.Decimal, len(prices))
	for i := range prices {
		if i < slowPeriod-1 {
			continue
		}
		macdLineSeries[i] = fastEMAs[i].Sub(slowEMAs[i])
	}

	// 信号线 = EMA(MACD 线序列, 信号线周期)
	validMACDSeries := macdLineSeries[slowPeriod-1:]
	signalLineSeries := CalculateEMASeries(validMACDSeries, signalPeriod)

	// 取最新的计算结果
	lastIndex := len(prices) - 1
	macdLine := macdLineSeries[lastIndex]
	signalLine := signalLineSeries[len(signalLineSeries)-1]
	histogram := macdLine.Sub(signalLine)

	return macdLine, signalLine, histogram, nil
}

// CalculateEMA 计算指数移动平均 (Exponential Moving Average)
func CalculateEMA(prices []decimal.Decimal, period int) decimal.Decimal {
	series := CalculateEMASeries(prices, period)
	if len(series) == 0 {
		return decimal.Zero
	}
	return series[len(series)-1]
}

// CalculateEMASeries 计算完整的 EMA 序列
func CalculateEMASeries(prices []decimal.Decimal, period int) []decimal.Decimal {
	if len(prices) == 0 || period <= 0 {
		return nil
	}

	emaSeries := make([]decimal.Decimal, len(prices))
	k := decimal.NewFromFloat(2.0 / float64(period+1))

	// 初始 EMA: 使用第一个价格点
	emaSeries[0] = prices[0]

	for i := 1; i < len(prices); i++ {
		// EMA 计算公式: EMA(t) = Price(t) * k + EMA(t-1) * (1 - k)
		prevEMA := emaSeries[i-1]
		currentPrice := prices[i]
		emaSeries[i] = currentPrice.Mul(k).Add(prevEMA.Mul(decimal.NewFromInt(1).Sub(k)))
	}

	return emaSeries
}

// CalculateBollingerBands 计算布林带 (Bollinger Bands)
// 返回: upperBand, middleBand, lowerBand
func (s *IndicatorService) CalculateBollingerBands(prices []decimal.Decimal, period int, stdDevMult float64) (decimal.Decimal, decimal.Decimal, decimal.Decimal, error) {
	if len(prices) < period {
		return decimal.Zero, decimal.Zero, decimal.Zero, fmt.Errorf("not enough data points for Bollinger Bands")
	}

	// 1. 计算中轨 (SMA)
	sum := decimal.Zero
	recentPrices := prices[len(prices)-period:]
	for _, p := range recentPrices {
		sum = sum.Add(p)
	}
	middleBand := sum.Div(decimal.NewFromInt(int64(period)))

	// 2. 计算标准差
	varianceSum := decimal.Zero
	for _, p := range recentPrices {
		diff := p.Sub(middleBand)
		varianceSum = varianceSum.Add(diff.Mul(diff))
	}
	variance := varianceSum.Div(decimal.NewFromInt(int64(period)))
	stdDev := decimal.NewFromFloat(math.Sqrt(variance.InexactFloat64()))

	// 3. 计算上下轨
	mult := decimal.NewFromFloat(stdDevMult)
	upperBand := middleBand.Add(stdDev.Mul(mult))
	lowerBand := middleBand.Sub(stdDev.Mul(mult))

	return upperBand, middleBand, lowerBand, nil
}

// CalculateATR 计算平均真实波幅 (Average True Range)
func (s *IndicatorService) CalculateATR(highs, lows, closes []decimal.Decimal, period int) (decimal.Decimal, error) {
	if len(highs) < period+1 {
		return decimal.Zero, fmt.Errorf("not enough data points for ATR")
	}

	trSeries := make([]decimal.Decimal, len(highs)-1)
	for i := 1; i < len(highs); i++ {
		// TR = max(high-low, abs(high-prev_close), abs(low-prev_close))
		h_l := highs[i].Sub(lows[i])
		h_pc := highs[i].Sub(closes[i-1]).Abs()
		l_pc := lows[i].Sub(closes[i-1]).Abs()
		
		tr := decimal.Max(h_l, h_pc, l_pc)
		trSeries[i-1] = tr
	}

	// 计算 TR 序列的 EMA 或 SMA 作为 ATR
	// 按照 Wilder 标准，通常使用类似 EMA 的平滑方式
	return CalculateEMA(trSeries, period), nil
}
