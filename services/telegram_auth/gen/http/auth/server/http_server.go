package httpauth

import (
    "context"

    authservice "github.com/godeirect/go-whatsapp-multidevice-api/services/telegram_auth/internal/auth"
)

// Service interface for the HTTP transport
type Service interface {
    ValidateToken(context.Context, *ValidateTokenRequest) (*ValidateTokenResponse, error)
    RefreshToken(context.Context, *RefreshTokenRequest) (*RefreshTokenResponse, error)
    Logout(context.Context, *LogoutRequest) (*LogoutResponse, error)
}

// Controller handles HTTP requests for auth service
type Controller struct {
    authService authservice.AuthService
}

// NewController creates a new auth HTTP controller
func NewController(authService authservice.AuthService) *Controller {
    return &Controller{
        authService: authService,
    }
}

// ValidateToken validates a JWT token
func (c *Controller) ValidateToken(ctx context.Context, req *ValidateTokenRequest) (*ValidateTokenResponse, error) {
    payload := &authservice.ValidateTokenPayload{
        Token: req.Token,
    }

    result, err := c.authService.ValidateToken(ctx, payload)
    if err != nil {
        return nil, err
    }

    return &ValidateTokenResponse{
        Valid:     result.Valid,
        UserID:    result.UserID,
        ExpiresAt: result.ExpiresAt,
    }, nil
}

// RefreshToken refreshes an access token
func (c *Controller) RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*RefreshTokenResponse, error) {
    payload := &authservice.RefreshTokenPayload{
        RefreshToken: req.RefreshToken,
    }

    result, err := c.authService.RefreshToken(ctx, payload)
    if err != nil {
        return nil, err
    }

    return &RefreshTokenResponse{
        AccessToken:  result.AccessToken,
        RefreshToken: result.RefreshToken,
        ExpiresAt:    result.ExpiresAt,
    }, nil
}

// Logout revokes a token
func (c *Controller) Logout(ctx context.Context, req *LogoutRequest) (*LogoutResponse, error) {
    payload := &authservice.LogoutPayload{
        Token: req.Token,
    }

    result, err := c.authService.Logout(ctx, payload)
    if err != nil {
        return nil, err
    }

    return &LogoutResponse{
        Success: result.Success,
    }, nil
}

// --- Request/Response structs ---

// ValidateTokenRequest is the request struct for ValidateToken
type ValidateTokenRequest struct {
    Token string
}

// ValidateTokenResponse is the response struct for ValidateToken
type ValidateTokenResponse struct {
    Valid     bool
    UserID    string
    ExpiresAt string
}

// RefreshTokenRequest is the request struct for RefreshToken
type RefreshTokenRequest struct {
    RefreshToken string
}

// RefreshTokenResponse is the response struct for RefreshToken
type RefreshTokenResponse struct {
    AccessToken  string
    RefreshToken string
    ExpiresAt    string
}

// LogoutRequest is the request struct for Logout
type LogoutRequest struct {
    Token string
}

// LogoutResponse is the response struct for Logout
type LogoutResponse struct {
    Success bool
}