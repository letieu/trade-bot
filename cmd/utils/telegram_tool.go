package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/letieu/trade-bot/internal/config"
)

func main() {
	action := flag.String("action", "get-id", "Action to perform: 'get-id' or 'test-msg'")
	configFile := flag.String("config", "trade-bot.yaml", "Path to config file")
	flag.Parse()

	// Load config
	cfg := config.Load(*configFile)
	if cfg.Telegram.BotToken == "" {
		log.Fatal("Bot token not found in config file")
	}

	bot, err := tgbotapi.NewBotAPI(cfg.Telegram.BotToken)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	if *action == "test-msg" {
		sendTestMessage(bot, cfg.Telegram.ChatID)
	} else {
		getChatID(bot)
	}
}

func sendTestMessage(bot *tgbotapi.BotAPI, chatIDStr string) {
	if chatIDStr == "" {
		log.Fatal("Chat ID is empty in config file")
	}

	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		log.Fatalf("Invalid Chat ID format: %v", err)
	}

	msg := tgbotapi.NewMessage(chatID, "✅ **Test Message**\n\nYour Telegram bot is configured correctly!")
	msg.ParseMode = "Markdown"

	_, err = bot.Send(msg)
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	fmt.Println("Successfully sent test message!")
}

func getChatID(bot *tgbotapi.BotAPI) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	fmt.Println("Waiting for messages... Please send a message to your bot on Telegram.")

	for update := range updates {
		if update.Message != nil { // If we got a message
			fmt.Printf("[%s] %s\n", update.Message.From.UserName, update.Message.Text)
			fmt.Printf("Chat ID: %d\n", update.Message.Chat.ID)
			fmt.Println("------------------------------")
			fmt.Println("Copy this Chat ID and put it in your trade-bot.yaml file.")
			return
		}
	}
}
