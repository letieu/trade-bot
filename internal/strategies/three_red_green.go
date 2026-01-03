package strategies

import (
	"fmt"

	"github.com/letieu/trade-bot/internal/types"
)

type ThreeCandleReversal struct{}

func NewThreeCandleReversal() *ThreeCandleReversal {
	return &ThreeCandleReversal{}
}

func (s *ThreeCandleReversal) Match(candles []types.Candle) (bool, error) {
	if len(candles) < 4 {
		return false, fmt.Errorf("need at least 4 candles, got %d", len(candles))
	}

	lastFour := candles[len(candles)-4:]

	colors := make([]types.CandleColor, 4)
	for i, candle := range lastFour {
		colors[i] = candle.Color()
	}

	firstThreeSame := colors[0] == colors[1] && colors[1] == colors[2]
	lastIsOpposite := colors[3] != colors[2]

	if !firstThreeSame || !lastIsOpposite {
		return false, nil
	}

	return true, nil
}

func (s *ThreeCandleReversal) GetName() string {
	return "ĐẢO CHIỀU"
}

func (s *ThreeCandleReversal) GetDescription() string {
	return "Detects 3 consecutive candles of the same color followed by an opposite color candle"
}

func (s *ThreeCandleReversal) GetRequiredCandles() int {
	return 5
}

func (s *ThreeCandleReversal) GetMetadata(candles []types.Candle) map[string]interface{} {
	return map[string]interface{}{} // No additional metadata for this pattern
}
