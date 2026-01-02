# Trading Bot

A flexible cryptocurrency trading bot built in Go that scans for technical patterns and sends signals to Telegram.

## Features

- **Flexible Architecture**: Easy to swap exchanges, strategies, and notification frontends
- **Real-time Market Data**: Integrates with Bybit for live crypto market data
- **Pattern Recognition**: Implements the "3 Red + Green Reversal" strategy (easily extensible)
- **Telegram Notifications**: Sends trading signals directly to your Telegram chat
- **Backtesting Framework**: Test strategies on historical data
- **Rate Limited**: Respects API rate limits to avoid being blocked

## Architecture

The bot is designed with flexibility in mind:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Market Data   │    │  Pattern       │    │  Notification  │
│   Provider     │───▶│  Matcher       │───▶│  Sender        │
│  (Bybit/...)   │    │ (Strategy)      │    │ (Telegram/...) │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Key Components

- **MarketDataProvider**: Interface for different exchanges (currently Bybit)
- **PatternMatcher**: Interface for trading strategies
- **NotificationSender**: Interface for different notification channels
- **Backtester**: Framework for testing strategies on historical data

## Quick Start

### 1. Configuration

You can configure the bot using either YAML configuration files (recommended) or environment variables.

#### Method 1: YAML Configuration (Recommended)

Create a `trade-bot.yaml` file in the project root:

```yaml
telegram:
  botToken: "your_bot_token"
  chatId: "your_chat_id"

bybit:
  baseUrl: "https://api.bybit.com"
  timeout: "10s"
  rateLimit: 20
  headers:
    Content-Type: "application/json"

bot:
  scanInterval: "1m"
  batchSize: 20
  maxConcurrency: 5
  enabledIntervals:
    - "1h"
    - "4h" 
    - "1d"

backtest:
  startTime: "2025-01-01T00:00:00Z"
  endTime: "2025-02-01T00:00:00Z"
  dataPath: "./data"
  saveResults: true
  resultsPath: "./results"
```

#### Method 2: Environment Variables (Legacy)

Set up environment variables:

```bash
export TELEGRAM_BOT_TOKEN="your_bot_token"
export TELEGRAM_CHAT_ID="your_chat_id"
export BOT_ENABLED_INTERVALS="1h,4h,1d"  # Optional
export BOT_SCAN_INTERVAL="1m"               # Optional
export BOT_BATCH_SIZE="20"                   # Optional
export BOT_MAX_CONCURRENCY="5"               # Optional
```

**Configuration Priority**: 
1. Specified config file (`-config=custom.yaml`)
2. Default `trade-bot.yaml` file
3. Environment variables (fallback only when no config file found)

### 2. Build the Bot

```bash
go mod download
go build -o trade-bot ./cmd/trade-bot
go build -o backtest ./cmd/backtest
```

### 3. Run the Bot

```bash
# Start the live trading bot
./trade-bot

# Or run backtests
./backtest -interval=1h -days=30 -symbols="BTCUSDT,ETHUSDT"
```

## Commands

### Trading Bot

```bash
./trade-bot [flags]
```

Flags:
- `-config`: Path to config file (optional, defaults to `trade-bot.yaml`, falls back to env vars)

### Backtesting

```bash
./backtest [flags]
```

Flags:
- `-config`: Path to config file (optional, defaults to `trade-bot.yaml`)
- `-interval`: Time interval (1m, 5m, 15m, 30m, 1h, 4h, 1d) [default: 1h]
- `-days`: Number of days to backtest [default: 30]
- `-symbols`: Comma-separated list of symbols (empty = all USDT symbols)
- `-save`: Save results to file [default: true]
- `-output`: Output directory for results [default: ./results]

**Using YAML Configuration with Backtest:**

The backtest tool reads backtest-specific settings from the YAML file:

```bash
# Use default config file
./backtest -interval=4h -days=30 -symbols="BTCUSDT,ETHUSDT"

# Use custom config file
./backtest -config=my-custom-config.yaml -interval=1d
```

## Strategy: Three Red + Green Reversal

The bot currently implements one strategy that detects:

1. Three consecutive red (bearish) candles
2. Followed by a green (bullish) candle
3. Indicates a potential bullish reversal

## Adding New Strategies

Create a new file in `internal/strategies/`:

```go
package strategies

import "github.com/letieu/trade-bot/internal/types"

type MyStrategy struct{}

func NewMyStrategy() *MyStrategy {
    return &MyStrategy{}
}

func (s *MyStrategy) Match(candles []types.Candle) (bool, error) {
    // Your pattern matching logic here
    return false, nil
}

func (s *MyStrategy) GetName() string {
    return "My Custom Strategy"
}

func (s *MyStrategy) GetDescription() string {
    return "Description of what this strategy does"
}
```

## Adding New Exchanges

Implement the `MarketDataProvider` interface in `internal/providers/`:

```go
type MyExchange struct{}

func (e *MyExchange) GetSymbols() ([]string, error) { /* ... */ }
func (e *MyExchange) GetCandles(symbol, interval string, limit int) ([]types.Candle, error) { /* ... */ }
func (e *MyExchange) GetTickerInfo(symbol string) (types.TickerInfo, error) { /* ... */ }
```

## Adding New Notification Channels

Implement the `NotificationSender` interface in `internal/frontends/`:

```go
type MyNotifier struct{}

func (n *MyNotifier) SendSignals(signals []types.Signal) error { /* ... */ }
func (n *MyNotifier) SendMessage(message string) error { /* ... */ }
```

## Configuration Options

### YAML Configuration (Recommended)

Copy `trade-bot.yaml.example` to `trade-bot.yaml` and customize:

```bash
cp trade-bot.yaml.example trade-bot.yaml
```

#### Configuration Sections:

**Telegram Configuration**
- `telegram.botToken`: Bot token from @BotFather
- `telegram.chatId`: Chat ID to send messages to

**Bot Configuration**
- `bot.scanInterval`: How often to scan for patterns (default: 1m)
- `bot.batchSize`: Number of symbols to process in parallel (default: 20)
- `bot.maxConcurrency`: Maximum concurrent goroutines (default: 5)
- `bot.enabledIntervals`: List of intervals to scan (default: ["1h", "4h", "1d"])

**Bybit Configuration**
- `bybit.baseUrl`: API base URL (default: https://api.bybit.com)
- `bybit.timeout`: Request timeout (default: 10s)
- `bybit.rateLimit`: Rate limit per request (default: 20)
- `bybit.headers`: HTTP headers for API requests

**Backtest Configuration**
- `backtest.startTime`: Backtest start time in RFC3339 format
- `backtest.endTime`: Backtest end time in RFC3339 format
- `backtest.dataPath`: Path to store cached data (default: ./data)
- `backtest.saveResults`: Save backtest results (default: true)
- `backtest.resultsPath`: Path to save results (default: ./results)

### Environment Variables (Legacy)

If no YAML config file is found, the bot falls back to environment variables:

**Telegram Configuration**
- `TELEGRAM_BOT_TOKEN`: Bot token from @BotFather
- `TELEGRAM_CHAT_ID`: Chat ID to send messages to

**Bot Configuration**
- `BOT_SCAN_INTERVAL`: How often to scan for patterns (default: 1m)
- `BOT_BATCH_SIZE`: Number of symbols to process in parallel (default: 20)
- `BOT_MAX_CONCURRENCY`: Maximum concurrent goroutines (default: 5)
- `BOT_ENABLED_INTERVALS`: Comma-separated intervals to scan (default: 1h,4h,1d)

**Bybit Configuration**
- `BYBIT_BASE_URL`: API base URL (default: https://api.bybit.com)
- `BYBIT_TIMEOUT`: Request timeout (default: 10s)
- `BYBIT_RATE_LIMIT`: Rate limit per request (default: 20)

**Backtest Configuration**
- `BACKTEST_DATA_PATH`: Path to store cached data (default: ./data)
- `BACKTEST_SAVE_RESULTS`: Save backtest results (default: true)
- `BACKTEST_RESULTS_PATH`: Path to save results (default: ./results)

## Project Structure

```
├── cmd/
│   ├── trade-bot/     # Main bot application
│   └── backtest/      # Backtesting CLI
├── internal/
│   ├── backtester/     # Backtesting engine
│   ├── bot/           # Main bot orchestrator
│   ├── config/        # Configuration management
│   ├── frontends/     # Notification senders
│   ├── providers/     # Market data providers
│   ├── strategies/     # Pattern matching strategies
│   └── types/         # Common types and interfaces
├── go.mod
├── go.sum
└── README.md
```

## Getting a Telegram Bot

1. Start a chat with [@BotFather](https://t.me/BotFather) on Telegram
2. Send `/newbot`
3. Follow the instructions to create your bot
4. Copy the bot token
5. Start a chat with your new bot and send any message
6. Get your chat ID by sending a message to [@userinfobot](https://t.me/userinfobot)

## License

This project is open source and available under the [MIT License](LICENSE).