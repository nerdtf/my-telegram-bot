package auth

import (
	"errors"
	"net/http"
	"strings"
	"sync"
)

type AuthClient struct {
	Tokens map[int64]string
	mu     sync.RWMutex
}

func NewAuthClient() *AuthClient {
	return &AuthClient{}
}
func (ac *AuthClient) SetToken(token string, chatID int64) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.Tokens[chatID] = token
}

func (ac *AuthClient) GetToken(chatID int64) string {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.Tokens[chatID]
}

func (ac *AuthClient) RefreshToken(apiBaseURL string, chatID int64) error {
	url := apiBaseURL + "/refresh"

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	token := ac.GetToken(chatID)
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to refresh token")
	}

	newToken := resp.Header.Get("Authorization")

	if newToken == "" {
		return errors.New("missing new token in response")
	}

	ac.SetToken(strings.TrimPrefix(newToken, "Bearer "), chatID)
	return nil
}
