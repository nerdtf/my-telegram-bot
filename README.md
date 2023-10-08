
## my-telegram-bot

**Description:**  
my-telegram-bot serves as a comprehensive Telegram interface for users to interact with an eCommerce platform. With a blend of intuitive keyboard interfaces and bot interactions, users can seamlessly register, browse an expansive product catalog, refine their search to find specific products, manage their shopping cart, and finalize orders for delivery. Additionally, the bot offers account management features and a detailed order history view, ensuring users can review past interactions and maintain their account settings with ease.

### Setup Instructions

1. Initialize the Go module for the project:
```
go mod init my-telegram-bot
```

2. Fill in your Telegram bot token in `main.go`:
Replace `YOUR_TELEGRAM_BOT_TOKEN` with your actual Telegram bot token.

3. Install the required dependencies:
```
go mod tidy
```

### Dependencies

This project uses the following dependencies:

- github.com/go-telegram-bot-api/telegram-bot-api v4.6.4+incompatible
- github.com/technoweenie/multipartstreamer v1.0.1 (indirect)

### Usage Instructions

#### Keyboards

The bot utilizes various keyboards to enhance user interaction. Below is a brief overview of the provided keyboards:

**Getting Started**
Initiate your interaction with the bot by sending the /start command. This will introduce you to the main functionalities and guide you through the initial setup process.

**Making an Order**
You can easily browse an extensive product catalog, refining your search to pinpoint specific items. Once you've made your selections, simply add them to your shopping cart and proceed to place your order.

**Managing Your Account**
Keep track of your personal details, including your shipping address, email, and any profile images you've uploaded. This ensures that your orders are processed smoothly and delivered to the correct location.

**Reviewing Order History**
For a comprehensive overview of your past transactions, access the order history. This provides a detailed record of all your purchases, helping you keep track of past interactions and expenditures.

**Completing an Order**
Once you're satisfied with the items in your shopping cart, you can finalize your order. Ensure all details are accurate before confirming, as this will initiate the delivery process.

**Miscellaneous Interactions**
Apart from the core functionalities, the bot is equipped to understand and assist you with a variety of other requests. If at any point you're unsure of what to do next, simply send a message, and the bot will guide you through the available options or assist with your query.
