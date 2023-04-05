package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"my-telegram-bot/pkg/auth"
	"net/http"
	"os"
	//"strings"
)

// APIClient is a struct that holds the base URL for the API and an HTTP client.
type APIClient struct {
	BaseURL string
	client  *http.Client
}

// NewAPIClient creates a new instance of the APIClient with the specified baseURL.
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		client:  &http.Client{},
	}
}

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

// RegisterData holds the data required for user registration.
type RegisterData struct {
	LastName  string `json:"last_name"`
	FirstName string `json:"first_name"`
	ImageData []byte `json:"-"`
	Address   string `json:"address"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
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

// ValidationError represents an error that occurs during validation.
type ValidationError struct {
	Message string
	Errors  map[string][]string
}

// Error returns the error message for the ValidationError.
func (ve *ValidationError) Error() string {
	return ve.Message
}

// Register sends a request to the API to register a new user with the provided data.
func (api *APIClient) Register(data RegisterData, authClient *auth.AuthClient) (*ValidationError, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add the image as a file part to the request
	if len(data.ImageData) > 0 {
		part, err := w.CreateFormFile("image", "profile_image")
		if err != nil {
			return nil, err
		}
		part.Write(data.ImageData)
	}

	// Add other form fields
	for key, value := range map[string]string{
		"last_name":  data.LastName,
		"first_name": data.FirstName,
		"address":    data.Address,
		"email":      data.Email,
		"phone":      data.Phone,
	} {
		if err := w.WriteField(key, value); err != nil {
			return nil, err
		}
	}

	// Close the multipart writer
	if err := w.Close(); err != nil {
		return nil, err
	}

	// Send the request
	req, err := http.NewRequest("POST", api.BaseURL+"/client/register", &b)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		// Check if the status code indicates a validation error
		if resp.StatusCode == http.StatusUnprocessableEntity {
			var ve ValidationError
			json.Unmarshal(bodyBytes, &ve)
			return &ve, nil
		}
		return nil, errors.New(string(bodyBytes))
	}

	var registerResponse RegisterResponse
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bodyBytes, &registerResponse)
	if err != nil {
		return nil, err
	}

	// Save the token using the authClient
	authClient.SetToken(registerResponse.Data.Token)

	return nil, nil
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

	fmt.Println(string(bodyBytes)) // Print the response body as a string
	os.Exit(1)
	err = json.Unmarshal(bodyBytes, &loginResponse)
	if err != nil {
		return "", err
	}

	return loginResponse.Data, nil
}

func makeAPIRequest(url string, authClient *auth.AuthClient) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add the token to the request headers
	token := authClient.GetToken()
	req.Header.Add("Authorization", "Bearer "+token)

	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// Add other methods for fetching products and managing orders here.
