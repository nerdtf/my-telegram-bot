package api

import (
	"net/http"
	"time"
)

// APIClient is a struct that holds the base URL for the API and an HTTP client.

type APIClient struct {
	BaseURL string
	client  *http.Client
}

// RegisterData holds the data required for user registration.
type RegisterData struct {
	LastName  string `json:"last_name"`
	FirstName string `json:"first_name"`
	ImageData []byte `json:"-"`
	Address   string `json:"address"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
}

// RegisterResponse holds the response data for user registration.
type RegisterResponse struct {
	Data struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Address   string `json:"address"`
		Email     string `json:"email"`
		Phone     string `json:"phone"`
		Image     string `json:"image"`
		Token     string `json:"token"`
	} `json:"data"`
}

// LoginData holds the data required for user login.
type LoginData struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse holds the response data after a successful login.
type LoginResponse struct {
	Data string `json:"data"`
}

// OrderResponseItem represents a single order in the order response.
type OrderResponseItem struct {
	ID         int         `json:"id"`
	ClientID   int         `json:"client_id"`
	CourierID  int         `json:"courier_id"`
	Status     string      `json:"status"`
	TotalPrice float64     `json:"totalPrice"`
	OrderItems []OrderItem `json:"orderItems"`
}

// OrderItem represents an item in an order.
type OrderItem struct {
	ID          int     `json:"id"`
	OrderID     int     `json:"order_id"`
	ProductID   int     `json:"product_id"`
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// CompleteOrderResponse holds the response data for a completed order.
type CompleteOrderResponse struct {
	Data OrderResponseItem `json:"data"`
}

// OrderHistoryResponse represents the response data for the order history endpoint.
type OrderHistoryResponse struct {
	Data []OrderResponseItem `json:"data"`
}

// Product represents a single product in the system.
type Product struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Image       string  `json:"image"`
	Weight      int     `json:"weight"`
	InCart      int     `json:"in_cart"`
}

// ProductsResponse encapsulates the list of products returned from the API.
type ProductsResponse struct {
	Data  []Product              `json:"data"`
	Links map[string]interface{} `json:"links"`
}

// CartItem represents a single item within a cart.
type CartItem struct {
	ProductID   int     `json:"product_id"`
	Quantity    int     `json:"quantity"`
	ProductName string  `json:"product_name"`
	Price       float64 `json:"product_price"`
}

// AccountInfo represents a user's information
type AccountInfo struct {
	Data struct {
		FirstName         string    `json:"first_name"`
		LastName          string    `json:"last_name"`
		Address           string    `json:"address"`
		Email             string    `json:"email"`
		Phone             string    `json:"phone"`
		Image             string    `json:"image"`
		CreatedDate       time.Time `json:"created_at"`
		DaysSinceCreation int       `json:"days_since_creation,omitempty"`
	} `json:"data"`
}

// CartResponse encapsulates the list of cart items returned from the API.
type CartResponse struct {
	Data struct {
		Products []CartItem `json:"products"`
	} `json:"data"`
}
