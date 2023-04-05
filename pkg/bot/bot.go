package bot

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"my-telegram-bot/pkg/api"

	"my-telegram-bot/pkg/auth"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type UserState struct {
	CurrentStep string
	Data        api.RegisterData
}

type Bot struct {
	bot       *tgbotapi.BotAPI
	apiClient *api.APIClient
	auth      *auth.AuthClient
	states    map[int64]*UserState
}

func NewBot(token string, apiClient *api.APIClient, authClient *auth.AuthClient) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	return &Bot{
		bot:       bot,
		apiClient: apiClient,
		auth:      authClient,
		states:    make(map[int64]*UserState),
	}, nil
}

func (b *Bot) Run() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := b.bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatal(err)
	}

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			b.handleCommand(update.Message)
		} else {
			b.handleMessage(update.Message)
		}
	}
}

func (b *Bot) handleCommand(msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start":
		b.handleStart(msg.Chat.ID)
		// msg := tgbotapi.NewMessage(msg.Chat.ID, "Please choose an option:")
		// msg.ReplyMarkup = createInlineKeyboard()
		// b.bot.Send(msg)
	default:
		b.handleUnknownCommand(msg.Chat.ID)
	}
}

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	if msg.Contact != nil {
		b.handleSharedContact(msg)
		return
	}

	state, ok := b.states[msg.Chat.ID]
	if !ok {
		return
	}

	switch state.CurrentStep {
	case "address":
		b.handleAddress(msg)
	case "email":
		b.handleEmail(msg)
	case "image":
		b.handleImage(msg)
	default:
		b.replyWithMessage(msg.Chat.ID, msg.Text, nil)
	}
}

func (b *Bot) handleStart(chatID int64) {
	text := "Welcome to the My Telegram Bot! \nIf you need any help, just type /help. \nPlease, share your contact to create an account"
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = createReplyKeyboard()
	b.bot.Send(msg)
	//b.apiClient.Login(api.LoginData{Email: "mygio39@gmail.com", Password: "qwerty123"})
}

func (b *Bot) handleUnknownCommand(chatID int64) {
	b.replyWithMessage(chatID, "Sorry, I didn't understand your command. Please try again.", nil)
}

func createLocationKeyboard() tgbotapi.ReplyKeyboardMarkup {
	var replyKeyboard tgbotapi.ReplyKeyboardMarkup

	locationButton := tgbotapi.NewKeyboardButton("Share My Location")
	locationButton.RequestLocation = true

	replyKeyboard.Keyboard = append(replyKeyboard.Keyboard, []tgbotapi.KeyboardButton{locationButton})
	replyKeyboard.OneTimeKeyboard = true
	replyKeyboard.ResizeKeyboard = true

	return replyKeyboard
}

func (b *Bot) replyWithMessage(chatID int64, text string, markup interface{}) {
	msg := tgbotapi.NewMessage(chatID, text)
	if markup != nil {
		/*switch m := markup.(type) {
		case tgbotapi.ReplyKeyboardMarkup:
			msg.ReplyMarkup = m
		case tgbotapi.InlineKeyboardMarkup:
			msg.ReplyMarkup = m
		}*/
		msg.ReplyMarkup = markup
	}
	b.bot.Send(msg)
}

func (b *Bot) handleAddress(msg *tgbotapi.Message) {
	// Check if the user sent a location or a text message
	if msg.Location != nil {
		// Use the latitude and longitude to get an address
		// You may need a geocoding API to do this.
		b.states[msg.Chat.ID].Data.Address = fmt.Sprintf("Lat: %f, Long: %f", msg.Location.Latitude, msg.Location.Longitude)
	} else {
		// Use the provided address
		b.states[msg.Chat.ID].Data.Address = msg.Text
	}

	// Ask for the email
	b.states[msg.Chat.ID].CurrentStep = "email"

	b.replyWithMessage(msg.Chat.ID, "Please enter your email address:", tgbotapi.ReplyKeyboardRemove{
		RemoveKeyboard: true,
		Selective:      false,
	})
}

func (b *Bot) handleEmail(msg *tgbotapi.Message) {
	b.states[msg.Chat.ID].Data.Email = msg.Text
	// Ask for the image
	b.states[msg.Chat.ID].CurrentStep = "image"
	b.replyWithMessage(msg.Chat.ID, "Please upload your profile image (jpeg, png, jpg, gif, svg with max size 2048KB) or send 'SKIP' to skip this step:", nil)
}

func (b *Bot) handleImage(msg *tgbotapi.Message) {
	if msg.Photo != nil {
		// Use the provided image
		photoSize := (*msg.Photo)[len(*msg.Photo)-1]
		fileID := photoSize.FileID
		imageData, err := b.downloadImage(fileID)
		if err != nil {
			// Handle the error
			b.replyWithMessage(msg.Chat.ID, "Failed to download the image. Please try again.", nil)
			return
		}
		b.states[msg.Chat.ID].Data.ImageData = imageData
	} else if strings.ToLower(msg.Text) == "skip" {
		// Skip the image upload
		b.states[msg.Chat.ID].Data.ImageData = []byte{0}
	} else {
		// Invalid input, ask for image again
		b.replyWithMessage(msg.Chat.ID, "Invalid input. Please upload your profile image (jpeg, png, jpg, gif, svg with max size 2048KB) or send 'SKIP'", nil)
		return
	}

	// Call the Register function with the collected data
	registerData := b.states[msg.Chat.ID].Data
	validationErr, err := b.apiClient.Register(registerData, b.auth)
	if err == nil {
		if validationErr != nil {
			// Registration failed due to validation error
			errorMessage := "While registration some errors occurred:\n"
			for field, fieldErrors := range validationErr.Errors {
				for _, fieldError := range fieldErrors {
					errorMessage += fmt.Sprintf("- %s: %s\n", field, fieldError)
				}
			}
			errorMessage += "\nPlease start registration over."
			b.replyWithMessage(msg.Chat.ID, errorMessage, nil)
			// Restart the registration process
			b.handleStart(msg.Chat.ID)
		} else {
			// Registration succeeded
			b.replyWithMessage(msg.Chat.ID, "Registration successful! You can now use the bot.", nil)
		}
	} else {
		// Registration failed
		b.replyWithMessage(msg.Chat.ID, fmt.Sprintf("Registration failed: %v", err), nil)
	}

	// Clear the state for this chat
	delete(b.states, msg.Chat.ID)
}

func (b *Bot) downloadImage(fileID string) ([]byte, error) {
	file, err := b.bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(file.Link(b.bot.Token))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

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

func (b *Bot) handleSharedContact(msg *tgbotapi.Message) {
	contact := msg.Contact
	// Initialize the state for this chat
	b.states[msg.Chat.ID] = &UserState{
		CurrentStep: "address",
		Data: api.RegisterData{
			LastName:  contact.LastName,
			FirstName: contact.FirstName,
			Phone:     contact.PhoneNumber,
		},
	}

	// Ask for the address
	b.replyWithMessage(msg.Chat.ID, "Please share your address or send your current location:", createLocationKeyboard())
}

func createReplyKeyboard() tgbotapi.ReplyKeyboardMarkup {
	var replyKeyboard tgbotapi.ReplyKeyboardMarkup

	shareContactButton := tgbotapi.NewKeyboardButton("Share My Contact")
	shareContactButton.RequestContact = true

	replyKeyboard.Keyboard = append(replyKeyboard.Keyboard, []tgbotapi.KeyboardButton{shareContactButton})
	replyKeyboard.OneTimeKeyboard = true
	replyKeyboard.ResizeKeyboard = true

	return replyKeyboard
}
