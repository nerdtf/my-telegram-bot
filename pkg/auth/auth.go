package auth

import "sync"

type AuthClient struct {
	token string
	mu    sync.RWMutex
}

func NewAuthClient() *AuthClient {
	return &AuthClient{}
}

func (ac *AuthClient) SetToken(token string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.token = token
}

func (ac *AuthClient) GetToken() string {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.token
}
