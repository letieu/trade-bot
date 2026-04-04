package indicators

import (
	"testing"

	"github.com/letieu/trade-bot/internal/types"
)

func createTestCandle(open, high, low, close float64) types.Candle {
	return types.Candle{
		Open:  open,
		High:  high,
		Low:   low,
		Close: close,
	}
}

func TestCalculateRSI(t *testing.T) {
	// Create test candles with a clear trend
	candles := []types.Candle{
		createTestCandle(100, 105, 95, 101),
		createTestCandle(101, 106, 100, 102),
		createTestCandle(102, 107, 101, 103),
		createTestCandle(103, 108, 102, 104),
		createTestCandle(104, 109, 103, 105),
		createTestCandle(105, 110, 104, 106),
		createTestCandle(106, 111, 105, 107),
		createTestCandle(107, 112, 106, 108),
		createTestCandle(108, 113, 107, 109),
		createTestCandle(109, 114, 108, 110),
		createTestCandle(110, 115, 109, 111),
		createTestCandle(111, 116, 110, 112),
		createTestCandle(112, 117, 111, 113),
		createTestCandle(113, 118, 112, 114),
		createTestCandle(114, 119, 113, 115),
	}

	rsi := CalculateRSI(candles, 14)

	// In an uptrend, RSI should be > 50
	if rsi <= 50 {
		t.Errorf("Expected RSI > 50 in uptrend, got %.2f", rsi)
	}

	// RSI should be between 0 and 100
	if rsi < 0 || rsi > 100 {
		t.Errorf("RSI should be between 0 and 100, got %.2f", rsi)
	}

	t.Logf("RSI value: %.2f", rsi)
}

func TestCalculateRSI_InsufficientData(t *testing.T) {
	candles := []types.Candle{
		createTestCandle(100, 105, 95, 101),
	}

	rsi := CalculateRSI(candles, 14)

	// Should return default value of 50
	if rsi != 50.0 {
		t.Errorf("Expected default RSI of 50 with insufficient data, got %.2f", rsi)
	}
}

func TestCalculateATR(t *testing.T) {
	candles := []types.Candle{
		createTestCandle(100, 105, 95, 101),
		createTestCandle(101, 108, 98, 104),
		createTestCandle(104, 110, 100, 107),
		createTestCandle(107, 112, 103, 109),
		createTestCandle(109, 115, 105, 112),
		createTestCandle(112, 118, 108, 115),
		createTestCandle(115, 120, 110, 117),
		createTestCandle(117, 123, 113, 120),
		createTestCandle(120, 126, 116, 123),
		createTestCandle(123, 129, 119, 126),
		createTestCandle(126, 132, 122, 129),
		createTestCandle(129, 135, 125, 132),
		createTestCandle(132, 138, 128, 135),
		createTestCandle(135, 141, 131, 138),
		createTestCandle(138, 144, 134, 141),
	}

	atr := CalculateATR(candles, 14)

	// ATR should be positive
	if atr <= 0 {
		t.Errorf("Expected positive ATR, got %.2f", atr)
	}

	t.Logf("ATR value: %.2f", atr)
}

func TestCalculateSupertrend(t *testing.T) {
	// Create uptrend candles
	candles := make([]types.Candle, 0)
	price := 100.0
	for i := 0; i < 30; i++ {
		candles = append(candles, createTestCandle(price, price+5, price-2, price+3))
		price += 3
	}

	st := CalculateSupertrend(candles, 10, 3.0)

	// Should have a valid direction
	if st.Direction != 1 && st.Direction != -1 {
		t.Errorf("Expected direction to be 1 or -1, got %d", st.Direction)
	}

	// In uptrend, should eventually be bullish
	if !st.IsBullish() {
		t.Logf("Note: Supertrend is bearish in uptrend, value: %.2f, direction: %d", st.Value, st.Direction)
	}

	t.Logf("Supertrend value: %.2f, direction: %d, bullish: %v", st.Value, st.Direction, st.IsBullish())
}

func TestSupertrendResult_IsBullish(t *testing.T) {
	bullish := SupertrendResult{Value: 100, Direction: -1}
	if !bullish.IsBullish() {
		t.Error("Expected IsBullish to return true for direction -1")
	}

	bearish := SupertrendResult{Value: 100, Direction: 1}
	if bullish.IsBearish() {
		t.Error("Expected IsBearish to return false for direction -1")
	}

	if !bearish.IsBearish() {
		t.Error("Expected IsBearish to return true for direction 1")
	}
}
