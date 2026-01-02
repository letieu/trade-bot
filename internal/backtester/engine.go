package backtester

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/letieu/trade-bot/internal/types"
)

type Engine struct {
	provider types.MarketDataProvider
}

func NewEngine(provider types.MarketDataProvider) *Engine {
	return &Engine{
		provider: provider,
	}
}

func (e *Engine) RunTest(symbols []string, matcher types.PatternMatcher, interval string, startTime, endTime time.Time) (*types.BacktestResult, error) {
	log.Printf("Starting backtest for %d symbols from %s to %s", len(symbols), startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))

	result := &types.BacktestResult{
		StartTime:       startTime,
		EndTime:         endTime,
		SignalsBySymbol: make(map[string]int),
		SignalsByTime:   []types.TimeSignal{},
		MissingSignals:  []types.MissingSignal{},
	}

	for _, symbol := range symbols {
		signals, missing, err := e.backtestSymbol(symbol, matcher, interval, startTime, endTime)
		if err != nil {
			log.Printf("Error backtesting symbol %s: %v", symbol, err)
			continue
		}

		if len(signals) > 0 {
			result.SignalsBySymbol[symbol] = len(signals)
			result.SignalsByTime = append(result.SignalsByTime, signals...)
		}

		result.MissingSignals = append(result.MissingSignals, missing...)
	}

	result.TotalSignals = len(result.SignalsByTime)
	result.Duration = endTime.Sub(startTime)

	log.Printf("Backtest completed. Total signals: %d, Duration: %v", result.TotalSignals, result.Duration)

	return result, nil
}

func (e *Engine) backtestSymbol(symbol string, matcher types.PatternMatcher, interval string, startTime, endTime time.Time) ([]types.TimeSignal, []types.MissingSignal, error) {
	candles, err := e.provider.GetCandles(symbol, interval, 1000)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get candles for %s: %w", symbol, err)
	}

	if len(candles) < 4 {
		return nil, []types.MissingSignal{{
			Time:   time.Now(),
			Symbol: symbol,
			Reason: "insufficient candles",
		}}, nil
	}

	var signals []types.TimeSignal
	var missing []types.MissingSignal

	for i := 3; i < len(candles); i++ {
		window := candles[i-3 : i+1]

		candleTime := time.Unix(window[3].Timestamp/1000, 0)
		if candleTime.Before(startTime) || candleTime.After(endTime) {
			continue
		}

		matched, err := matcher.Match(window)
		if err != nil {
			missing = append(missing, types.MissingSignal{
				Time:   candleTime,
				Symbol: symbol,
				Reason: fmt.Sprintf("pattern match error: %v", err),
			})
			continue
		}

		if matched {
			signals = append(signals, types.TimeSignal{
				Time:    candleTime,
				Symbol:  symbol,
				Pattern: matcher.GetName(),
			})
		}
	}

	return signals, missing, nil
}

func (e *Engine) SaveResults(result *types.BacktestResult, filePath string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("failed to encode results: %w", err)
	}

	log.Printf("Results saved to %s", filePath)
	return nil
}
