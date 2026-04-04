package strategies

import (
	"fmt"

	"github.com/letieu/trade-bot/internal/indicators"
	"github.com/letieu/trade-bot/internal/types"
)

// EngulfingAntiFomoConfig holds the configuration for the strategy
type EngulfingAntiFomoConfig struct {
	RSILength        int     // Default: 14
	RSIOverBought    float64 // Default: 70
	RSIOverSold      float64 // Default: 30
	SupertrendMult   float64 // Default: 3.0
	SupertrendPeriod int     // Default: 10
	UseSupertrend    bool    // Default: true
	SLBufferPercent  float64 // Default: 25.0
}

// DefaultEngulfingAntiFomoConfig returns default configuration
func DefaultEngulfingAntiFomoConfig() EngulfingAntiFomoConfig {
	return EngulfingAntiFomoConfig{
		RSILength:        14,
		RSIOverBought:    70,
		RSIOverSold:      30,
		SupertrendMult:   3.0,
		SupertrendPeriod: 10,
		UseSupertrend:    true,
		SLBufferPercent:  25.0,
	}
}

// EngulfingAntiFomo implements a trading strategy that combines:
// - Engulfing patterns
// - RSI overbought/oversold conditions
// - Anti-FOMO confirmation (waits for 3 consecutive candles + reversal)
// - Optional Supertrend filter
type EngulfingAntiFomo struct {
	config             EngulfingAntiFomoConfig
	signalPendingLong  bool
	signalPendingShort bool
}

// NewEngulfingAntiFomo creates a new instance with custom config
func NewEngulfingAntiFomo(config EngulfingAntiFomoConfig) *EngulfingAntiFomo {
	return &EngulfingAntiFomo{
		config:             config,
		signalPendingLong:  false,
		signalPendingShort: false,
	}
}

// NewEngulfingAntiFomoDefault creates a new instance with default config
func NewEngulfingAntiFomoDefault() *EngulfingAntiFomo {
	return NewEngulfingAntiFomo(DefaultEngulfingAntiFomoConfig())
}

// isBullishEngulfing checks if current candle engulfs previous bearish candle
func (s *EngulfingAntiFomo) isBullishEngulfing(candles []types.Candle) bool {
	if len(candles) < 2 {
		return false
	}
	current := candles[len(candles)-1]
	previous := candles[len(candles)-2]

	return current.Close >= previous.Open && previous.Close < previous.Open
}

// isBearishEngulfing checks if current candle engulfs previous bullish candle
func (s *EngulfingAntiFomo) isBearishEngulfing(candles []types.Candle) bool {
	if len(candles) < 2 {
		return false
	}
	current := candles[len(candles)-1]
	previous := candles[len(candles)-2]

	return current.Close <= previous.Open && previous.Close > previous.Open
}

// isRSIOversold checks if RSI is oversold (current or previous 2 candles)
func (s *EngulfingAntiFomo) isRSIOversold(candles []types.Candle) bool {
	if len(candles) < 3 {
		return false
	}

	rsiCurrent := indicators.CalculateRSI(candles, s.config.RSILength)
	rsi1 := indicators.CalculateRSI(candles[:len(candles)-1], s.config.RSILength)
	rsi2 := indicators.CalculateRSI(candles[:len(candles)-2], s.config.RSILength)

	return rsiCurrent <= s.config.RSIOverSold ||
		rsi1 <= s.config.RSIOverSold ||
		rsi2 <= s.config.RSIOverSold
}

// isRSIOverbought checks if RSI is overbought (current or previous 2 candles)
func (s *EngulfingAntiFomo) isRSIOverbought(candles []types.Candle) bool {
	if len(candles) < 3 {
		return false
	}

	rsiCurrent := indicators.CalculateRSI(candles, s.config.RSILength)
	rsi1 := indicators.CalculateRSI(candles[:len(candles)-1], s.config.RSILength)
	rsi2 := indicators.CalculateRSI(candles[:len(candles)-2], s.config.RSILength)

	return rsiCurrent >= s.config.RSIOverBought ||
		rsi1 >= s.config.RSIOverBought ||
		rsi2 >= s.config.RSIOverBought
}

// hasThreeRedCandles checks if the 3 candles before current are all red
func (s *EngulfingAntiFomo) hasThreeRedCandles(candles []types.Candle) bool {
	if len(candles) < 4 {
		return false
	}

	idx := len(candles) - 1
	return candles[idx-3].Close < candles[idx-3].Open &&
		candles[idx-2].Close < candles[idx-2].Open &&
		candles[idx-1].Close < candles[idx-1].Open
}

// hasThreeGreenCandles checks if the 3 candles before current are all green
func (s *EngulfingAntiFomo) hasThreeGreenCandles(candles []types.Candle) bool {
	if len(candles) < 4 {
		return false
	}

	idx := len(candles) - 1
	return candles[idx-3].Close > candles[idx-3].Open &&
		candles[idx-2].Close > candles[idx-2].Open &&
		candles[idx-1].Close > candles[idx-1].Open
}

// isConfirmationGreen checks if current candle is green
func (s *EngulfingAntiFomo) isConfirmationGreen(candles []types.Candle) bool {
	if len(candles) == 0 {
		return false
	}
	current := candles[len(candles)-1]
	return current.Close > current.Open
}

// isConfirmationRed checks if current candle is red
func (s *EngulfingAntiFomo) isConfirmationRed(candles []types.Candle) bool {
	if len(candles) == 0 {
		return false
	}
	current := candles[len(candles)-1]
	return current.Close < current.Open
}

// Match implements the PatternMatcher interface
func (s *EngulfingAntiFomo) Match(candles []types.Candle) (bool, error) {
	requiredCandles := s.GetRequiredCandles()
	if len(candles) < requiredCandles {
		return false, fmt.Errorf("need at least %d candles, got %d", requiredCandles, len(candles))
	}

	// Check for pending signals and set flags
	if s.isRSIOversold(candles) && s.isBullishEngulfing(candles) {
		s.signalPendingLong = true
		s.signalPendingShort = false
	}

	if s.isRSIOverbought(candles) && s.isBearishEngulfing(candles) {
		s.signalPendingShort = true
		s.signalPendingLong = false
	}

	// Check Supertrend if enabled
	supertrendOk := true
	if s.config.UseSupertrend {
		st := indicators.CalculateSupertrend(candles, s.config.SupertrendPeriod, s.config.SupertrendMult)

		// For long: need bullish supertrend
		// For short: need bearish supertrend
		longCondition := s.signalPendingLong && s.hasThreeRedCandles(candles) && s.isConfirmationGreen(candles)
		shortCondition := s.signalPendingShort && s.hasThreeGreenCandles(candles) && s.isConfirmationRed(candles)

		if longCondition && !st.IsBullish() {
			supertrendOk = false
		}
		if shortCondition && !st.IsBearish() {
			supertrendOk = false
		}
	}

	// Check long condition
	longCondition := s.signalPendingLong &&
		s.hasThreeRedCandles(candles) &&
		s.isConfirmationGreen(candles) &&
		supertrendOk

	// Check short condition
	shortCondition := s.signalPendingShort &&
		s.hasThreeGreenCandles(candles) &&
		s.isConfirmationRed(candles) &&
		supertrendOk

	// Reset pending signals after match
	if longCondition {
		s.signalPendingLong = false
		return true, nil
	}

	if shortCondition {
		s.signalPendingShort = false
		return true, nil
	}

	return false, nil
}

// GetName returns the strategy name
func (s *EngulfingAntiFomo) GetName() string {
	return "ENGULFING ANTI-FOMO + SUPERTREND"
}

// GetDescription returns the strategy description
func (s *EngulfingAntiFomo) GetDescription() string {
	return "Combines engulfing patterns, RSI oversold/overbought, anti-FOMO confirmation, and optional Supertrend filter"
}

// GetRequiredCandles returns the minimum number of candles needed
func (s *EngulfingAntiFomo) GetRequiredCandles() int {
	// Need at least: 3 candles for confirmation + 1 current + RSI period + Supertrend period + 1 for calculation
	// The +1 is needed because indicators need at least N+1 candles to calculate N periods
	minRequired := 4 + s.config.RSILength + s.config.SupertrendPeriod + 1
	return minRequired
}

// GetMetadata returns additional information about the match
func (s *EngulfingAntiFomo) GetMetadata(candles []types.Candle) map[string]interface{} {
	if len(candles) < s.GetRequiredCandles() {
		return map[string]interface{}{}
	}

	rsi := indicators.CalculateRSI(candles, s.config.RSILength)
	st := indicators.CalculateSupertrend(candles, s.config.SupertrendPeriod, s.config.SupertrendMult)

	// Calculate stop loss level
	slLevel := s.calculateStopLoss(candles)

	return map[string]interface{}{
		"rsi":                rsi,
		"rsi_oversold":       s.config.RSIOverSold,
		"rsi_overbought":     s.config.RSIOverBought,
		"supertrend_value":   st.Value,
		"supertrend_bullish": st.IsBullish(),
		"stop_loss":          slLevel,
		"sl_buffer_percent":  s.config.SLBufferPercent,
	}
}

// calculateStopLoss calculates the stop loss level based on cluster range
func (s *EngulfingAntiFomo) calculateStopLoss(candles []types.Candle) float64 {
	if len(candles) < 2 {
		return 0
	}

	current := candles[len(candles)-1]
	previous := candles[len(candles)-2]

	clusterLow := min(current.Low, previous.Low)
	clusterHigh := max(current.High, previous.High)
	clusterRange := clusterHigh - clusterLow

	slMultiplier := s.config.SLBufferPercent / 100.0

	// For long positions: SL below cluster low
	// For short positions: SL above cluster high
	// We'll return both, consumer can choose based on position type
	if s.isConfirmationGreen(candles) {
		return clusterLow - (clusterRange * slMultiplier)
	}
	return clusterHigh + (clusterRange * slMultiplier)
}
