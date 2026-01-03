package strategies

import (
	"fmt"

	"github.com/letieu/trade-bot/internal/types"
)

type ConsecutiveCandles struct {
	MinCount int
}

func NewConsecutiveCandles(minCount int) *ConsecutiveCandles {
	return &ConsecutiveCandles{MinCount: minCount}
}

func (s *ConsecutiveCandles) countConsecutive(candles []types.Candle) int {
	if len(candles) == 0 {
		return 0
	}

	// Check from the end backward to find consecutive candles of same color
	lastCandle := candles[len(candles)-1]
	targetColor := lastCandle.Color()
	consecutiveCount := 1

	for i := len(candles) - 2; i >= 0; i-- {
		if candles[i].Color() == targetColor {
			consecutiveCount++
		} else {
			break
		}
	}

	return consecutiveCount
}

func (s *ConsecutiveCandles) Match(candles []types.Candle) (bool, error) {
	if len(candles) < s.MinCount {
		return false, fmt.Errorf("need at least %d candles, got %d", s.MinCount, len(candles))
	}

	consecutiveCount := s.countConsecutive(candles)
	return consecutiveCount >= s.MinCount, nil
}

func (s *ConsecutiveCandles) GetMetadata(candles []types.Candle) map[string]interface{} {
	count := s.countConsecutive(candles)
	return map[string]interface{}{
		"consecutive_count": count,
	}
}

func (s *ConsecutiveCandles) GetName() string {
	return "TĂNG GIẢM LIÊN TỤC"
}

func (s *ConsecutiveCandles) GetDescription() string {
	return fmt.Sprintf("Detects at least %d consecutive candles of the same color", s.MinCount)
}

func (s *ConsecutiveCandles) GetRequiredCandles() int {
	return s.MinCount + 1
}
