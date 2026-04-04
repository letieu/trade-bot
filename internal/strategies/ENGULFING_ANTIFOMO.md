# Engulfing Anti-Fomo + Supertrend Strategy

## Overview

This strategy combines multiple technical indicators to identify high-probability trading opportunities while avoiding FOMO (Fear Of Missing Out) entries.

## Strategy Components

### 1. Engulfing Patterns
- **Bullish Engulfing**: Current candle closes above previous candle's open, and previous candle was red
- **Bearish Engulfing**: Current candle closes below previous candle's open, and previous candle was green

### 2. RSI (Relative Strength Index)
- **Oversold**: RSI ≤ 30 (configurable)
- **Overbought**: RSI ≥ 70 (configurable)
- Checks current candle and previous 2 candles for oversold/overbought conditions

### 3. Anti-FOMO Confirmation
- **For Long Signals**: Waits for 3 consecutive red candles followed by a green confirmation candle
- **For Short Signals**: Waits for 3 consecutive green candles followed by a red confirmation candle

### 4. Supertrend Filter (Optional)
- **Bullish Supertrend**: Direction < 0 (allows long entries)
- **Bearish Supertrend**: Direction > 0 (allows short entries)
- Can be disabled via configuration

## Signal Logic

### Long Signal
1. RSI is oversold (current or previous 2 candles)
2. Bullish engulfing pattern occurs
3. Signal is set to "pending long"
4. Wait for 3 consecutive red candles
5. Green confirmation candle appears
6. (Optional) Supertrend is bullish
7. **LONG ENTRY**

### Short Signal
1. RSI is overbought (current or previous 2 candles)
2. Bearish engulfing pattern occurs
3. Signal is set to "pending short"
4. Wait for 3 consecutive green candles
5. Red confirmation candle appears
6. (Optional) Supertrend is bearish
7. **SHORT ENTRY**

## Stop Loss Calculation

Stop loss is calculated using cluster range method:
- **Cluster Low**: Minimum of current and previous candle's low
- **Cluster High**: Maximum of current and previous candle's high
- **Cluster Range**: Cluster High - Cluster Low

**For Long Positions:**
```
SL = Cluster Low - (Cluster Range × SL Buffer %)
```

**For Short Positions:**
```
SL = Cluster High + (Cluster Range × SL Buffer %)
```

Default SL Buffer: 25%

## Configuration

```go
type EngulfingAntiFomoConfig struct {
    RSILength        int     // Default: 14
    RSIOverBought    float64 // Default: 70
    RSIOverSold      float64 // Default: 30
    SupertrendMult   float64 // Default: 3.0
    SupertrendPeriod int     // Default: 10
    UseSupertrend    bool    // Default: true
    SLBufferPercent  float64 // Default: 25.0
}
```

## Usage

### Using Default Configuration
```go
import "github.com/letieu/trade-bot/internal/strategies"

strategy := strategies.NewEngulfingAntiFomoDefault()
```

### Using Custom Configuration
```go
import "github.com/letieu/trade-bot/internal/strategies"

config := strategies.EngulfingAntiFomoConfig{
    RSILength:        14,
    RSIOverBought:    75,
    RSIOverSold:      25,
    SupertrendMult:   3.0,
    SupertrendPeriod: 10,
    UseSupertrend:    true,
    SLBufferPercent:  20.0,
}

strategy := strategies.NewEngulfingAntiFomo(config)
```

### Checking for Signals
```go
candles := // ... get candles from provider
matched, err := strategy.Match(candles)
if err != nil {
    // handle error
}

if matched {
    metadata := strategy.GetMetadata(candles)
    rsi := metadata["rsi"].(float64)
    stopLoss := metadata["stop_loss"].(float64)
    // ... use signal data
}
```

## Minimum Candles Required

The strategy requires at least:
```
4 (confirmation) + RSI Length (14) + Supertrend Period (10) + 1 (calculation) = 29 candles
```

With default settings, you need minimum 29 candles of historical data.

## Indicators Package

The strategy uses two technical indicators from the `internal/indicators` package:

### RSI (Relative Strength Index)
```go
import "github.com/letieu/trade-bot/internal/indicators"

rsi := indicators.CalculateRSI(candles, 14)
```

### Supertrend
```go
import "github.com/letieu/trade-bot/internal/indicators"

st := indicators.CalculateSupertrend(candles, 10, 3.0)
if st.IsBullish() {
    // bullish trend
}
```

## Notes

- This strategy is designed to reduce FOMO by requiring multiple confirmations
- The 3-candle wait period helps filter out false signals
- Supertrend filter adds an additional layer of trend confirmation
- Stop loss is automatically calculated based on recent price action
- The strategy maintains state (`signalPendingLong`, `signalPendingShort`) between calls

## Original Source

Based on TradingView script "Engulfing Anti-Fomo + Supertrend" by ahmedirshad419
- Mozilla Public License 2.0
- Updated by Gemini with SL Buffer default 25%
