package main

import (
	"flag"
	"log"

	"github.com/letieu/trade-bot/internal/bot"
	"github.com/letieu/trade-bot/internal/config"
)

func main() {
	var (
		configFile = flag.String("config", "", "Path to config file (optional, uses env vars by default)")
		runOnce    = flag.Bool("once", false, "Run the bot once and exit")
	)
	flag.Parse()

	cfg := config.Load(*configFile)
	
	if *runOnce {
		cfg.Bot.RunOnce = true
	}

	tradingBot := bot.NewBot(cfg)
	if err := tradingBot.Start(); err != nil {
		log.Fatalf("Failed to start trading bot: %v", err)
	}
}
