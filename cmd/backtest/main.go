package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/letieu/trade-bot/internal/backtester"
	"github.com/letieu/trade-bot/internal/config"
	"github.com/letieu/trade-bot/internal/providers/bybit"
	"github.com/letieu/trade-bot/internal/strategies"
)

func main() {
	var (
		interval   = flag.String("interval", "1h", "Time interval (1m, 5m, 15m, 30m, 1h, 4h, 1d)")
		symbols    = flag.String("symbols", "", "Comma-separated list of symbols (empty = all USDT symbols)")
		save       = flag.Bool("save", true, "Save results to file")
		output     = flag.String("output", "./results", "Output directory for results")
		configFile = flag.String("config", "", "Path to config file (optional, uses env vars by default)")
	)
	flag.Parse()

	cfg := config.Load(*configFile)

	backtestStart, backtestEnd, err := cfg.GetBacktestTimes()
	if err != nil {
		log.Fatalf("Failed to parse backtest times: %v", err)
	}

	bybitClient := bybit.NewClient(&cfg.Bybit)

	engine := backtester.NewEngine(bybitClient)
	strategy := strategies.NewThreeCandleReversal()

	var symbolList []string
	if *symbols != "" {
		symbolList = strings.Split(*symbols, ",")
		for i, s := range symbolList {
			symbolList[i] = strings.TrimSpace(s)
		}
	} else {
		allSymbols, err := bybitClient.GetSymbols()
		if err != nil {
			log.Fatalf("Failed to get symbols: %v", err)
		}
		symbolList = allSymbols
	}

	fmt.Printf("Running backtest with %s strategy on %d symbols\n", strategy.GetName(), len(symbolList))
	fmt.Printf("Time range: %s to %s\n", backtestStart.Format(time.RFC3339), backtestEnd.Format(time.RFC3339))
	fmt.Printf("Interval: %s\n", *interval)

	result, err := engine.RunTest(symbolList, strategy, *interval, backtestStart, backtestEnd)
	if err != nil {
		log.Fatalf("Backtest failed: %v", err)
	}

	fmt.Printf("\nBacktest Results:\n")
	fmt.Printf("Total Signals: %d\n", result.TotalSignals)
	fmt.Printf("Duration: %v\n", result.Duration)
	fmt.Printf("Signals by Symbol:\n")
	for symbol, count := range result.SignalsBySymbol {
		fmt.Printf("  %s: %d\n", symbol, count)
	}

	if *save {
		filename := fmt.Sprintf("backtest_%s_%s_%s.json",
			strategy.GetName(), *interval, time.Now().Format("20060102_150405"))
		filePath := filepath.Join(*output, filename)

		if err := engine.SaveResults(result, filePath); err != nil {
			log.Printf("Failed to save results: %v", err)
		}
	}
}
