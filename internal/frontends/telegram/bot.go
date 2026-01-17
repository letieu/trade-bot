package telegram

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

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

	// Group signals by pattern and interval
	type groupKey struct {
		pattern  string
		interval string
	}
	groups := make(map[groupKey][]types.Signal)

	for _, signal := range signals {
		key := groupKey{pattern: signal.Pattern, interval: signal.Interval}
		groups[key] = append(groups[key], signal)
	}

	// Sort groups by pattern first, then by interval
	// This ensures messages from same strategy are sent together
	type sortableGroup struct {
		key     groupKey
		signals []types.Signal
	}
	var sortedGroups []sortableGroup
	for key, sigs := range groups {
		sortedGroups = append(sortedGroups, sortableGroup{key: key, signals: sigs})
	}

	// Sort: by pattern name first, then by interval
	sort.Slice(sortedGroups, func(i, j int) bool {
		if sortedGroups[i].key.pattern != sortedGroups[j].key.pattern {
			return sortedGroups[i].key.pattern < sortedGroups[j].key.pattern
		}
		return sortedGroups[i].key.interval < sortedGroups[j].key.interval
	})

	// Send messages for each group (potentially chunked)
	for _, group := range sortedGroups {
		if err := b.sendGroupedSignals(group.key.pattern, group.key.interval, group.signals); err != nil {
			return err
		}
	}

	return nil
}

type symbolInfo struct {
	symbol string
	count  int
}

func (b *Bot) sendGroupedSignals(pattern, interval string, signals []types.Signal) error {
	// Create a map to store symbol with its consecutive count
	var bullish []symbolInfo
	var bearish []symbolInfo

	for _, signal := range signals {
		info := symbolInfo{
			symbol: signal.Symbol,
			count:  signal.ConsecutiveCount,
		}
		if signal.Trend == "bearish" {
			bearish = append(bearish, info)
		} else {
			bullish = append(bullish, info)
		}
	}

	// Sort by count (descending), then alphabetically by symbol
	sort.Slice(bullish, func(i, j int) bool {
		if bullish[i].count != bullish[j].count {
			return bullish[i].count > bullish[j].count
		}
		return bullish[i].symbol < bullish[j].symbol
	})
	sort.Slice(bearish, func(i, j int) bool {
		if bearish[i].count != bearish[j].count {
			return bearish[i].count > bearish[j].count
		}
		return bearish[i].symbol < bearish[j].symbol
	})

	// Calculate chunks based on message length (Telegram limit is 4096 chars)
	// Estimate: each symbol ~30 chars with tags and count, header ~100 chars
	// Safe limit: ~100 symbols per message to avoid hitting 4096 limit
	maxSymbolsPerChunk := 100

	totalSymbols := len(bullish) + len(bearish)
	if totalSymbols <= maxSymbolsPerChunk {
		// Single message
		message := b.formatGroupedMessage(pattern, interval, bullish, bearish, 1, 1, signals[0].Timestamp)
		return b.SendMessage(message)
	}

	// Need to chunk - split bullish and bearish separately
	bullishChunks := chunkSymbolInfos(bullish, maxSymbolsPerChunk)
	bearishChunks := chunkSymbolInfos(bearish, maxSymbolsPerChunk)

	totalChunks := len(bullishChunks) + len(bearishChunks)
	currentChunk := 0

	// Send bullish chunks
	for _, chunk := range bullishChunks {
		currentChunk++
		message := b.formatGroupedMessage(pattern, interval, chunk, nil, currentChunk, totalChunks, signals[0].Timestamp)
		if err := b.SendMessage(message); err != nil {
			return err
		}
	}

	// Send bearish chunks
	for _, chunk := range bearishChunks {
		currentChunk++
		message := b.formatGroupedMessage(pattern, interval, nil, chunk, currentChunk, totalChunks, signals[0].Timestamp)
		if err := b.SendMessage(message); err != nil {
			return err
		}
	}

	return nil
}

func chunkSymbolInfos(infos []symbolInfo, chunkSize int) [][]symbolInfo {
	if len(infos) == 0 {
		return nil
	}

	var chunks [][]symbolInfo
	for i := 0; i < len(infos); i += chunkSize {
		end := i + chunkSize
		if end > len(infos) {
			end = len(infos)
		}
		chunks = append(chunks, infos[i:end])
	}
	return chunks
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

func (b *Bot) formatGroupedMessage(pattern, interval string, bullish, bearish []symbolInfo, currentChunk, totalChunks int, timestamp time.Time) string {
	var builder strings.Builder
	loc := time.FixedZone("UTC+7", 7*60*60)

	// Header - only show on first chunk
	if currentChunk == 1 {
		runTime := timestamp.In(loc).Format("2006-01-02 15:04:05")
		chunkInfo := ""
		if totalChunks > 1 {
			chunkInfo = fmt.Sprintf(" [%d/%d]", currentChunk, totalChunks)
		}
		builder.WriteString(fmt.Sprintf("📊 <b>%s</b> (%s)%s\n", pattern, interval, chunkInfo))
		builder.WriteString(fmt.Sprintf("🕒 <code>%s</code>\n\n", runTime))
	} else {
		// Continuation chunk - just show chunk number
		builder.WriteString(fmt.Sprintf("[%d/%d]\n\n", currentChunk, totalChunks))
	}

	// Format bullish symbols - on same line with spaces
	if len(bullish) > 0 {
		builder.WriteString("🟢 ")
		for i, info := range bullish {
			if i > 0 {
				builder.WriteString("  ")
			}
			if info.count > 0 {
				// Show count for consecutive candles pattern
				builder.WriteString(fmt.Sprintf("<code>%s</code>(%d)", info.symbol, info.count))
			} else {
				// No count for other patterns
				builder.WriteString(fmt.Sprintf("<code>%s</code>", info.symbol))
			}
		}
		builder.WriteString("\n")
	}

	// Add spacing between bullish and bearish
	if len(bullish) > 0 && len(bearish) > 0 {
		builder.WriteString("\n")
	}

	// Format bearish symbols - on same line with spaces
	if len(bearish) > 0 {
		builder.WriteString("🔴 ")
		for i, info := range bearish {
			if i > 0 {
				builder.WriteString("  ")
			}
			if info.count > 0 {
				// Show count for consecutive candles pattern
				builder.WriteString(fmt.Sprintf("<code>%s</code>(%d)", info.symbol, info.count))
			} else {
				// No count for other patterns
				builder.WriteString(fmt.Sprintf("<code>%s</code>", info.symbol))
			}
		}
		builder.WriteString("\n")
	}

	return builder.String()
}
