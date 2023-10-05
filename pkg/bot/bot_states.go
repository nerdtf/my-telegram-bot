package bot

import (
	"my-telegram-bot/pkg/api"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// SetUserState sets a new UserState for the given chatID.
func (b *Bot) SetUserState(chatID int64, state *UserState) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.states[chatID] = state
}

// GetUserState retrieves the UserState for the given chatID.
func (b *Bot) GetUserState(chatID int64) *UserState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.states[chatID]
}

// DeleteUserState deletes the UserState for the given chatID.
func (b *Bot) DeleteUserState(chatID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.states, chatID)
}

// setAddress sets the address field in the given UserState.
func setAddress(state *UserState, value string) {
	state.Data.Address = value
}

// getAddress retrieves the address field from the given UserState.
func getAddress(state *UserState) string {
	return state.Data.Address
}

// setEmail sets the email field in the given UserState.
func setEmail(state *UserState, value string) {
	state.Data.Email = value
}

// getEmail retrieves the email field from the given UserState.
func getEmail(state *UserState) string {
	return state.Data.Email
}

// getCurrentStep retrieves the current step field from the given UserState.
func getCurrentStep(state *UserState) string {
	return state.CurrentStep
}

// setCurrentStep sets the current step field in the given UserState.
func setCurrentStep(state *UserState, value string) {
	state.CurrentStep = value
}

// setImage sets the image data field in the given UserState.
func setImage(state *UserState, value []byte) {
	state.Data.ImageData = value
}

// getImage retrieves the image data field from the given UserState.
func getImage(state *UserState) []byte {
	return state.Data.ImageData
}

// setDataForState sets a specified data field for the state of a given chatID using a handler function.
func (b *Bot) setDataForState(chatID int64, handler func(*UserState, string), value string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	state := b.states[chatID]
	handler(state, value)
}

// getDataFromState retrieves a specified data field from the state of a given chatID using a handler function.
func (b *Bot) getDataFromState(chatID int64, handler func(*UserState) string) string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	state := b.states[chatID]
	return handler(state)
}

// initUserState initializes a new UserState for the given chatID if it doesn't already exist.
// It uses the provided contact data to initialize the UserState.
func (b *Bot) initUserState(chatID int64, contact *tgbotapi.Contact) {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, exists := b.states[chatID]
	if !exists {
		state := &UserState{}
		if contact != nil {
			state.CurrentStep = "address"
			data := api.RegisterData{
				Phone: contact.PhoneNumber,
			}
			if contact.FirstName != "" {
				data.FirstName = contact.FirstName
			}
			if contact.LastName != "" {
				data.LastName = contact.LastName
			}
			state.Data = data
		} else {
			state.CurrentStep = ""
			state.Data = api.RegisterData{} // Initializing Data to an empty RegisterData struct
		}
		b.states[chatID] = state
	}
}

// setDataForImageState sets the image data for the state of a given chatID.
func (b *Bot) setDataForImageState(chatID int64, handler func(*UserState, []byte), value []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	state := b.states[chatID]
	handler(state, value)
}
