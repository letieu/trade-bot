package telegram

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/letieu/trade-bot/internal/config"
	"github.com/letieu/trade-bot/internal/types"
)

type Bot struct {
	config *config.TelegramConfig
	bot    *tgbotapi.BotAPI
}

func NewBot(cfg *config.TelegramConfig) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	bot.Debug = false

	return &Bot{
		config: cfg,
		bot:    bot,
	}, nil
}

func (b *Bot) SendSignals(signals []types.Signal) error {
	if len(signals) == 0 {
		return nil
	}

	message := b.formatSignalsMessage(signals)
	return b.SendMessage(message)
}

func (b *Bot) SendMessage(message string) error {
	chatID, err := strconv.ParseInt(b.config.ChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID format: %w", err)
	}
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"

	_, sendErr := b.bot.Send(msg)
	if sendErr != nil {
		return fmt.Errorf("failed to send telegram message: %w", sendErr)
	}
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}

	return nil
}

func (b *Bot) formatSignalsMessage(signals []types.Signal) string {
	if len(signals) == 0 {
		return "No trading signals found"
	}

	var builder strings.Builder

	pattern := signals[0].Pattern
	interval := signals[0].Interval

	builder.WriteString(fmt.Sprintf("%s — %s\n\n", pattern, interval))

	for _, signal := range signals {
		trendIcon := "⬆️"
		if signal.Trend == "bearish" {
			trendIcon = "⬇️"
		}

		line := fmt.Sprintf("*%s* %s ‣ RSI: %.2f ‣ EMA20: %.2f ‣ Vol: %.1f",
			signal.Symbol, trendIcon, signal.RSI, signal.EMA, signal.Volume)

		builder.WriteString(line)
		builder.WriteString("\n")
	}

	return builder.String()
}
