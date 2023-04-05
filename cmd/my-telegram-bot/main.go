package main

import (
	"log"
	"my-telegram-bot/pkg/api"

	"my-telegram-bot/pkg/auth"
	"my-telegram-bot/pkg/bot"
)

func main() {
	YOUR_TELEGRAM_BOT_TOKEN := "1015617928:AAFVLXthgF6p27dk8ieh5Wo2lteLJhP0po4"

	apiClient := api.NewAPIClient("http://127.0.0.1:8000/api")
	authClient := auth.NewAuthClient()
	bot, err := bot.NewBot(YOUR_TELEGRAM_BOT_TOKEN, apiClient, authClient)

	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	bot.Run()
}
