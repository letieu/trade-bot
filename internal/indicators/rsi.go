package indicators

import (
	"math"

	"github.com/letieu/trade-bot/internal/types"
)

// CalculateRSI calculates the Relative Strength Index for a given period
// Returns RSI value between 0 and 100
func CalculateRSI(candles []types.Candle, period int) float64 {
	if len(candles) < period+1 {
		return 50.0 // Default neutral value if not enough data
	}

	// Calculate price changes
	gains := make([]float64, 0, len(candles)-1)
	losses := make([]float64, 0, len(candles)-1)

	for i := 1; i < len(candles); i++ {
		change := candles[i].Close - candles[i-1].Close
		if change > 0 {
			gains = append(gains, change)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, math.Abs(change))
		}
	}

	if len(gains) < period {
		return 50.0
	}

	// Calculate initial average gain and loss
	avgGain := 0.0
	avgLoss := 0.0
	for i := 0; i < period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	// Calculate smoothed averages for remaining values
	for i := period; i < len(gains); i++ {
		avgGain = ((avgGain * float64(period-1)) + gains[i]) / float64(period)
		avgLoss = ((avgLoss * float64(period-1)) + losses[i]) / float64(period)
	}

	// Avoid division by zero
	if avgLoss == 0 {
		return 100.0
	}

	rs := avgGain / avgLoss
	rsi := 100.0 - (100.0 / (1.0 + rs))

	return rsi
}

// CalculateRSIHistory calculates RSI values for all candles
func CalculateRSIHistory(candles []types.Candle, period int) []float64 {
	rsiValues := make([]float64, len(candles))

	for i := range candles {
		if i < period {
			rsiValues[i] = 50.0 // Default neutral value
		} else {
			rsiValues[i] = CalculateRSI(candles[:i+1], period)
		}
	}

	return rsiValues
}
