package main

import (
	"flag"
	"log"
	"time"

	"github.com/letieu/trade-bot/internal/bot"
	"github.com/letieu/trade-bot/internal/config"
)

func main() {
	var (
		configFile = flag.String("config", "", "Path to config file (optional, uses env vars by default)")
		runOnce    = flag.Bool("once", false, "Run the bot once and exit")
		timeStr    = flag.String("time", "", "Target time for scan (RFC3339 format, e.g., 2026-01-02T15:04:05Z)")
	)
	flag.Parse()

	cfg := config.Load(*configFile)

	if *runOnce {
		cfg.Bot.RunOnce = true
	}

	if *timeStr != "" {
		t, err := time.Parse(time.RFC3339, *timeStr)
		if err != nil {
			log.Fatalf("Invalid time format (use RFC3339): %v", err)
		}
		cfg.Bot.TargetTime = t.UnixMilli()
		cfg.Bot.RunOnce = true // -time implies -once for logic clarity
	}

	tradingBot := bot.NewBot(cfg)
	if err := tradingBot.Start(); err != nil {
		log.Fatalf("Failed to start trading bot: %v", err)
	}
}
