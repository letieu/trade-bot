package console

import (
	"fmt"
	"strings"

	"github.com/letieu/trade-bot/internal/types"
)

type Bot struct {
}

func NewBot() *Bot {
	return &Bot{}
}

func (b *Bot) SendSignals(signals []types.Signal) error {
	if len(signals) == 0 {
		return nil
	}

	message := b.formatSignalsMessage(signals)
	fmt.Println(message)
	return nil
}

func (b *Bot) SendMessage(message string) error {
	fmt.Println(message)
	return nil
}

func (b *Bot) formatSignalsMessage(signals []types.Signal) string {
	if len(signals) == 0 {
		return "No trading signals found"
	}

	var builder strings.Builder

	pattern := signals[0].Pattern
	interval := signals[0].Interval

	builder.WriteString(fmt.Sprintf("\n=== \033[1m%s — %s\033[0m ===\n", pattern, interval))

	for _, signal := range signals {
		trendIcon := "⬆️"
		colorCode := "\033[92m" // Green
		if signal.Trend == "bearish" {
			trendIcon = "⬇️"
			colorCode = "\033[91m" // Red
		}

		line := fmt.Sprintf("%s[%s] %s %-7s\033[0m | RSI: %.2f | EMA20: %.2f | Vol: %.1f",
			colorCode, signal.Symbol, trendIcon, strings.ToUpper(signal.Trend), signal.RSI, signal.EMA, signal.Volume)

		builder.WriteString(line)
		builder.WriteString("\n")
	}
	builder.WriteString("==========================\n")

	return builder.String()
}
