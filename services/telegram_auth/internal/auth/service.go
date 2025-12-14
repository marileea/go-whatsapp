package authservice

import (
	"context"
)

// ValidateTokenPayload is the payload for the ValidateToken method
type ValidateTokenPayload struct {
	Token string
}

// ValidateTokenResult is the result for the ValidateToken method
type ValidateTokenResult struct {
	Valid     bool
	UserID    string
	ExpiresAt string
}

// RefreshTokenPayload is the payload for the RefreshToken method
type RefreshTokenPayload struct {
	RefreshToken string
}

// RefreshTokenResult is the result for the RefreshToken method
type RefreshTokenResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    string
}

// LogoutPayload is the payload for the Logout method
type LogoutPayload struct {
	Token string
}

// LogoutResult is the result for the Logout method
type LogoutResult struct {
	Success bool
}

// AuthService is the interface for the auth service
type AuthService interface {
	ValidateToken(context.Context, *ValidateTokenPayload) (*ValidateTokenResult, error)
	RefreshToken(context.Context, *RefreshTokenPayload) (*RefreshTokenResult, error)
	Logout(context.Context, *LogoutPayload) (*LogoutResult, error)
}