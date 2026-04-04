package indicators

import (
	"math"

	"github.com/letieu/trade-bot/internal/types"
)

// SupertrendResult holds the supertrend line value and direction
type SupertrendResult struct {
	Value     float64 // Supertrend line value
	Direction int     // -1 for bullish (buy), 1 for bearish (sell)
}

// CalculateATR calculates the Average True Range
func CalculateATR(candles []types.Candle, period int) float64 {
	if len(candles) < period+1 {
		return 0
	}

	trueRanges := make([]float64, 0, len(candles)-1)

	for i := 1; i < len(candles); i++ {
		high := candles[i].High
		low := candles[i].Low
		prevClose := candles[i-1].Close

		tr1 := high - low
		tr2 := math.Abs(high - prevClose)
		tr3 := math.Abs(low - prevClose)

		tr := math.Max(tr1, math.Max(tr2, tr3))
		trueRanges = append(trueRanges, tr)
	}

	if len(trueRanges) < period {
		return 0
	}

	// Calculate initial ATR (simple average)
	atr := 0.0
	for i := 0; i < period; i++ {
		atr += trueRanges[i]
	}
	atr /= float64(period)

	// Calculate smoothed ATR using Wilder's smoothing
	for i := period; i < len(trueRanges); i++ {
		atr = ((atr * float64(period-1)) + trueRanges[i]) / float64(period)
	}

	return atr
}

// CalculateSupertrend calculates the Supertrend indicator
func CalculateSupertrend(candles []types.Candle, period int, multiplier float64) SupertrendResult {
	if len(candles) < period+1 {
		return SupertrendResult{Value: 0, Direction: 0}
	}

	// Calculate basic bands
	var basicUpperBand, basicLowerBand float64
	var finalUpperBand, finalLowerBand float64
	var supertrend float64
	direction := 1 // 1 for bearish, -1 for bullish

	// Process all candles to get final supertrend value
	for i := period; i < len(candles); i++ {
		currentCandles := candles[:i+1]
		currentATR := CalculateATR(currentCandles, period)

		hl2 := (candles[i].High + candles[i].Low) / 2.0

		basicUpperBand = hl2 + (multiplier * currentATR)
		basicLowerBand = hl2 - (multiplier * currentATR)

		// Calculate final bands
		if i == period {
			finalUpperBand = basicUpperBand
			finalLowerBand = basicLowerBand
		} else {
			// Final Upper Band
			if basicUpperBand < finalUpperBand || candles[i-1].Close > finalUpperBand {
				finalUpperBand = basicUpperBand
			}
			// Final Lower Band
			if basicLowerBand > finalLowerBand || candles[i-1].Close < finalLowerBand {
				finalLowerBand = basicLowerBand
			}
		}

		// Determine supertrend and direction
		if i == period {
			if candles[i].Close <= finalUpperBand {
				supertrend = finalUpperBand
				direction = 1 // Bearish
			} else {
				supertrend = finalLowerBand
				direction = -1 // Bullish
			}
		} else {
			prevDirection := direction

			if prevDirection == 1 {
				if candles[i].Close > finalUpperBand {
					direction = -1
					supertrend = finalLowerBand
				} else {
					supertrend = finalUpperBand
				}
			} else {
				if candles[i].Close < finalLowerBand {
					direction = 1
					supertrend = finalUpperBand
				} else {
					supertrend = finalLowerBand
				}
			}
		}
	}

	return SupertrendResult{
		Value:     supertrend,
		Direction: direction,
	}
}

// IsBullish returns true if Supertrend is in bullish mode
func (s SupertrendResult) IsBullish() bool {
	return s.Direction < 0
}

// IsBearish returns true if Supertrend is in bearish mode
func (s SupertrendResult) IsBearish() bool {
	return s.Direction > 0
}
