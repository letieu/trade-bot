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
	now := time.Now()
	candles := []types.Candle{
		{Timestamp: now.Add(-4 * time.Hour).Unix(), Open: 100, Close: 90}, // Red (Oldest)
		{Timestamp: now.Add(-3 * time.Hour).Unix(), Open: 90, Close: 80},  // Red
		{Timestamp: now.Add(-2 * time.Hour).Unix(), Open: 80, Close: 70},  // Red
		{Timestamp: now.Add(-1 * time.Hour).Unix(), Open: 70, Close: 75},  // Green (Reversal!)
		{Timestamp: now.Unix(), Open: 75, Close: 70},                      // Red (Forming - Should be ignored)
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
			ScanInterval:     time.Minute,
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