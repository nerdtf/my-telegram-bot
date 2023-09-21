package bot

import (
	"fmt"
	"my-telegram-bot/pkg/api"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type AccountField struct {
	Name     string
	Value    string
	EditData string
}

func DefaultAccountFields(accountInfo *api.AccountInfo) []AccountField {
	return []AccountField{
		{"First Name", accountInfo.Data.FirstName, "edit_first_name"},
		{"Last Name", accountInfo.Data.LastName, "edit_last_name"},
		{"Address", accountInfo.Data.Address, "edit_address"},
		{"Email", accountInfo.Data.Email, "edit_email"},
		{"Phone", accountInfo.Data.Phone, "edit_phone"},
	}
}

// setUserEditingState sets the editing state for a user.
func (b *Bot) setUserEditingState(chatID int64, field string) {
	b.userEditingStates[chatID] = field
}

// getUserEditingState retrieves the editing state for a user.
func (b *Bot) getUserEditingState(chatID int64) string {
	return b.userEditingStates[chatID]
}

// clearUserEditingState clears the editing state for a user.
func (b *Bot) clearUserEditingState(chatID int64) {
	delete(b.userEditingStates, chatID)
}

// handleEditingState processes the user's input when they're in an editing state.
func (b *Bot) handleEditingState(msg *tgbotapi.Message) {
	editingField := b.getUserEditingState(msg.Chat.ID)

	if editingField == "image" {
		if msg.Photo == nil {
			b.handleAccountUpdateFailure(msg.Chat.ID, "Invalid input. Please upload a valid profile image.", nil)
			return
		}

		imageData, err := b.downloadImageForEditing(msg)
		if err != nil {
			b.handleAccountUpdateFailure(msg.Chat.ID, fmt.Sprintf("Failed to download the image: %v", err), nil)
			return
		}

		// Update the image field
		updatedAccountInfo, err := b.apiClient.UpdateField(msg.Chat.ID, b.auth, editingField, imageData)
		if err != nil {
			var validationErr *api.ValidationError
			if apiErr, ok := err.(*api.Error); ok {
				validationErr, _ = apiErr.Details.(*api.ValidationError)
			}
			b.handleAccountUpdateFailure(msg.Chat.ID, "", validationErr)
			return
		}

		b.handleMyAccount(msg.Chat.ID, updatedAccountInfo)
		b.clearUserEditingState(msg.Chat.ID)
		return
	}

	// For other fields
	newValue := strings.TrimSpace(msg.Text)
	if newValue == "" {
		b.replyWithMessage(msg.Chat.ID, "Value cannot be empty. Please enter a valid value.", nil)
		return
	}

	updatedAccountInfo, err := b.apiClient.UpdateField(msg.Chat.ID, b.auth, editingField, newValue)
	if err != nil {
		var validationErr *api.ValidationError
		if apiErr, ok := err.(*api.Error); ok {
			validationErr, _ = apiErr.Details.(*api.ValidationError)
		}
		b.handleAccountUpdateFailure(msg.Chat.ID, "", validationErr)
		return
	}

	b.handleMyAccount(msg.Chat.ID, updatedAccountInfo)
	b.clearUserEditingState(msg.Chat.ID)
}

func (b *Bot) handleAccountUpdateFailure(chatID int64, errMsg string, validationErr *api.ValidationError) {
	var errorMessage string
	if validationErr != nil {
		// Validation errors
		errorMessage = "Some errors occurred while updating your account:\n"
		for field, fieldErrors := range validationErr.Errors {
			for _, fieldError := range fieldErrors {
				errorMessage += fmt.Sprintf("- %s: %s\n", field, fieldError)
			}
		}
	} else {
		// Generic or other types of errors
		errorMessage = errMsg
	}

	// Adding a message for retrying or canceling the operation
	errorMessage += "\nYou can either try updating again or cancel the editing process."

	// Inline buttons to retry or cancel
	tryAgainButton := tgbotapi.NewInlineKeyboardButtonData("Try Again", "retry_update")
	cancelButton := tgbotapi.NewInlineKeyboardButtonData("Cancel", "cancel_update")
	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tryAgainButton, cancelButton))

	msg := tgbotapi.NewMessage(chatID, errorMessage)
	msg.ReplyMarkup = inlineKeyboard
	b.bot.Send(msg)
}
