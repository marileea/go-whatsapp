package authservice

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents the claims in a JWT token
type JWTClaims struct {
	UserID    string `json:"user_id"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

// Service represents the auth service implementation
type Service struct {
	jwtSecret string
	tokenTTL  time.Duration
}

// New creates a new auth service
func New(jwtSecret string, tokenTTL time.Duration) *Service {
	return &Service{
		jwtSecret: jwtSecret,
		tokenTTL:  tokenTTL,
	}
}

// ValidateToken validates a JWT token
func (s *Service) ValidateToken(ctx context.Context, payload *ValidateTokenPayload) (*ValidateTokenResult, error) {
	claims, err := s.parseToken(payload.Token)
	if err != nil {
		return &ValidateTokenResult{
			Valid: false,
		}, nil
	}

	return &ValidateTokenResult{
		Valid:     true,
		UserID:    claims.UserID,
		ExpiresAt: claims.ExpiresAt.Time.Format(time.RFC3339),
	}, nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *Service) RefreshToken(ctx context.Context, payload *RefreshTokenPayload) (*RefreshTokenResult, error) {
	// Validate the refresh token
	claims, err := s.parseToken(payload.RefreshToken)
	if err != nil {
		return nil, err
	}

	// Generate new access token
	accessToken, err := s.generateToken(claims.UserID, "access", s.tokenTTL)
	if err != nil {
		return nil, err
	}

	// Generate new refresh token (with longer TTL)
	refreshToken, err := s.generateToken(claims.UserID, "refresh", time.Hour*24*7)
	if err != nil {
		return nil, err
	}

	expTime := time.Now().Add(s.tokenTTL)

	return &RefreshTokenResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expTime.Format(time.RFC3339),
	}, nil
}

// Logout revokes a token
func (s *Service) Logout(ctx context.Context, payload *LogoutPayload) (*LogoutResult, error) {
	// In a production system, you would add the token to a blacklist
	// For now, we just validate that the token is well-formed
	_, err := s.parseToken(payload.Token)
	if err != nil {
		return &LogoutResult{
			Success: false,
		}, nil
	}

	return &LogoutResult{
		Success: true,
	}, nil
}

// generateToken generates a new JWT token
func (s *Service) generateToken(userID, tokenType string, ttl time.Duration) (string, error) {
	expirationTime := time.Now().Add(ttl)
	
	claims := &JWTClaims{
		UserID:    userID,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// parseToken parses and validates a JWT token
func (s *Service) parseToken(tokenString string) (*JWTClaims, error) {
	claims := &JWTClaims{}
	
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})
	
	if err != nil {
		return nil, err
	}
	
	if !token.Valid {
		return nil, jwt.ErrInvalidKey
	}
	
	return claims, nil
}