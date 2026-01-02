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
	config   *config.Config
	provider types.MarketDataProvider
	sender   types.NotificationSender
	strategy types.PatternMatcher
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

	strategy := strategies.NewThreeCandleReversal()

	return &Bot{
		config:   cfg,
		provider: bybitClient,
		sender:   sender,
		strategy: strategy,
	}
}

func (b *Bot) Start() error {
	log.Printf("Starting trading bot with strategy: %s", b.strategy.GetName())

	if b.config.Bot.RunOnce {
		log.Println("Running in one-time mode")
		return b.scan()
	}

	ticker := time.NewTicker(b.config.Bot.ScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := b.scan(); err != nil {
				log.Printf("Error during scan: %v", err)
			}
		}
	}
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

				if signal := b.checkSymbol(sym, interval); signal != nil {
					mu.Lock()
					signals = append(signals, *signal)
					mu.Unlock()
				}
			}(symbol)
		}

		time.Sleep(300 * time.Millisecond)
	}

	wg.Wait()
	return signals
}

func (b *Bot) checkSymbol(symbol, interval string) *types.Signal {
	requiredCandles := b.strategy.GetRequiredCandles()
	candles, err := b.provider.GetCandles(symbol, interval, requiredCandles)
	if err != nil {
		log.Printf("Failed to get candles for %s: %v", symbol, err)
		return nil
	}

	// Exclude the last candle (incomplete/forming) to ensure confirmed signals
	if len(candles) > 0 {
		candles = candles[:len(candles)-1]
	}

	if len(candles) < 4 {
		// Not enough closed candles
		return nil
	}

	matched, err := b.strategy.Match(candles)
	if err != nil {
		log.Printf("Error matching pattern for %s: %v", symbol, err)
		return nil
	}

	if !matched {
		// log.Printf("Skip %s", symbol)
		return nil
	}

	tickerInfo, err := b.provider.GetTickerInfo(symbol)
	if err != nil {
		log.Printf("Failed to get ticker info for %s: %v", symbol, err)
	}

	lastCandles := candles[len(candles)-4:]

	signal := types.Signal{
		Symbol:    symbol,
		Interval:  interval,
		Pattern:   b.strategy.GetName(),
		Trend:     "bullish",
		Price:     tickerInfo.LastPrice,
		RSI:       0,
		EMA:       0,
		Volume:    tickerInfo.Volume24h,
		Timestamp: time.Now(),
		Candles:   lastCandles,
	}

	if lastCandles[len(lastCandles)-1].Color() == types.ColorRed {
		signal.Trend = "bearish"
	}

	log.Printf("Signal found: %s %s %s", symbol, interval, signal.Pattern)

	return &signal
}
