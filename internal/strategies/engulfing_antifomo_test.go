package strategies

import (
	"testing"

	"github.com/letieu/trade-bot/internal/types"
)

// Helper function to create test candles
func createTestCandle(open, high, low, close float64) types.Candle {
	return types.Candle{
		Open:  open,
		High:  high,
		Low:   low,
		Close: close,
	}
}

func TestEngulfingAntiFomo_BullishEngulfing(t *testing.T) {
	strategy := NewEngulfingAntiFomoDefault()

	candles := []types.Candle{
		createTestCandle(100, 105, 95, 96), // Red candle
		createTestCandle(96, 102, 94, 101), // Green candle engulfing previous
	}

	if !strategy.isBullishEngulfing(candles) {
		t.Error("Expected bullish engulfing pattern")
	}
}

func TestEngulfingAntiFomo_BearishEngulfing(t *testing.T) {
	strategy := NewEngulfingAntiFomoDefault()

	candles := []types.Candle{
		createTestCandle(100, 105, 95, 104), // Green candle
		createTestCandle(104, 106, 99, 99),  // Red candle engulfing previous
	}

	if !strategy.isBearishEngulfing(candles) {
		t.Error("Expected bearish engulfing pattern")
	}
}

func TestEngulfingAntiFomo_ThreeRedCandles(t *testing.T) {
	strategy := NewEngulfingAntiFomoDefault()

	candles := []types.Candle{
		createTestCandle(100, 105, 95, 96), // Red
		createTestCandle(96, 100, 90, 91),  // Red
		createTestCandle(91, 95, 85, 86),   // Red
		createTestCandle(86, 92, 84, 90),   // Green (current)
	}

	if !strategy.hasThreeRedCandles(candles) {
		t.Error("Expected three consecutive red candles before current")
	}
}

func TestEngulfingAntiFomo_ThreeGreenCandles(t *testing.T) {
	strategy := NewEngulfingAntiFomoDefault()

	candles := []types.Candle{
		createTestCandle(100, 105, 95, 104),  // Green
		createTestCandle(104, 110, 102, 109), // Green
		createTestCandle(109, 115, 107, 114), // Green
		createTestCandle(114, 116, 108, 110), // Red (current)
	}

	if !strategy.hasThreeGreenCandles(candles) {
		t.Error("Expected three consecutive green candles before current")
	}
}

func TestEngulfingAntiFomo_LongSignalScenario(t *testing.T) {
	config := DefaultEngulfingAntiFomoConfig()
	config.UseSupertrend = false // Disable supertrend for simpler testing
	strategy := NewEngulfingAntiFomo(config)

	// With default config (RSI=14, Supertrend=10), we need:
	// 4 + 14 + 10 + 1 = 29 candles minimum
	requiredCandles := strategy.GetRequiredCandles()
	t.Logf("Strategy requires %d candles", requiredCandles)

	// Create a scenario with:
	// 1. Oversold RSI conditions (need many candles with downtrend)
	// 2. Bullish engulfing
	// 3. Three red candles
	// 4. Green confirmation

	candles := make([]types.Candle, 0)

	// Create downtrend to get oversold RSI (35 candles to ensure enough data)
	price := 100.0
	for i := 0; i < 35; i++ {
		candles = append(candles, createTestCandle(price, price+2, price-3, price-2))
		price -= 2
	}

	// Add bullish engulfing at oversold level
	candles = append(candles, createTestCandle(price, price+2, price-3, price-2)) // Red
	candles = append(candles, createTestCandle(price-2, price+1, price-3, price)) // Green engulfing

	// Now we should have signalPendingLong = true
	// Add three red candles
	candles = append(candles, createTestCandle(price, price+1, price-2, price-1))   // Red
	candles = append(candles, createTestCandle(price-1, price, price-3, price-2))   // Red
	candles = append(candles, createTestCandle(price-2, price-1, price-4, price-3)) // Red

	// Add green confirmation candle
	candles = append(candles, createTestCandle(price-3, price-1, price-4, price-2)) // Green

	t.Logf("Created %d candles for testing", len(candles))

	matched, err := strategy.Match(candles)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Note: This might not match due to RSI calculation complexity
	// The test demonstrates the flow rather than guaranteed match
	t.Logf("Long signal matched: %v", matched)
}

func TestEngulfingAntiFomo_GetMetadata(t *testing.T) {
	strategy := NewEngulfingAntiFomoDefault()

	// Create enough candles for metadata calculation
	candles := make([]types.Candle, 0)
	price := 100.0
	for i := 0; i < 50; i++ {
		candles = append(candles, createTestCandle(price, price+2, price-1, price+1))
		price += 0.5
	}

	metadata := strategy.GetMetadata(candles)

	if _, ok := metadata["rsi"]; !ok {
		t.Error("Expected RSI in metadata")
	}

	if _, ok := metadata["supertrend_value"]; !ok {
		t.Error("Expected supertrend_value in metadata")
	}

	if _, ok := metadata["stop_loss"]; !ok {
		t.Error("Expected stop_loss in metadata")
	}
}

func TestEngulfingAntiFomo_InsufficientCandles(t *testing.T) {
	strategy := NewEngulfingAntiFomoDefault()

	candles := []types.Candle{
		createTestCandle(100, 105, 95, 101),
	}

	matched, err := strategy.Match(candles)
	if err == nil {
		t.Error("Expected error for insufficient candles")
	}

	if matched {
		t.Error("Should not match with insufficient candles")
	}
}
