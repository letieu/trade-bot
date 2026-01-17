package tests

import (
	"testing"
	"time"

	"github.com/letieu/trade-bot/internal/bot"
	"github.com/letieu/trade-bot/internal/config"
	"github.com/letieu/trade-bot/internal/types"
)

// MockProvider implements types.MarketDataProvider
type MockProvider struct {
	candles []types.Candle
}

func (m *MockProvider) GetSymbols() ([]string, error) {
	return []string{"BTCUSDT"}, nil
}

func (m *MockProvider) GetCandles(symbol, interval string, limit int, endTime int64) ([]types.Candle, error) {
	return m.candles, nil
}

// MockSender implements types.NotificationSender
type MockSender struct {
	Signals []types.Signal
}

func (m *MockSender) SendSignals(signals []types.Signal) error {
	m.Signals = append(m.Signals, signals...)
	return nil
}

func (m *MockSender) SendMessage(message string) error {
	return nil
}

func TestBot_Scan_Integration(t *testing.T) {
	// Setup Mock Data: 3 Red + 1 Green + 1 Forming (Red)
	// Chronological Order: Oldest -> Newest
	// We align 'now' to the hour to simulate a clean run, similar to how the bot works
	now := time.Now().Truncate(time.Hour)
	candles := []types.Candle{
		{Timestamp: now.Add(-4 * time.Hour).UnixMilli(), Open: 100, Close: 90}, // Red (Oldest)
		{Timestamp: now.Add(-3 * time.Hour).UnixMilli(), Open: 90, Close: 80},  // Red
		{Timestamp: now.Add(-2 * time.Hour).UnixMilli(), Open: 80, Close: 70},  // Red
		{Timestamp: now.Add(-1 * time.Hour).UnixMilli(), Open: 70, Close: 75},  // Green (Reversal!)
		{Timestamp: now.UnixMilli(), Open: 75, Close: 70},                      // Red (Forming - Should be ignored)
	}

	// We need 5 candles for GetRequiredCandles
	mockProvider := &MockProvider{candles: candles}
	mockSender := &MockSender{}

	// Setup Config
	cfg := &config.Config{
		Bot: config.BotConfig{
			EnabledIntervals: []string{"1h"},
			MaxConcurrency:   1,
			BatchSize:        1,
			Frontend:         "console",
			RunOnce:          true,
		},
	}

	// Create Bot with Mocks
	tradingBot := bot.NewBotWithDeps(cfg, mockProvider, mockSender)

	// Run Scan
	// Since RunOnce is true, Start() should call scan() once and return.
	if err := tradingBot.Start(); err != nil {
		t.Fatalf("Bot Start failed: %v", err)
	}

	// Verify Results
	if len(mockSender.Signals) != 1 {
		t.Fatalf("Expected 1 signal, got %d", len(mockSender.Signals))
	}

	signal := mockSender.Signals[0]
	if signal.Symbol != "BTCUSDT" {
		t.Errorf("Expected symbol BTCUSDT, got %s", signal.Symbol)
	}
	if signal.Trend != "bullish" {
		t.Errorf("Expected bullish trend, got %s", signal.Trend)
	}
	// Verify that the 'Green' candle (index 3) determined the trend
	// The signal.Candles should contain the 4 candles used for matching
	// Which are indices 0, 1, 2, 3.
	// Index 4 (Forming) should be excluded.
	
	lastCandle := signal.Candles[len(signal.Candles)-1]
	if lastCandle.Close <= lastCandle.Open {
		// It should be Green (Close > Open)
		t.Errorf("Expected last confirmed candle to be Green, got Red/Neutral (Open: %f, Close: %f)", lastCandle.Open, lastCandle.Close)
	}
}

func TestBot_DoesNotRemoveClosedCandle(t *testing.T) {
	// Scenario: Provider returns only closed candles (lag or just closed).
	// We expect the bot to KEEP the last candle and use it for analysis.
	
	// Time: 14:01. Last closed: 13:00-14:00. Open: 14:00-15:00.
	// Provider returns candles ending at 13:00 (Closed).
	now := time.Now().UTC().Truncate(time.Hour) // 14:00
	// We simulate running at 14:01, so 'now' in test setup matches the Open Start Time.
	
	// Candles: 3 Reds + 1 Green (The Green one is the one at 13:00)
	// If stripped (Old Bug), we see 3 Reds -> Bearish Signal.
	// If kept (Fix), we see 3 Reds + 1 Green -> Reversal Signal.
	
	candles := []types.Candle{
		{Timestamp: now.Add(-4 * time.Hour).UnixMilli(), Open: 100, Close: 90}, // Red
		{Timestamp: now.Add(-3 * time.Hour).UnixMilli(), Open: 90, Close: 80},  // Red
		{Timestamp: now.Add(-2 * time.Hour).UnixMilli(), Open: 80, Close: 70},  // Red
		{Timestamp: now.Add(-1 * time.Hour).UnixMilli(), Open: 70, Close: 75},  // Green (Closed!)
		// NO Open Candle provided
	}

	mockProvider := &MockProvider{candles: candles}
	mockSender := &MockSender{}

	cfg := &config.Config{
		Bot: config.BotConfig{
			EnabledIntervals: []string{"1h"},
			MaxConcurrency:   1,
			BatchSize:        1,
			Frontend:         "console",
			RunOnce:          true,
		},
	}

	tradingBot := bot.NewBotWithDeps(cfg, mockProvider, mockSender)

	if err := tradingBot.Start(); err != nil {
		t.Fatalf("Bot Start failed: %v", err)
	}

	// Analysis:
	// Old Logic: Removed Green. Saw 3 Reds. Match "Consecutive Candles" (Bearish).
	// New Logic: Keeps Green. Sees 3 Reds + Green. Match "Three Red + Green" (Bullish Reversal).
	
	if len(mockSender.Signals) == 0 {
		t.Fatalf("Expected signals, got 0")
	}

	// We might get multiple signals depending on strategies enabled.
	// "Three Red + Green" should match.
	// "Consecutive Candles" might NOT match (ends with Green).
	
	foundReversal := false
	foundConsecutive := false
	
	for _, sig := range mockSender.Signals {
		if sig.Pattern == "ĐẢO CHIỀU" { // "Three Red + Green" name
			foundReversal = true
		}
		if sig.Pattern == "TĂNG GIẢM LIÊN TỤC" {
			foundConsecutive = true
		}
	}
	
	if !foundReversal {
		t.Errorf("Expected 'ĐẢO CHIỀU' (Reversal) signal. Logic likely stripped the Green candle!")
	}
	
	if foundConsecutive {
		// If we kept the Green candle, "Consecutive" (3 Reds) shouldn't match because the last one is Green?
		// Wait, consecutive_candles.go checks:
		// lastCandle = Green. Count backwards. Green... Only 1 Green.
		// If MinCount=3, it should NOT match.
		// So if we find Consecutive, it means we stripped the Green candle and saw the Reds!
		t.Errorf("Found 'TĂNG GIẢM LIÊN TỤC' signal. This implies the last Green candle was stripped and the bot saw the previous Reds!")
	}
}
