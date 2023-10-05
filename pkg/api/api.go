package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"my-telegram-bot/pkg/auth"
	"net/http"
	"net/url"
	"time"
)

// NewAPIClient creates a new instance of the APIClient with the specified baseURL.
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Register sends a request to the API to register a new user with the provided data.
func (api *APIClient) Register(data RegisterData, chatID int64, authClient *auth.AuthClient) (*ValidationError, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add the image as a file part to the request
	if data.ImageData != nil {
		part, err := w.CreateFormFile("image", "profile_image")
		if err != nil {
			return nil, &Error{Err: err, Message: "Failed to create request"}
		}
		part.Write(data.ImageData)
	}

	// Define form fields
	fields := map[string]string{
		"address": data.Address,
		"email":   data.Email,
		"phone":   data.Phone,
	}
	// Conditionally add name fields if they are not empty
	if data.FirstName != "" {
		fields["first_name"] = data.FirstName
	}
	if data.LastName != "" {
		fields["last_name"] = data.LastName
	}

	// Add other form fields
	for key, value := range fields {
		if err := w.WriteField(key, value); err != nil {
			return nil, &Error{Err: err, Message: "Failed to write fields"}
		}
	}

	// Close the multipart writer
	if err := w.Close(); err != nil {
		return nil, &Error{Err: err, Message: "Failed to close writer"}
	}

	// Send the request
	req, err := http.NewRequest("POST", api.BaseURL+"/client/register", &b)
	if err != nil {
		return nil, &Error{Err: err, Message: "Failed to send request"}
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, &Error{Err: err, Message: "Failed to do request"}
	}
	//defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, &Error{Err: err, Message: "Failed to read response body"}
		}
		// Check if the status code indicates a validation error
		if resp.StatusCode == http.StatusUnprocessableEntity {
			var ve ValidationError
			json.Unmarshal(bodyBytes, &ve)
			return &ve, &Error{Err: errors.New(string(bodyBytes)), Message: "Validation error"}
		}

		return nil, &Error{Err: errors.New(string(bodyBytes)), Message: "API error"}

	}

	var registerResponse RegisterResponse
	api.decodeResponse(resp, &registerResponse)
	// Save the token using the authClient
	authClient.SetToken(registerResponse.Data.Token, chatID)

	return nil, nil
}

// decodeResponse decodes the HTTP response into the specified result.
// If the response status code is 400 or higher, it returns an error.
func (api *APIClient) decodeResponse(resp *http.Response, result interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return &Error{Err: err, Message: "Failed to read response"}
		}
		if resp.StatusCode == 422 { // Unprocessable Entity
			var errorResponse struct {
				Errors map[string][]string `json:"errors"`
			}
			err = json.Unmarshal(bodyBytes, &errorResponse)
			if err == nil {
				if cartErrors, exists := errorResponse.Errors["cart"]; exists && len(cartErrors) > 0 {
					return &Error{Err: errors.New(cartErrors[0]), Message: cartErrors[0]}
				}
			}
		}
		if resp.StatusCode == http.StatusUnauthorized {
			var errorResponse struct {
				Message string `json:"message"`
			}
			err := json.Unmarshal(bodyBytes, &errorResponse)
			if err != nil {
				return &Error{Err: err, Message: "Failed to decode unauthorized response"}
			}
			if errorResponse.Message == "Token has expired" || errorResponse.Message == "Unauthenticated." {
				return &Error{Err: err, Message: "Token has expired"}
			}
		}
		// Check if the status code indicates a validation error
		if resp.StatusCode == http.StatusUnprocessableEntity {
			var ve ValidationError
			if err := json.Unmarshal(bodyBytes, &ve); err != nil {
				return &Error{Err: err, Message: "Failed to decode validation error"}
			}
			return &Error{Err: errors.New("Validation error"), Message: "Validation error", Details: &ve}
		}
		return &Error{Err: errors.New(string(bodyBytes)), Message: "API error"}
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return &Error{Err: err, Message: "Error decoding response"}
	}

	return nil
}

// Login sends a request to the API to log in a user with the provided data.
// It returns the access token if successful.
func (api *APIClient) Login(data LoginData) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	resp, err := api.client.Post(api.BaseURL+"/client/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return "", errors.New(string(bodyBytes))
	}

	var loginResponse LoginResponse
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(bodyBytes, &loginResponse)
	if err != nil {
		return "", err
	}

	return loginResponse.Data, nil
}

// makeAPIRequest creates and sends an API request. If the token is expired, it refreshes the token and retries.
func (api *APIClient) makeAPIRequest(method, url string, body io.Reader, authClient *auth.AuthClient, chatID int64, contentType ...string) (*http.Response, error) {
	defaultContentType := "application/json"

	// Convert the io.Reader content to a byte slice
	bodyBytes, err := readerToBytes(body)
	if err != nil {
		return nil, &Error{Err: err, Message: "Failed to read body content"}
	}

	for i := 0; i < 2; i++ {
		req, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, &Error{Err: err, Message: "New request error"}
		}
		// Add the token to the request headers
		token := authClient.GetToken(chatID)
		req.Header.Add("Authorization", "Bearer "+token)
		// Set content type
		if len(contentType) > 0 && contentType[0] != "" {
			req.Header.Add("Content-Type", contentType[0])
		} else {
			req.Header.Add("Content-Type", defaultContentType)
		}

		req.Header.Add("Accept", "application/json")

		response, err := api.client.Do(req)
		if err != nil {
			return nil, &Error{Err: err, Message: "Response error"}
		}
		// If the response contains a token expired error, refresh the token and retry the API call
		if api.isTokenExpired(response) {
			if err := authClient.RefreshToken(api.BaseURL, chatID); err != nil {
				return nil, &Error{Err: err, Message: "Error while refreshing token"}
			}
			continue
		}
		return response, nil
	}

	return nil, errors.New("failed to make API request with token refresh")
}

func readerToBytes(reader io.Reader) ([]byte, error) {
	if reader == nil {
		return nil, nil
	}

	var buf bytes.Buffer
	_, err := buf.ReadFrom(reader)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GetProducts fetches a list of products with pagination. It also updates the 'InCart' field for each product based on the items in the cart.
func (api *APIClient) GetProducts(perPage int, page int, authClient *auth.AuthClient, chatID int64, search string) ([]Product, bool, error) {

	urlStr := fmt.Sprintf("%s/products?per_page=%d&page=%d", api.BaseURL, perPage, page)
	if search != "" {
		urlStr = fmt.Sprintf("%s&search=%s", urlStr, url.QueryEscape(search))
	}
	resp, err := api.makeAPIRequest("", urlStr, nil, authClient, chatID)
	if err != nil {
		return nil, false, err
	}

	var productsResponse ProductsResponse
	if err := api.decodeResponse(resp, &productsResponse); err != nil {
		return nil, false, err
	}

	next := productsResponse.Links["next"]

	// Get the cart items
	cartItems, err := api.GetCartItems(authClient, false, chatID)

	if err != nil {
		return nil, false, err
	}

	// Create a map to store product IDs and their quantities in the cart
	cartItemsMap := make(map[int]int)
	for _, cartItem := range cartItems {
		cartItemsMap[cartItem.ProductID] = cartItem.Quantity
	}

	// Update the products with the number of items in the cart
	for i, product := range productsResponse.Data {
		if quantity, ok := cartItemsMap[product.ID]; ok {
			productsResponse.Data[i].InCart = quantity
		} else {
			productsResponse.Data[i].InCart = 0
		}
	}

	return productsResponse.Data, next != nil, nil
}

// GetCartItems fetches items in the cart. If showNames is true, it also fetches the names and prices of the products.
func (api *APIClient) GetCartItems(authClient *auth.AuthClient, showNames bool, chatID int64) ([]CartItem, error) {
	url := fmt.Sprintf("%s/cart?showNamesAndPrices=%t", api.BaseURL, showNames)
	resp, err := api.makeAPIRequest("", url, nil, authClient, chatID)

	if err != nil {
		return nil, err
	}
	var cartResponse CartResponse
	if err := api.decodeResponse(resp, &cartResponse); err != nil {
		return nil, err
	}
	return cartResponse.Data.Products, nil
}

// AddProductToCart adds a product with a specific quantity to the cart.
func (api *APIClient) AddProductToCart(productID, quantity int, authClient *auth.AuthClient, chatID int64) error {
	url := fmt.Sprintf("%s/cart", api.BaseURL)

	data := map[string]int{
		"product_id": productID,
		"quantity":   quantity,
	}
	jsonData, err := json.Marshal(data)

	if err != nil {
		return &Error{Err: err, Message: "Failed to json decode"}
	}
	resp, err := api.makeAPIRequest(http.MethodPost, url, bytes.NewBuffer(jsonData), authClient, chatID)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return api.decodeResponse(resp, nil)
	}

	return nil
}

// RemoveProductFromCart removes a product from the cart. If deleteWholeProduct is true, it removes all quantities of this product from the cart.
func (api *APIClient) RemoveProductFromCart(productID int, authClient *auth.AuthClient, chatID int64, deleteWholeProduct bool) error {
	url := fmt.Sprintf("%s/cart/%d?delete_whole_product=%t", api.BaseURL, productID, deleteWholeProduct)

	resp, err := api.makeAPIRequest(http.MethodDelete, url, nil, authClient, chatID)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return api.decodeResponse(resp, nil)
	}

	return nil
}

// isTokenExpired checks if the API response indicates an expired token.
func (api *APIClient) isTokenExpired(response *http.Response) bool {
	if response.StatusCode == http.StatusUnauthorized {
		var errorResponse struct {
			Message string `json:"message"`
		}

		if err := api.decodeResponse(response, &errorResponse); err != nil {
			return false
		}
		if errorResponse.Message == "Token has expired" || errorResponse.Message == "Unauthenticated." {
			return true
		}
	}

	return false
}

// CompleteOrder completes the order and returns the order details.
func (api *APIClient) CompleteOrder(authClient *auth.AuthClient, chatID int64) (*CompleteOrderResponse, error) {
	url := api.BaseURL + "/orders"

	resp, err := api.makeAPIRequest("POST", url, nil, authClient, chatID)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		if err := api.decodeResponse(resp, nil); err != nil {
			if apiErr, ok := err.(*Error); ok {
				return nil, fmt.Errorf("%s", apiErr.Message)
			}
			return nil, err
		}
		return nil, fmt.Errorf("API returned unexpected status code: %d", resp.StatusCode)

	}

	var orderResponse CompleteOrderResponse
	if err := api.decodeResponse(resp, &orderResponse); err != nil {

		return nil, err
	}
	return &orderResponse, nil
}

func calcDifferenceInDates(date time.Time) int {
	// Get current time
	currentTime := time.Now()

	// Calculate difference in days
	daysDiff := currentTime.Sub(date).Hours() / 24

	return int(daysDiff)
}

func (api *APIClient) GetAccountInfo(authClient *auth.AuthClient, chatID int64) (*AccountInfo, error) {
	url := api.BaseURL + "/client"
	resp, err := api.makeAPIRequest("", url, nil, authClient, chatID)

	if err != nil {
		return nil, err
	}

	var accountInfo AccountInfo
	if err := api.decodeResponse(resp, &accountInfo); err != nil {
		return nil, err
	}

	accountInfo.Data.DaysSinceCreation = calcDifferenceInDates(accountInfo.Data.CreatedDate)

	return &accountInfo, nil
}

// UpdateField updates a specific field for a user's account.
func (api *APIClient) UpdateField(chatID int64, authClient *auth.AuthClient, fieldName string, fieldValue interface{}) (*AccountInfo, error) {
	url := api.BaseURL + "/client/update"

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	var contentType string

	if fieldName == "image" {
		imageData, ok := fieldValue.([]byte)
		if !ok {
			return nil, &Error{Message: "Expected image data as []byte"}
		}
		part, err := w.CreateFormFile("image", "profile_image")
		if err != nil {
			return nil, &Error{Err: err, Message: "Failed to create multipart form"}
		}
		part.Write(imageData)
	} else {
		valueStr, ok := fieldValue.(string)
		if !ok {
			return nil, &Error{Message: "Expected value as string"}
		}
		w.WriteField(fieldName, valueStr)
	}

	if err := w.Close(); err != nil {
		return nil, &Error{Err: err, Message: "Failed to close writer"}
	}

	body := &b
	contentType = w.FormDataContentType()

	// Create the request
	resp, err := api.makeAPIRequest(http.MethodPost, url, body, authClient, chatID, contentType)
	if err != nil {
		return nil, err
	}

	var accountInfo AccountInfo
	if err := api.decodeResponse(resp, &accountInfo); err != nil {
		return nil, err
	}
	accountInfo.Data.DaysSinceCreation = calcDifferenceInDates(accountInfo.Data.CreatedDate)

	return &accountInfo, nil

}

// GetOrderHistory retrieves the order history and returns the order details.
func (api *APIClient) GetOrderHistory(authClient *auth.AuthClient, chatID int64) (*OrderHistoryResponse, error) {
	url := api.BaseURL + "/orders"
	resp, err := api.makeAPIRequest("", url, nil, authClient, chatID)

	if err != nil {
		return nil, err
	}
	var orderHistoryResponse OrderHistoryResponse
	if err := api.decodeResponse(resp, &orderHistoryResponse); err != nil {
		return nil, err
	}
	return &orderHistoryResponse, nil
}
