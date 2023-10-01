package bot

import (
	"fmt"
	"log"
	"my-telegram-bot/pkg/api"
	"my-telegram-bot/pkg/auth"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const perPage = 5

// UserState stores the current step and data of a user during a register operation
type UserState struct {
	CurrentStep string
	Data        api.RegisterData
}

// Bot contains the Telegram Bot API, API client, authentication client, user states, and cart
type Bot struct {
	mu                sync.RWMutex
	bot               *tgbotapi.BotAPI
	apiClient         *api.APIClient
	auth              *auth.AuthClient
	states            map[int64]*UserState
	cart              map[int64]map[int]BotCartItem
	userEditingStates map[int64]string
}

type BotCartItem struct {
	Quantity  int
	MessageID int
}

// NewBot initializes a new Bot instance
func NewBot(token string, apiClient *api.APIClient, authClient *auth.AuthClient) (*Bot, error) {
	// Create new Telegram Bot API instance
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create new bot: %w", err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	b := &Bot{
		bot:               bot,
		apiClient:         apiClient,
		auth:              authClient,
		states:            make(map[int64]*UserState),
		cart:              make(map[int64]map[int]BotCartItem),
		userEditingStates: make(map[int64]string),
	}

	return b, nil
}

// Run starts the Bot instance and listens for updates
func (b *Bot) Run() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := b.bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatalf("failed to get updates channel: %v", err)
	}

	for update := range updates {
		if update.Message == nil && update.CallbackQuery == nil {
			continue
		}

		if update.CallbackQuery != nil {
			b.handleCallbackQuery(update.CallbackQuery)
			continue
		}

		if update.Message.IsCommand() {
			b.handleCommand(update.Message)
		} else {
			b.handleMessage(update.Message)
		}
	}
}
