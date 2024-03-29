package bot

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// handleCommand handles commands received from users.
func (b *Bot) handleCommand(msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start":
		b.handleStart(msg.Chat.ID)
	default:
		b.handleUnknownCommand(msg.Chat.ID)
	}
}

// handleMessage handles messages received from users.
func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	// Handle contact sharing
	if msg.Contact != nil {
		b.handleSharedContact(msg)
		return
	}
	// Check if the user is in an editing state
	if b.getUserEditingState(msg.Chat.ID) != "" {
		b.handleEditingState(msg)
		return
	}

	// Handle text-based commands
	switch msg.Text {
	case "Make Order 🛍️":
		b.handleMakeOrder(msg.Chat.ID, 1, "")
	case "My Account 📋":
		b.handleMyAccount(msg.Chat.ID, nil)
	case "Order's History 📖":
		b.handleOrderHistory(msg.Chat.ID)
	case "Complete Order 📦":
		b.handleCompleteOrder(msg.Chat.ID, true)
	case "Cart 🛒":
		b.handleCartAction(msg.Chat.ID)
	default:
		// Handle user state-specific actions
		state := b.GetUserState(msg.Chat.ID)
		if state != nil {
			switch state.CurrentStep {
			case "address":
				b.handleAddress(msg)
			case "email":
				b.handleEmail(msg)
			case "image":
				b.handleImage(msg)
			case "search":
				b.handleMakeOrder(msg.Chat.ID, 1, msg.Text)
			default:
				b.replyWithMessage(msg.Chat.ID, msg.Text, nil)
			}
		}
	}
}

// handleUnknownCommand informs the user that their command was not understood.
func (b *Bot) handleUnknownCommand(chatID int64) {
	b.replyWithMessage(chatID, "Sorry, I didn't understand your command. Please try again.", nil)
}

// replyWithMessage sends a message to the specified chatID with optional markup.
func (b *Bot) replyWithMessage(chatID int64, text string, markup interface{}) {
	msg := tgbotapi.NewMessage(chatID, text)

	// Attach markup if provided
	if markup != nil {
		msg.ReplyMarkup = markup
	}

	// Send message and handle possible error
	if _, err := b.bot.Send(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}
