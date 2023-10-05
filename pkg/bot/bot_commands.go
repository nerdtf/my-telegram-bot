package bot

import (
	"fmt"
	"html"
	"log"
	"my-telegram-bot/pkg/api"
	"net/url"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// handleStart sends a welcome message to the user when the bot is started.
func (b *Bot) handleStart(chatID int64) {
	text := "Welcome to the My Telegram Bot! \nIf you need any help, just type /help. \nPlease, share your contact to create an account"
	b.sendTextMessageWithReplyMarkup(chatID, text, createReplyKeyboard())
}

// handleAddress processes the address shared by the user.
// If the user sends a location, it uses the latitude and longitude of the location. Otherwise, it uses the text message as the address.
func (b *Bot) handleAddress(msg *tgbotapi.Message) {
	// Check if the user sent a location or a text message
	if msg.Location != nil {
		// Use the latitude and longitude to get an address
		b.setDataForState(msg.Chat.ID, setAddress, fmt.Sprintf("Lat: %f, Long: %f", msg.Location.Latitude, msg.Location.Longitude))
	} else {
		// Use the provided address
		b.setDataForState(msg.Chat.ID, setAddress, msg.Text)
	}

	// Ask for the email
	b.setDataForState(msg.Chat.ID, setCurrentStep, "email")

	b.replyWithMessage(msg.Chat.ID, "Please enter your email address:", tgbotapi.ReplyKeyboardRemove{
		RemoveKeyboard: true,
		Selective:      false,
	})
}

// handleEmail processes the email shared by the user.
func (b *Bot) handleEmail(msg *tgbotapi.Message) {
	b.setDataForState(msg.Chat.ID, setEmail, msg.Text)
	// Ask for the image
	b.setDataForState(msg.Chat.ID, setCurrentStep, "image")
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
	b.setDataForImageState(msg.Chat.ID, setImage, imageData)
}
func (b *Bot) handleSkipImage(msg *tgbotapi.Message) {
	b.setDataForImageState(msg.Chat.ID, setImage, nil)
}

func (b *Bot) handleInvalidImageInput(msg *tgbotapi.Message) {
	b.replyWithMessage(msg.Chat.ID, "Invalid input. Please upload your profile image (jpeg, png, jpg, gif, svg with max size 2048KB) or send 'SKIP'", nil)
}

func (b *Bot) handleRegistration(msg *tgbotapi.Message) {
	// Call the Register function with the collected data
	registerData := b.GetUserState(msg.Chat.ID).Data
	validationErr, err := b.apiClient.Register(registerData, msg.Chat.ID, b.auth)
	if err == nil {
		b.handleRegistrationSuccess(msg, registerData)
	} else {
		b.handleRegistrationFailure(msg, err, validationErr)
	}

	// Clear the state for this chat
	b.DeleteUserState(msg.Chat.ID)
}

// handleSharedContact processes the contact shared by the user.
// It initializes the state for the chat and asks for the address of the user.
func (b *Bot) handleSharedContact(msg *tgbotapi.Message) {
	contact := msg.Contact
	// Initialize the state for this chat
	b.initUserState(msg.Chat.ID, contact)

	// Ask for the address
	b.replyWithMessage(msg.Chat.ID, "Please share your address or send your current location:", createLocationKeyboard())
}

// handleMakeOrder displays the list of products for ordering.
// It also sends an inline keyboard with paging and search button.
func (b *Bot) handleMakeOrder(chatID int64, page int, search string) {
	// Call the API to retrieve the list of products
	products, hasNextPage, err := b.apiClient.GetProducts(perPage, page, b.auth, chatID, search)
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
			b.sendImage(chatID, product.Image, "product")
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
	var menu tgbotapi.InlineKeyboardMarkup
	var menuText string
	if search != "" {
		menu = createPaginationKeyboard(page, hasNextPage, search)
		menuText = fmt.Sprintf("Results for '%s'. Use the buttons below to navigate between pages:", search)
	} else {
		menu = createPaginationKeyboard(page, hasNextPage, "")
		menuText = "Use the buttons below to navigate between pages or search for a specific product:"
	}
	b.sendTextMessageWithReplyMarkup(chatID, menuText, menu)
}

func (b *Bot) handleSearchInit(chatID int64) {
	b.initUserState(chatID, nil) // Assuming initUserState can handle nil contact
	b.setDataForState(chatID, setCurrentStep, "search")
	b.replyWithMessage(chatID, "Please enter a product name to search for:", nil)
}

func (b *Bot) handleSearch(chatID int64, page int, searchQuery string) {
	// If searchQuery is empty, prompt the user to enter a search query
	if searchQuery == "" {
		b.replyWithMessage(chatID, "Please enter a product name to search for:", nil)
		return
	}
	b.DeleteUserState(chatID)
	// Call the refactored handleMakeOrder with the search query
	b.handleMakeOrder(chatID, page, searchQuery)
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
		b.handlePreviousPage(data, chatID)
	case strings.HasPrefix(data, "next_page_"):
		b.handleNextPage(data, chatID)
	case data == "search":
		b.handleSearchInit(chatID)
	case data == "upload_avatar" || data == "edit_image":
		b.setUserEditingState(chatID, "image")
		b.replyWithMessage(chatID, "Please upload your new profile image.", nil)
	case strings.HasPrefix(data, "edit_"):
		field := strings.TrimPrefix(data, "edit_")
		b.setUserEditingState(chatID, field)
		b.replyWithMessage(chatID, fmt.Sprintf("Please enter the new value for your %s:", field), nil)
	case callbackQuery.Data == "retry_update":
		field := b.getUserEditingState(chatID)
		if field != "" {
			b.replyWithMessage(chatID, fmt.Sprintf("Please enter the new value for your %s:", field), nil)
		} else {
			b.replyWithMessage(chatID, "Please start the update process again.", nil)
		}
	case callbackQuery.Data == "cancel_update":
		b.clearUserEditingState(chatID)
		b.replyWithMessage(chatID, "Account update process canceled.", nil)

	case data == "back":
		b.sendMenu(chatID)
	case data == "modify_cart":
		b.handleMakeOrder(chatID, 1, "")
	case data == "complete_order":
		b.handleCompleteOrder(chatID, false)
	case data == "cart":
		b.handleCartAction(chatID)
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

func (b *Bot) handleCartAction(chatID int64) {
	err := b.InitUserCart(chatID)
	if err != nil {
		log.Printf("Error initializing user cart: %v", err)
	}
	cartItems, err := b.apiClient.GetCartItems(b.auth, true, chatID)
	if err != nil {
		b.replyWithMessage(chatID, "An error occurred while fetching your cart. Please try again.", nil)
	} else if len(cartItems) == 0 {
		b.replyWithMessage(chatID, "Your cart is empty.", nil)
		b.sendMenu(chatID)
	} else {
		b.handleUserCart(cartItems, chatID)
	}
}

func (b *Bot) handlePreviousPage(data string, chatID int64) {
	parts := strings.Split(data, "_")
	page, _ := strconv.Atoi(parts[2])
	page -= 1
	search := ""
	if len(parts) > 3 {
		search, _ = url.QueryUnescape(parts[3])
	}
	b.handleMakeOrder(chatID, page, search)
}

func (b *Bot) handleNextPage(data string, chatID int64) {
	parts := strings.Split(data, "_")
	page, _ := strconv.Atoi(parts[2])
	page += 1
	search := ""
	if len(parts) > 3 {
		search, _ = url.QueryUnescape(parts[3])
	}
	b.handleMakeOrder(chatID, page, search)
}

func (b *Bot) handleUserCart(cartItems []api.CartItem, chatID int64) {
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
	editCartButton := tgbotapi.NewInlineKeyboardButtonData("🛒 Edit the Cart", "modify_cart")
	completeOrderButton := tgbotapi.NewInlineKeyboardButtonData("🛍 Complete Order", "complete_order")
	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(editCartButton, completeOrderButton))
	b.sendTextMessageWithReplyMarkup(chatID, "Choose your action:", inlineKeyboard)
}

func (b *Bot) sendTextMessageWithReplyMarkup(chatID int64, text string, replyMarkup interface{}) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = replyMarkup
	b.bot.Send(msg)
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
	b.DeleteUserState(msg.Chat.ID)
}

func (b *Bot) handleRegistrationSuccess(msg *tgbotapi.Message, registerData api.RegisterData) {
	// Registration succeeded
	b.replyWithMessage(msg.Chat.ID, "Registration successful! You can now use the bot.", nil)
	greetingMessage := fmt.Sprintf("Hello, %s! Welcome to our bot. To get started, use the buttons we'll provide to make orders, manage your account, view your order history, and manage your cart.", registerData.FirstName)
	b.replyWithMessage(msg.Chat.ID, greetingMessage, nil)
	b.sendMenu(msg.Chat.ID)

	// Clear the state for this chat
	b.DeleteUserState(msg.Chat.ID)
}

// handleCompleteOrder processes the user's request to complete an order.
func (b *Bot) handleCompleteOrder(chatID int64, sendMenuOnFailing bool) {
	if len(b.cart[chatID]) == 0 {
		b.replyWithMessage(chatID, "Your cart is empty. Please add at least one product to the cart before placing an order.", nil)
		if sendMenuOnFailing == true {
			b.sendMenu(chatID)
		}
		return
	}
	// Call the CompleteOrder function of the APIClient to complete the order
	orderResponse, err := b.apiClient.CompleteOrder(b.auth, chatID)
	if err != nil {
		b.replyWithMessage(chatID, fmt.Sprintf("Error completing the order: %v Please try again later.", err), nil)
		b.sendMenu(chatID)
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
			"\n%s\nQuantity: %d\nPrice: %.2f\n",
			item.ProductName,
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
	b.sendMenu(chatID)
}

// handleMyAccount fetches and displays the user's account details and provides editing options.
func (b *Bot) handleMyAccount(chatID int64, accountInfoFromUpdate *api.AccountInfo) {

	var accountInfo *api.AccountInfo
	// Fetch account info
	if accountInfoFromUpdate == nil {
		var err error
		accountInfo, err = b.apiClient.GetAccountInfo(b.auth, chatID)
		if err != nil {
			b.replyWithMessage(chatID, "Error fetching account details. Please try again later.", nil)
			return
		}
	} else {
		accountInfo = accountInfoFromUpdate
	}

	if accountInfo.Data.Image != "" {
		b.sendImage(chatID, accountInfo.Data.Image, "account")
		b.sendMessageWithEditButton(chatID, "Current Account Image", "edit_image")
	} else {
		// Create and send the upload button
		uploadButton := tgbotapi.NewInlineKeyboardButtonData("Upload Image", "upload_avatar")
		b.sendTextMessageWithReplyMarkup(chatID, "You don't have an avatar image yet. Please upload one:",
			tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(uploadButton)))
	}

	// Handle other fields
	fields := DefaultAccountFields(accountInfo)
	for _, field := range fields {
		b.sendMessageWithEditButton(chatID, fmt.Sprintf("*%s* ➤ `%s`", field.Name, field.Value), field.EditData)
	}

	daysMessage := fmt.Sprintf("You are our favorite customer already for *%d* days! 🎉🥳", accountInfo.Data.DaysSinceCreation)
	msg := tgbotapi.NewMessage(chatID, daysMessage)
	msg.ParseMode = "Markdown"
	b.bot.Send(msg)
	b.sendMenu(chatID)
}

func (b *Bot) sendMessageWithEditButton(chatID int64, msgText, editData string) {
	editButton := tgbotapi.NewInlineKeyboardButtonData("✏️ Edit", editData)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(editButton))
	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "Markdown"
	b.bot.Send(msg)
}

func (b *Bot) handleOrderHistory(chatID int64) {
	orderHistory, err := b.apiClient.GetOrderHistory(b.auth, chatID)
	if err != nil {
		b.replyWithMessage(chatID, "Error fetching order history. Please try again later.", nil)
		b.sendMenu(chatID)
		return
	}

	if len(orderHistory.Data) == 0 {
		b.replyWithMessage(chatID, "You have no orders yet. Start shopping to see your orders here! 🛍️", nil)
		b.sendMenu(chatID)
		return
	}

	reversedOrders := reverseOrderArray(orderHistory.Data)

	for i, order := range reversedOrders {
		orderNumber := len(reversedOrders) - i
		orderStatus := strings.ToUpper(order.Status)
		orderTotalPrice := fmt.Sprintf("%.2f💲", order.TotalPrice)

		timestamp, err := time.Parse(time.RFC3339Nano, order.OrderItems[0].CreatedAt)
		if err != nil {
			fmt.Println("Error parsing timestamp:", err)
			continue
		}
		orderCreatedAt := timestamp.Format("Monday, 02 January 2006")

		var orderItemsStrings []string
		for _, item := range order.OrderItems {
			orderItemsStrings = append(orderItemsStrings, fmt.Sprintf("%d x %s", item.Quantity, item.ProductName))
		}
		orderItemsList := strings.Join(orderItemsStrings, "\n")

		messageText := fmt.Sprintf(
			"*Order #%d* 📦\n"+
				"*Status:* `%s`\n"+
				"*Total Price: *%s\n"+
				"*Date:* %s\n"+
				"*Items:*\n%s",
			orderNumber, orderStatus, orderTotalPrice, orderCreatedAt, orderItemsList,
		)
		msg := tgbotapi.NewMessage(chatID, messageText)
		msg.ParseMode = "Markdown"
		b.bot.Send(msg)
	}
	b.sendMenu(chatID)
}

func reverseOrderArray(orders []api.OrderResponseItem) []api.OrderResponseItem {
	// Create a new slice to hold the reversed order
	reversed := make([]api.OrderResponseItem, len(orders))

	// Iterate through the original slice in reverse order and copy each element to the new slice
	for i, j := len(orders)-1, 0; i >= 0; i, j = i-1, j+1 {
		reversed[j] = orders[i]
	}

	return reversed
}
