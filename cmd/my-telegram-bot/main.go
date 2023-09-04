package main

import (
	"log"
	"my-telegram-bot/pkg/api"

	"my-telegram-bot/pkg/auth"
	"my-telegram-bot/pkg/bot"
)

func main() {
	YOUR_TELEGRAM_BOT_TOKEN := ""

	apiClient := api.NewAPIClient("http://127.0.0.1:8000/api")
	authClient := auth.NewAuthClient()
	authClient.Tokens = make(map[int64]string)
	bot, err := bot.NewBot(YOUR_TELEGRAM_BOT_TOKEN, apiClient, authClient)

	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	bot.Run()
}
