package bot

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/letieu/trade-bot/internal/config"
	"github.com/letieu/trade-bot/internal/frontends/console"
	"github.com/letieu/trade-bot/internal/frontends/telegram"
	"github.com/letieu/trade-bot/internal/providers/bybit"
	"github.com/letieu/trade-bot/internal/strategies"
	"github.com/letieu/trade-bot/internal/types"
)

type Bot struct {
	config     *config.Config
	provider   types.MarketDataProvider
	sender     types.NotificationSender
	strategies []types.PatternMatcher
}

func NewBot(cfg *config.Config) *Bot {
	bybitClient := bybit.NewClient(&cfg.Bybit)

	var sender types.NotificationSender
	var err error

	switch cfg.Bot.Frontend {
	case "console":
		sender = console.NewBot()
	case "telegram":
		sender, err = telegram.NewBot(&cfg.Telegram)
		if err != nil {
			log.Fatalf("Failed to create telegram bot: %v", err)
		}
	default:
		log.Printf("Unknown frontend '%s', defaulting to telegram", cfg.Bot.Frontend)
		sender, err = telegram.NewBot(&cfg.Telegram)
		if err != nil {
			log.Fatalf("Failed to create telegram bot: %v", err)
		}
	}

	// Initialize multiple strategies
	strategies := []types.PatternMatcher{
		strategies.NewThreeCandleReversal(),
		strategies.NewConsecutiveCandles(3),
	}

	return &Bot{
		config:     cfg,
		provider:   bybitClient,
		sender:     sender,
		strategies: strategies,
	}
}

// NewBotWithDeps allows creating a bot with injected dependencies (useful for testing)
func NewBotWithDeps(cfg *config.Config, provider types.MarketDataProvider, sender types.NotificationSender) *Bot {
	strategies := []types.PatternMatcher{
		strategies.NewThreeCandleReversal(),
		strategies.NewConsecutiveCandles(3),
	}
	return &Bot{
		config:     cfg,
		provider:   provider,
		sender:     sender,
		strategies: strategies,
	}
}

func (b *Bot) Start() error {
	strategyNames := make([]string, len(b.strategies))
	for i, s := range b.strategies {
		strategyNames[i] = s.GetName()
	}
	log.Printf("Starting trading bot with %d strategies: %v", len(b.strategies), strategyNames)

	if b.config.Bot.RunOnce {
		log.Println("Running in one-time mode")
		return b.scan()
	}

	var wg sync.WaitGroup
	for _, interval := range b.config.Bot.EnabledIntervals {
		wg.Add(1)
		go func(intervalStr string) {
			defer wg.Done()
			b.runIntervalLoop(intervalStr)
		}(interval)
	}

	wg.Wait()
	return nil
}

func (b *Bot) runIntervalLoop(intervalStr string) {
	duration, err := types.ParseInterval(intervalStr)
	if err != nil {
		log.Printf("[%s] Failed to parse interval: %v", intervalStr, err)
		return
	}

	for {
		now := time.Now().UTC()
		next := now.Truncate(duration).Add(duration)
		// Add a small delay to ensure the candle is fully closed and data is available on the provider side
		next = next.Add(60 * time.Second)

		sleepDuration := time.Until(next)
		log.Printf("[%s] Next scan in %v at %v", intervalStr, sleepDuration.Round(time.Second), next.Local().Format("15:04:05"))

		timer := time.NewTimer(sleepDuration)
		<-timer.C

		log.Printf("[%s] Starting scan...", intervalStr)
		if err := b.scanSpecificInterval(intervalStr); err != nil {
			log.Printf("[%s] Error during scan: %v", intervalStr, err)
		}
	}
}

func (b *Bot) scanSpecificInterval(interval string) error {
	symbols, err := b.provider.GetSymbols()
	if err != nil {
		return fmt.Errorf("failed to get symbols: %w", err)
	}

	signals := b.scanInterval(symbols, interval)

	if len(signals) > 0 {
		log.Printf("[%s] Found %d signals, sending result", interval, len(signals))
		if err := b.sender.SendSignals(signals); err != nil {
			return fmt.Errorf("failed to send signals: %w", err)
		}
	} else {
		log.Printf("[%s] No signals found", interval)
	}

	return nil
}

func (b *Bot) scan() error {
	symbols, err := b.provider.GetSymbols()
	if err != nil {
		return fmt.Errorf("failed to get symbols: %w", err)
	}

	log.Printf("Scanning %d symbols for patterns", len(symbols))

	var wg sync.WaitGroup
	signalsChan := make(chan []types.Signal, len(b.config.Bot.EnabledIntervals))

	for _, interval := range b.config.Bot.EnabledIntervals {
		wg.Add(1)
		go func(intervalStr string) {
			defer wg.Done()
			signals := b.scanInterval(symbols, intervalStr)
			signalsChan <- signals
		}(interval)
	}

	wg.Wait()
	close(signalsChan)

	allSignals := make([]types.Signal, 0)
	for signals := range signalsChan {
		allSignals = append(allSignals, signals...)
	}

	if len(allSignals) > 0 {
		log.Printf("Found %d signals, sending result", len(allSignals))
		if err := b.sender.SendSignals(allSignals); err != nil {
			return fmt.Errorf("failed to send signals: %w", err)
		}
	} else {
		log.Printf("No signals found in this scan")
	}

	return nil
}

func (b *Bot) scanInterval(symbols []string, interval string) []types.Signal {
	var signals []types.Signal
	var mu sync.Mutex

	semaphore := make(chan struct{}, b.config.Bot.MaxConcurrency)
	var wg sync.WaitGroup

	for i := 0; i < len(symbols); i += b.config.Bot.BatchSize {
		end := i + b.config.Bot.BatchSize
		if end > len(symbols) {
			end = len(symbols)
		}

		batch := symbols[i:end]

		for _, symbol := range batch {
			wg.Add(1)
			go func(sym string) {
				defer wg.Done()

				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				// Check all strategies for this symbol
				symbolSignals := b.checkSymbol(sym, interval)
				if len(symbolSignals) > 0 {
					mu.Lock()
					signals = append(signals, symbolSignals...)
					mu.Unlock()
				}
			}(symbol)
		}

		time.Sleep(300 * time.Millisecond)
	}

	wg.Wait()
	return signals
}

func (b *Bot) checkSymbol(symbol, interval string) []types.Signal {
	// Get the maximum required candles across all strategies
	maxRequired := 0
	for _, strategy := range b.strategies {
		if req := strategy.GetRequiredCandles(); req > maxRequired {
			maxRequired = req
		}
	}

	candles, err := b.provider.GetCandles(symbol, interval, maxRequired, b.config.Bot.TargetTime)
	if err != nil {
		log.Printf("Failed to get candles for %s: %v", symbol, err)
		return nil
	}

	// Exclude the last candle if it is incomplete/forming to ensure confirmed signals
	if len(candles) > 0 {
		duration, err := types.ParseInterval(interval)
		if err != nil {
			log.Printf("Failed to parse interval %s for symbol %s: %v", interval, symbol, err)
			return nil
		}

		// Calculate the expected start time of the current OPEN candle
		// We use UTC to match Bybit's time standard
		currentOpenTime := time.Now().UTC().Truncate(duration).UnixMilli()
		lastCandle := candles[len(candles)-1]

		// Only remove the last candle if it matches the current open interval
		if lastCandle.Timestamp == currentOpenTime {
			candles = candles[:len(candles)-1]
		} else {
			log.Printf("%s dont need remove last, open: %d", symbol, currentOpenTime)
		}
	}

	if len(candles) < 4 {
		// Not enough closed candles
		return nil
	}

	var signals []types.Signal

	// Check all strategies
	for _, strategy := range b.strategies {
		matched, err := strategy.Match(candles)
		if err != nil {
			log.Printf("Error matching pattern %s for %s: %v", strategy.GetName(), symbol, err)
			continue
		}

		if !matched {
			continue
		}

		lastCandles := candles[len(candles)-4:]
		lastCandle := candles[len(candles)-1]

		// Get metadata from strategy (e.g., consecutive count)
		metadata := strategy.GetMetadata(candles)
		consecutiveCount := 0
		if count, ok := metadata["consecutive_count"].(int); ok {
			consecutiveCount = count
		}

		signal := types.Signal{
			Symbol:           symbol,
			Interval:         interval,
			Pattern:          strategy.GetName(),
			Trend:            "bullish",
			Price:            lastCandle.Close,
			RSI:              0,
			EMA:              0,
			Volume:           lastCandle.Volume,
			Timestamp:        time.Now(),
			Candles:          lastCandles,
			ConsecutiveCount: consecutiveCount,
		}

		if lastCandles[len(lastCandles)-1].Color() == types.ColorRed {
			signal.Trend = "bearish"
		}

		log.Printf("Signal found: %s %s %s", symbol, interval, signal.Pattern)
		signals = append(signals, signal)
	}

	return signals
}
