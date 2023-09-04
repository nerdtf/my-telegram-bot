package bot

import (
	"fmt"
	"html"
	"log"
	"my-telegram-bot/pkg/api"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// handleStart sends a welcome message to the user when the bot is started.
func (b *Bot) handleStart(chatID int64) {
	text := "Welcome to the My Telegram Bot! \nIf you need any help, just type /help. \nPlease, share your contact to create an account"
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = createReplyKeyboard()
	b.bot.Send(msg)
}

// handleAddress processes the address shared by the user.
// If the user sends a location, it uses the latitude and longitude of the location. Otherwise, it uses the text message as the address.
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

// handleEmail processes the email shared by the user.
func (b *Bot) handleEmail(msg *tgbotapi.Message) {
	b.states[msg.Chat.ID].Data.Email = msg.Text
	// Ask for the image
	b.states[msg.Chat.ID].CurrentStep = "image"
	b.replyWithMessage(msg.Chat.ID, "Please upload your profile image (jpeg, png, jpg, gif, svg with max size 2048KB) or send 'SKIP' to skip this step:", nil)
}

// handleImage processes the profile image shared by the user.
func (b *Bot) handleImage(msg *tgbotapi.Message) {
	if msg.Photo != nil {
		b.handlePhotoImage(msg)
	} else if strings.ToLower(msg.Text) == "skip" {
		b.handleSkipImage(msg)
	} else {
		b.handleInvalidImageInput(msg)
	}

	b.handleRegistration(msg)
}
func (b *Bot) handlePhotoImage(msg *tgbotapi.Message) {
	photoSize := (*msg.Photo)[len(*msg.Photo)-1]
	fileID := photoSize.FileID
	imageData, err := b.downloadImage(fileID)
	if err != nil {
		b.replyWithMessage(msg.Chat.ID, fmt.Sprintf("Failed to download the image: %v. Please try again.", err), nil)
		return
	}
	b.states[msg.Chat.ID].Data.ImageData = imageData
}
func (b *Bot) handleSkipImage(msg *tgbotapi.Message) {
	b.states[msg.Chat.ID].Data.ImageData = nil
}

func (b *Bot) handleInvalidImageInput(msg *tgbotapi.Message) {
	b.replyWithMessage(msg.Chat.ID, "Invalid input. Please upload your profile image (jpeg, png, jpg, gif, svg with max size 2048KB) or send 'SKIP'", nil)
}

func (b *Bot) handleRegistration(msg *tgbotapi.Message) {
	// Call the Register function with the collected data
	registerData := b.states[msg.Chat.ID].Data
	validationErr, err := b.apiClient.Register(registerData, msg.Chat.ID, b.auth)
	if err == nil {
		b.handleRegistrationSuccess(msg, registerData)
	} else {
		b.handleRegistrationFailure(msg, err, validationErr)
	}

	// Clear the state for this chat
	delete(b.states, msg.Chat.ID)
}

// handleSharedContact processes the contact shared by the user.
// It initializes the state for the chat and asks for the address of the user.
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

// handleMakeOrder displays the list of products for ordering.
// It also sends an inline keyboard with paging and search button.
func (b *Bot) handleMakeOrder(chatID int64, page int) {
	// Call the API to retrieve the list of products
	products, hasNextPage, err := b.apiClient.GetProducts(perPage, page, b.auth, chatID)
	if err != nil {
		b.replyWithMessage(chatID, fmt.Sprintf("An error occurred while fetching products: %v. Please try again later.", err), nil)
		return
	}

	if len(products) == 0 {
		b.replyWithMessage(chatID, "No more products available.", nil)
		return
	}
	b.InitUserCart(chatID)
	for _, product := range products {
		if product.Image != "" {
			b.sendProductImage(chatID, product.Image)
		}

		productInfo := fmt.Sprintf("<b>Name:</b> %s\n<b>Price:</b> $%.2f\n<b>Weight:</b> %d g\n<b>Description:</b> %s",
			html.EscapeString(product.Name), product.Price, product.Weight, html.EscapeString(product.Description))

		// Create inline keyboard buttons for adding and removing the product from the cart
		// Use buildCartKeyboard to generate the inline keyboard
		inlineKeyboard := b.buildCartKeyboard(chatID, product.ID)
		// Send product information text with the inline keyboard
		textMsg := tgbotapi.NewMessage(chatID, productInfo)
		textMsg.ParseMode = "HTML"
		textMsg.ReplyMarkup = inlineKeyboard
		sentMsg, err := b.bot.Send(textMsg)
		if err != nil {
			b.replyWithMessage(chatID, fmt.Sprintf("An error occurred while sending products: %v. Please try again later.", err), nil)
			return
		}
		// Update the MessageID in the cart
		cartItem := b.cart[chatID][product.ID]
		cartItem.MessageID = sentMsg.MessageID
		b.cart[chatID][product.ID] = cartItem
	}

	// Send the inline keyboard with paging and the search button
	menu := createPaginationKeyboard(page, hasNextPage)

	msg := tgbotapi.NewMessage(chatID, "Use the buttons below to navigate between pages or search for a specific product:")
	msg.ReplyMarkup = menu
	b.bot.Send(msg)
}

// handleCallbackQuery handles the callback queries from the inline keyboard buttons.
func (b *Bot) handleCallbackQuery(callbackQuery *tgbotapi.CallbackQuery) {
	data := callbackQuery.Data
	chatID := callbackQuery.Message.Chat.ID
	messageID := callbackQuery.Message.MessageID

	switch {
	case data == "disabled":
		return
	case strings.HasPrefix(data, "previous_page_"):
		page, _ := strconv.Atoi(strings.TrimPrefix(data, "previous_page_"))
		page -= 1
		b.handleMakeOrder(chatID, page)
	case strings.HasPrefix(data, "next_page_"):
		page, _ := strconv.Atoi(strings.TrimPrefix(data, "next_page_"))
		page += 1
		b.handleMakeOrder(chatID, page)
	case data == "search":
		// Implement search functionality here
	case data == "back":
		b.sendMenu(chatID)
	case data == "edit_cart":
		b.handleMakeOrder(chatID, 1)
	case data == "complete_order":
		b.handleCompleteOrder(chatID)
	case data == "cart":
		err := b.InitUserCart(chatID)
		if err != nil {
			log.Printf("Error initializing user cart: %v", err)
		}
		cartItems, err := b.apiClient.GetCartItems(b.auth, true, chatID)
		if err != nil {
			b.replyWithMessage(chatID, "An error occurred while fetching your cart. Please try again.", nil)
		} else if len(cartItems) == 0 {
			b.replyWithMessage(chatID, "Your cart is empty.", nil)
		} else {
			cartMessage := "Shopping Cart Items: \n"
			totalCost := 0.0
			for _, cartItem := range cartItems {
				itemTotalPrice := float64(cartItem.Quantity) * cartItem.Price
				totalCost += itemTotalPrice
				cartMessage += fmt.Sprintf("<b>%s:</b> %d items | $%.2f \n", html.EscapeString(cartItem.ProductName), cartItem.Quantity, itemTotalPrice)
			}
			cartMessage += fmt.Sprintf("\nTotal : $%.2f", totalCost)

			textMsg := tgbotapi.NewMessage(chatID, cartMessage)
			textMsg.ParseMode = "HTML"
			b.bot.Send(textMsg)

			// Add 'Edit Cart' and 'Complete Order' buttons
			editCartButton := tgbotapi.NewInlineKeyboardButtonData("🛒 Edit the Cart", "edit_cart")
			completeOrderButton := tgbotapi.NewInlineKeyboardButtonData("🛍 Complete Order", "complete_order")
			inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(editCartButton, completeOrderButton))

			textMsg = tgbotapi.NewMessage(chatID, "Choose your action:")
			textMsg.ReplyMarkup = inlineKeyboard
			b.bot.Send(textMsg)
		}
	case strings.HasPrefix(data, "add_to_cart_"):
		productID, _ := strconv.Atoi(strings.TrimPrefix(data, "add_to_cart_"))
		if !b.isMostRecentMessage(chatID, messageID, productID) {
			b.replyWithMessage(chatID, "🚨 Warning: 🚨 \nYou're trying to update the cart from an older message. Please scroll to the most recent message to make changes to your cart. 🛒", nil)
			return
		}
		err := b.reduceOrIncreaseAmountInCart(chatID, productID, 1, false)
		if err == nil {
			b.editCartMessage(chatID, messageID, productID)
			b.replyWithMessage(chatID, "Product added to your cart.", nil)
		}

	case strings.HasPrefix(data, "reduce_amount_in_cart_"):
		productID, _ := strconv.Atoi(strings.TrimPrefix(data, "reduce_amount_in_cart_"))
		if !b.isMostRecentMessage(chatID, messageID, productID) {
			b.replyWithMessage(chatID, "🚨 Warning: 🚨 \nYou're trying to update the cart from an older message. Please scroll to the most recent message to make changes to your cart. 🛒", nil)
			return
		}
		err := b.reduceOrIncreaseAmountInCart(chatID, productID, -1, true)
		if err == nil {
			b.editCartMessage(chatID, messageID, productID)
			b.replyWithMessage(chatID, "Quantity of product is reduced", nil)
		}
	case strings.HasPrefix(data, "remove_from_cart_"):
		productID, _ := strconv.Atoi(strings.TrimPrefix(data, "remove_from_cart_"))
		if !b.isMostRecentMessage(chatID, messageID, productID) {
			b.replyWithMessage(chatID, "🚨 Warning: 🚨 \nYou're trying to update the cart from an older message. Please scroll to the most recent message to make changes to your cart. 🛒", nil)
			return
		}
		err := b.reduceOrIncreaseAmountInCart(chatID, productID, 0, true)
		if err == nil {
			b.editCartMessage(chatID, messageID, productID)
			b.replyWithMessage(chatID, "Product is removed", nil)
		}
	default:
		b.replyWithMessage(chatID, "Sorry, I didn't understand your action. Please try again.", nil)
	}

	// Acknowledge the callback query
	b.bot.AnswerCallbackQuery(tgbotapi.NewCallback(callbackQuery.ID, ""))
}

func (b *Bot) reduceOrIncreaseAmountInCart(chatID int64, productID int, amount int, remove bool) error {

	var err error
	if remove {
		removeProduct := false
		if amount == 0 {
			removeProduct = true
		}
		err = b.apiClient.RemoveProductFromCart(productID, b.auth, chatID, removeProduct)
	} else {
		err = b.apiClient.AddProductToCart(productID, amount, b.auth, chatID)
	}

	if err != nil {
		b.replyWithMessage(chatID, "Error updating cart. Please try again.", nil)
		return err
	}

	if _, ok := b.cart[chatID]; !ok {
		b.cart[chatID] = make(map[int]BotCartItem)
	}
	cartItem := b.cart[chatID][productID]
	if (remove && amount == 0) || (remove && cartItem.Quantity == 0) {
		cartItem.Quantity = 0
	} else {
		cartItem.Quantity += amount
	}

	// Update the CartItem in the cart
	b.cart[chatID][productID] = cartItem
	return nil
}

func (b *Bot) handleRegistrationFailure(msg *tgbotapi.Message, err error, validationErr *api.ValidationError) {
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
		// Registration failed
		b.replyWithMessage(msg.Chat.ID, fmt.Sprintf("Registration failed: %v", err), nil)
	}

	// Clear the state for this chat
	delete(b.states, msg.Chat.ID)
}

func (b *Bot) handleRegistrationSuccess(msg *tgbotapi.Message, registerData api.RegisterData) {
	// Registration succeeded
	b.replyWithMessage(msg.Chat.ID, "Registration successful! You can now use the bot.", nil)
	greetingMessage := fmt.Sprintf("Hello, %s! Welcome to our bot. To get started, use the buttons we'll provide to make orders, manage your account, view your order history, and manage your cart.", registerData.FirstName)
	b.replyWithMessage(msg.Chat.ID, greetingMessage, nil)
	b.sendMenu(msg.Chat.ID)

	// Clear the state for this chat
	delete(b.states, msg.Chat.ID)
}

// handleCompleteOrder processes the user's request to complete an order.
func (b *Bot) handleCompleteOrder(chatID int64) {
	if len(b.cart[chatID]) == 0 {
		b.replyWithMessage(chatID, "Your cart is empty. Please add at least one product to the cart before placing an order.", nil)
		return
	}
	// Call the CompleteOrder function of the APIClient to complete the order
	orderResponse, err := b.apiClient.CompleteOrder(b.auth, chatID)
	if err != nil {
		b.replyWithMessage(chatID, fmt.Sprintf("Error completing the order: %v Please try again later.", err), nil)
		return
	}

	// Constructing the response message with details from CompleteOrderResponse
	responseMsg := fmt.Sprintf(
		"Order Completed!\nOrder ID: %d\nStatus: %s\nTotal Price: %.2f\n",
		orderResponse.Data.ID,
		orderResponse.Data.Status,
		orderResponse.Data.TotalPrice,
	)

	for _, item := range orderResponse.Data.OrderItems {
		responseMsg += fmt.Sprintf(
			"\nQuantity: %d\nPrice: %.2f\n",
			item.Quantity,
			item.Price,
		)
	}

	b.replyWithMessage(chatID, responseMsg, nil)

	// Reset quantities to 0 and update display
	for productID, cartItem := range b.cart[chatID] {
		cartItem.Quantity = 0                                    // Reset quantity to 0
		b.cart[chatID][productID] = cartItem                     // Update cart map
		b.editCartMessage(chatID, cartItem.MessageID, productID) // Update cart display
	}

	// Clear the cart entirely
	delete(b.cart, chatID)
}
