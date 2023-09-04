package bot

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// newKeyboard creates a new instance of ReplyKeyboardMarkup with default properties
func newKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.ReplyKeyboardMarkup{
		OneTimeKeyboard: true,
		ResizeKeyboard:  true,
	}
}

// createLocationKeyboard creates a keyboard with a button to share location
func createLocationKeyboard() tgbotapi.ReplyKeyboardMarkup {
	keyboard := newKeyboard()

	locationButton := tgbotapi.NewKeyboardButton("Share My Location")
	locationButton.RequestLocation = true

	keyboard.Keyboard = append(keyboard.Keyboard, []tgbotapi.KeyboardButton{locationButton})

	return keyboard
}

// createInlineKeyboard creates an inline keyboard with different options
func createInlineKeyboard() tgbotapi.InlineKeyboardMarkup {
	var inlineKeyboard tgbotapi.InlineKeyboardMarkup

	row1 := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("Option 1", "option1"),
		tgbotapi.NewInlineKeyboardButtonData("Option 2", "option2"),
	}

	row2 := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("Option 3", "option3"),
	}

	inlineKeyboard.InlineKeyboard = append(inlineKeyboard.InlineKeyboard, row1, row2)

	return inlineKeyboard
}

// createReplyKeyboard creates a keyboard with a button to share contact
func createReplyKeyboard() tgbotapi.ReplyKeyboardMarkup {
	keyboard := newKeyboard()

	shareContactButton := tgbotapi.NewKeyboardButton("Share My Contact")
	shareContactButton.RequestContact = true

	keyboard.Keyboard = append(keyboard.Keyboard, []tgbotapi.KeyboardButton{shareContactButton})

	return keyboard
}

// createMenuKeyboard creates a keyboard with different menu options
func createMenuKeyboard() tgbotapi.ReplyKeyboardMarkup {
	keyboard := newKeyboard()

	keyboard.Keyboard = append(keyboard.Keyboard,
		[]tgbotapi.KeyboardButton{
			tgbotapi.NewKeyboardButton("Make Order 🛍️"),
			tgbotapi.NewKeyboardButton("My Account 📋"),
			tgbotapi.NewKeyboardButton("Complete Order 📦"),
		},
		[]tgbotapi.KeyboardButton{
			tgbotapi.NewKeyboardButton("Order's History 📖"),
			tgbotapi.NewKeyboardButton("Cart 🛒"),
		},
	)

	return keyboard
}

// createPaginationKeyboard creates a keyboard with "Previous", "Next", and "Complete Order" buttons.
func createPaginationKeyboard(page int, hasNextPage bool) tgbotapi.InlineKeyboardMarkup {

	// Create "Previous Page" and "Next Page" buttons
	prevPageData := "disabled"
	nextPageData := "disabled"
	if page > 1 {
		prevPageData = fmt.Sprintf("previous_page_%d", page)
	}
	if hasNextPage {
		nextPageData = fmt.Sprintf("next_page_%d", page)
	}

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Search 🔍", "search"),
			tgbotapi.NewInlineKeyboardButtonData("Back ⬅️", "back"),
			tgbotapi.NewInlineKeyboardButtonData("Cart 🛒", "cart"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Previous Page", prevPageData),
			tgbotapi.NewInlineKeyboardButtonData("Complete Order 📦", "complete_order"),
			tgbotapi.NewInlineKeyboardButtonData("Next Page", nextPageData),
		),
	)
}
