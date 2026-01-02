package strategies

import (
	"testing"
	"time"

	"github.com/letieu/trade-bot/internal/types"
)

func TestThreeCandleReversal_Match(t *testing.T) {
	strategy := NewThreeCandleReversal()

	tests := []struct {
		name      string
		candles   []types.Candle
		wantMatch bool
	}{
		{
			name: "Bullish Reversal (Red, Red, Red, Green)",
			candles: []types.Candle{
				createCandle(100, 90),  // Red
				createCandle(90, 80),   // Red
				createCandle(80, 70),   // Red
				createCandle(70, 75),   // Green
			},
			wantMatch: true,
		},
		{
			name: "Bearish Reversal (Green, Green, Green, Red)",
			candles: []types.Candle{
				createCandle(10, 20),   // Green
				createCandle(20, 30),   // Green
				createCandle(30, 40),   // Green
				createCandle(40, 35),   // Red
			},
			wantMatch: true,
		},
		{
			name: "No Pattern (Red, Red, Red, Red)",
			candles: []types.Candle{
				createCandle(100, 90),
				createCandle(90, 80),
				createCandle(80, 70),
				createCandle(70, 60), // Red (should be Green for match)
			},
			wantMatch: false,
		},
		{
			name: "Mixed Pattern (Red, Green, Red, Green)",
			candles: []types.Candle{
				createCandle(100, 90),
				createCandle(90, 100),
				createCandle(100, 90),
				createCandle(90, 100),
			},
			wantMatch: false,
		},
		{
			name: "Not Enough Candles",
			candles: []types.Candle{
				createCandle(100, 90),
				createCandle(90, 80),
				createCandle(80, 70),
			},
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := strategy.Match(tt.candles)
			if (err != nil) != (tt.name == "Not Enough Candles") {
				// Only "Not Enough Candles" expects an error here? 
				// Actually strategy returns error if len < 4.
				if tt.name != "Not Enough Candles" {
					t.Errorf("Match() error = %v, wantErr %v", err, false)
				}
			}
			if got != tt.wantMatch {
				t.Errorf("Match() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func createCandle(open, close float64) types.Candle {
	return types.Candle{
		Timestamp: time.Now().Unix(),
		Open:      open,
		Close:     close,
		High:      max(open, close) + 1,
		Low:       min(open, close) - 1,
		Volume:    1000,
	}
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
