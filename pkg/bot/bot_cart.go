package bot

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	// Menu message
	menuMessage = "Please choose an option:"
)

// sendMenu sends a menu to a chat identified by chatID
func (b *Bot) sendMenu(chatID int64) {
	menu := createMenuKeyboard()

	msg := tgbotapi.NewMessage(chatID, menuMessage)
	msg.ReplyMarkup = menu
	_, err := b.bot.Send(msg)
	if err != nil {
		log.Printf("sendMenu failed: %v", err)
	}
}

// InitUserCart initializes the user's cart when they start a session with the bot.
func (b *Bot) InitUserCart(chatID int64) error {

	// If this user already has an entry in the cart map, return early to avoid re-initialization
	if _, ok := b.cart[chatID]; ok {

		return nil
	}

	// Retrieve the current state of the cart using the API client's GetCartItems method
	cartItems, err := b.apiClient.GetCartItems(b.auth, false, chatID)
	if err != nil {
		return fmt.Errorf("Error retrieving cart items: %w", err)
	}

	// Initialize an empty cart for this user
	userCart := make(map[int]BotCartItem)

	// Populate the user's cart with the retrieved cart items
	for _, item := range cartItems {
		userCart[item.ProductID] = BotCartItem{Quantity: item.Quantity, MessageID: 0}
	}

	// Add this user's cart to the bot's cart map
	b.cart[chatID] = userCart

	return nil
}

// editCartMessage edits a cart message identified by chatID and messageID, and updates the product identified by productID
func (b *Bot) editCartMessage(chatID int64, messageID int, productID int) error {

	if !b.isMostRecentMessage(chatID, messageID, productID) {
		return fmt.Errorf("Warning: You're trying to update the cart from an older message. Please scroll to the most recent message to make changes to your cart.")
	}

	// Build keyboard
	keyboard := b.buildCartKeyboard(chatID, productID)

	// Send edit message
	edit := tgbotapi.EditMessageReplyMarkupConfig{
		BaseEdit: tgbotapi.BaseEdit{
			ChatID:      chatID,
			MessageID:   messageID,
			ReplyMarkup: &keyboard,
		},
	}

	_, err := b.bot.Send(edit)
	if err != nil {
		return fmt.Errorf("failed to edit cart message: %v", err)
	}

	return nil
}

// buildCartKeyboard makes an inline keybord with buttons to add or remove products from cart
func (b *Bot) buildCartKeyboard(chatID int64, productID int) tgbotapi.InlineKeyboardMarkup {

	cartItem := b.cart[chatID][productID]

	// Build buttons
	buttons := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("➕", fmt.Sprintf("add_to_cart_%d", productID)),
		tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🛒 %d", b.cart[chatID][productID].Quantity), "cart"),
		tgbotapi.NewInlineKeyboardButtonData("➖", fmt.Sprintf("reduce_amount_in_cart_%d", productID)),
	}

	if cartItem.Quantity > 0 {
		removeButton := tgbotapi.NewInlineKeyboardButtonData("❌", fmt.Sprintf("remove_from_cart_%d", productID))
		buttons = append(buttons, removeButton)
	}

	// Build keyboard
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(buttons...),
	)
}

// isMostRecentMessage checks if the provided messageID is the most recent cart-related message for the given chatID.
// Returns true if it is the most recent message, false otherwise.
func (b *Bot) isMostRecentMessage(chatID int64, messageID int, productID int) bool {
	cartItem, exists := b.cart[chatID][productID]
	if !exists {
		return true
	}
	return cartItem.MessageID == messageID
}
