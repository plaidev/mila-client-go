package mila

import (
	"context"
	"fmt"
	"sync"

	"cloud.google.com/go/auth"
	"cloud.google.com/go/auth/credentials"
	"cloud.google.com/go/auth/credentials/idtoken"
)

// TokenProvider は有効な Bearer token を返す。複数の goroutine から呼ばれる場合がある。
type TokenProvider func(context.Context) (string, error)

type authentication struct {
	bearerToken string
	provider    TokenProvider
	iapClientID string
	keyFilename string
	mu          sync.Mutex
	credentials *auth.Credentials
}

func (a *authentication) authorization(ctx context.Context, refresh bool) (string, error) {
	if a.bearerToken != "" {
		return "Bearer " + a.bearerToken, nil
	}
	if a.provider != nil {
		token, err := a.provider(ctx)
		return authorization(token, err)
	}
	if a.iapClientID == "" {
		return "", nil
	}
	creds, err := a.iapCredentials(refresh)
	if err != nil {
		return "", &AuthenticationError{Err: err}
	}
	token, err := creds.Token(ctx)
	if err != nil {
		return "", &AuthenticationError{Err: err}
	}
	return authorization(token.Value, nil)
}

func authorization(token string, err error) (string, error) {
	if err == nil && token == "" {
		err = fmt.Errorf("token provider returned an empty token")
	}
	if err != nil {
		return "", &AuthenticationError{Err: err}
	}
	return "Bearer " + token, nil
}

func (a *authentication) iapCredentials(refresh bool) (*auth.Credentials, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if refresh {
		a.credentials = nil
	}
	if a.credentials == nil {
		opts := &idtoken.Options{Audience: a.iapClientID}
		var err error
		if a.keyFilename != "" {
			a.credentials, err = idtoken.NewCredentialsFromFile(credentials.ServiceAccount, a.keyFilename, opts)
		} else {
			a.credentials, err = idtoken.NewCredentials(opts)
		}
		if err != nil {
			return nil, err
		}
	}
	return a.credentials, nil
}
