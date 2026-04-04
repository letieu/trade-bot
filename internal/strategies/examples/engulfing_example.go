package main

import (
	"fmt"
	"log"

	"github.com/letieu/trade-bot/internal/strategies"
	"github.com/letieu/trade-bot/internal/types"
)

// This example demonstrates how to use the Engulfing Anti-Fomo strategy
// with different configurations

func main() {
	// Example 1: Using default configuration
	fmt.Println("=== Example 1: Default Configuration ===")
	strategyDefault := strategies.NewEngulfingAntiFomoDefault()
	fmt.Printf("Strategy: %s\n", strategyDefault.GetName())
	fmt.Printf("Description: %s\n", strategyDefault.GetDescription())
	fmt.Printf("Required Candles: %d\n\n", strategyDefault.GetRequiredCandles())

	// Example 2: Using custom configuration - More aggressive (tighter RSI)
	fmt.Println("=== Example 2: Aggressive Configuration ===")
	aggressiveConfig := strategies.EngulfingAntiFomoConfig{
		RSILength:        14,
		RSIOverBought:    75,  // Higher threshold = fewer overbought signals
		RSIOverSold:      25,  // Lower threshold = fewer oversold signals
		SupertrendMult:   2.5, // Tighter supertrend
		SupertrendPeriod: 10,
		UseSupertrend:    true,
		SLBufferPercent:  20.0, // Tighter stop loss
	}
	_ = strategies.NewEngulfingAntiFomo(aggressiveConfig) // Create but don't use for this example
	fmt.Printf("RSI Range: %.0f - %.0f\n", aggressiveConfig.RSIOverSold, aggressiveConfig.RSIOverBought)
	fmt.Printf("SL Buffer: %.1f%%\n\n", aggressiveConfig.SLBufferPercent)

	// Example 3: Using custom configuration - Conservative (wider RSI)
	fmt.Println("=== Example 3: Conservative Configuration ===")
	conservativeConfig := strategies.EngulfingAntiFomoConfig{
		RSILength:        14,
		RSIOverBought:    65,  // Lower threshold = more overbought signals
		RSIOverSold:      35,  // Higher threshold = more oversold signals
		SupertrendMult:   3.5, // Wider supertrend
		SupertrendPeriod: 14,
		UseSupertrend:    true,
		SLBufferPercent:  30.0, // Wider stop loss for more breathing room
	}
	_ = strategies.NewEngulfingAntiFomo(conservativeConfig) // Create but don't use for this example
	fmt.Printf("RSI Range: %.0f - %.0f\n", conservativeConfig.RSIOverSold, conservativeConfig.RSIOverBought)
	fmt.Printf("SL Buffer: %.1f%%\n\n", conservativeConfig.SLBufferPercent)

	// Example 4: Without Supertrend filter (RSI + Engulfing only)
	fmt.Println("=== Example 4: No Supertrend Filter ===")
	noSupertrendConfig := strategies.EngulfingAntiFomoConfig{
		RSILength:        14,
		RSIOverBought:    70,
		RSIOverSold:      30,
		SupertrendMult:   3.0,
		SupertrendPeriod: 10,
		UseSupertrend:    false, // Disabled
		SLBufferPercent:  25.0,
	}
	strategyNoST := strategies.NewEngulfingAntiFomo(noSupertrendConfig)
	fmt.Printf("Supertrend Enabled: %v\n", noSupertrendConfig.UseSupertrend)
	fmt.Printf("Required Candles: %d\n\n", strategyNoST.GetRequiredCandles())

	// Example 5: Checking signals with sample data
	fmt.Println("=== Example 5: Checking Signals ===")
	candles := createSampleCandles()

	matched, err := strategyDefault.Match(candles)
	if err != nil {
		log.Printf("Error: %v\n", err)
	} else if matched {
		fmt.Println("Signal detected!")
		metadata := strategyDefault.GetMetadata(candles)
		fmt.Printf("RSI: %.2f\n", metadata["rsi"].(float64))
		fmt.Printf("Stop Loss: %.2f\n", metadata["stop_loss"].(float64))
		fmt.Printf("Supertrend Bullish: %v\n", metadata["supertrend_bullish"].(bool))
	} else {
		fmt.Println("No signal detected")
	}
}

// createSampleCandles creates sample candle data for demonstration
func createSampleCandles() []types.Candle {
	candles := make([]types.Candle, 0)

	// Create a downtrend with oversold conditions
	price := 100.0
	for i := 0; i < 30; i++ {
		candles = append(candles, types.Candle{
			Open:  price,
			High:  price + 2,
			Low:   price - 3,
			Close: price - 2,
		})
		price -= 2
	}

	// Add bullish engulfing
	candles = append(candles, types.Candle{
		Open:  price,
		High:  price + 2,
		Low:   price - 3,
		Close: price - 2,
	}) // Red

	candles = append(candles, types.Candle{
		Open:  price - 2,
		High:  price + 1,
		Low:   price - 3,
		Close: price,
	}) // Green engulfing

	// Add three red candles
	candles = append(candles, types.Candle{
		Open:  price,
		High:  price + 1,
		Low:   price - 2,
		Close: price - 1,
	})
	candles = append(candles, types.Candle{
		Open:  price - 1,
		High:  price,
		Low:   price - 3,
		Close: price - 2,
	})
	candles = append(candles, types.Candle{
		Open:  price - 2,
		High:  price - 1,
		Low:   price - 4,
		Close: price - 3,
	})

	// Add green confirmation
	candles = append(candles, types.Candle{
		Open:  price - 3,
		High:  price - 1,
		Low:   price - 4,
		Close: price - 2,
	})

	return candles
}
