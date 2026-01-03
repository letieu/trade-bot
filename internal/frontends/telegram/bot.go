package telegram

import (
	"fmt"
	"sort"
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

	// Sort signals: Bullish (green) first, then Bearish (red)
	sort.Slice(signals, func(i, j int) bool {
		isIBullish := signals[i].Trend != "bearish"
		isJBullish := signals[j].Trend != "bearish"

		if isIBullish != isJBullish {
			return isIBullish
		}
		return signals[i].Symbol < signals[j].Symbol
	})

	// Split into chunks to avoid Telegram message limit (4096 chars)
	// A safe batch size is around 50 signals per message
	batchSize := 50
	totalChunks := (len(signals) + batchSize - 1) / batchSize

	for i := 0; i < len(signals); i += batchSize {
		end := i + batchSize
		if end > len(signals) {
			end = len(signals)
		}

		currentChunk := (i / batchSize) + 1
		batch := signals[i:end]
		message := b.formatSignalsMessage(batch, currentChunk, totalChunks)
		if err := b.SendMessage(message); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bot) SendMessage(message string) error {
	chatID, err := strconv.ParseInt(b.config.ChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID format: %w", err)
	}
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true

	_, sendErr := b.bot.Send(msg)
	if sendErr != nil {
		return fmt.Errorf("failed to send telegram message: %w", sendErr)
	}
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}

	return nil
}

func (b *Bot) formatSignalsMessage(signals []types.Signal, currentChunk, totalChunks int) string {
	if len(signals) == 0 {
		return ""
	}

	var builder strings.Builder

	pattern := signals[0].Pattern
	interval := signals[0].Interval
	runTime := signals[0].Timestamp.Format("2006-01-02 15:04:05")

	// Header
	if currentChunk == 1 {
		chunkInfo := ""
		if totalChunks > 1 {
			chunkInfo = fmt.Sprintf(" [%d/%d]", currentChunk, totalChunks)
		}
		builder.WriteString(fmt.Sprintf("📊 <b>%s</b> (%s)%s\n", pattern, interval, chunkInfo))
		builder.WriteString(fmt.Sprintf("🕒 <code>%s</code>\n\n", runTime))
	} else {
		// Continuation header
		builder.WriteString(fmt.Sprintf("... [%d/%d]\n\n", currentChunk, totalChunks))
	}

	for _, signal := range signals {
		icon := "🟢"
		if signal.Trend == "bearish" {
			icon = "🔴"
		}

		// Format: 🟢 <a href="..."><b>BTCUSDT</b></a>
		url := fmt.Sprintf("https://www.bybitglobal.com/trade/usdt/%s", signal.Symbol)
		line := fmt.Sprintf("%s <a href=\"%s\"><b>%s</b></a>\n", icon, url, signal.Symbol)

		builder.WriteString(line)
	}

	// Footer for non-last chunks
	if currentChunk < totalChunks {
		builder.WriteString("\n...")
	}

	return builder.String()
}
