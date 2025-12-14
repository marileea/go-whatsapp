package deeplinkservice

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/skip2/go-qrcode"
)

// Service represents the deep link service implementation
type Service struct {
	baseURL string
}

// New creates a new deep link service
func New(baseURL string) *Service {
	return &Service{
		baseURL: baseURL,
	}
}

// GenerateLink generates a deep link with QR code
func (s *Service) GenerateLink(ctx context.Context, payload *GenerateLinkPayload) (*GenerateLinkResult, error) {
	// Generate unique token for the deep link
	token := fmt.Sprintf("%s_%d_%s", payload.UserID, time.Now().Unix(), s.generateToken())

	// Create the deep link
	link := fmt.Sprintf("%s/telegram/auth?token=%s&redirect=%s", s.baseURL, token, payload.RedirectURL)

	// Generate QR code
	qrCode, err := qrcode.New(link, qrcode.Medium)
	if err != nil {
		return nil, err
	}

	// Encode QR code to base64
	qrCodeData, err := qrCode.PNG(256)
	if err != nil {
		return nil, err
	}
	qrCodeBase64 := base64.StdEncoding.EncodeToString(qrCodeData)

	return &GenerateLinkResult{
		Link:    link,
		QRCode:  qrCodeBase64,
	}, nil
}

// ValidateLink validates a deep link
func (s *Service) ValidateLink(ctx context.Context, payload *ValidateLinkPayload) (*ValidateLinkResult, error) {
	// Parse the link
	if payload.Link == "" {
		return &ValidateLinkResult{
			Valid: false,
		}, nil
	}

	// Basic validation - in a real implementation, you'd check the token format
	// and verify it hasn't expired or been used
	link := payload.Link
	
	// Check if link starts with expected base URL
	if s.baseURL != "" && !hasPrefix(link, s.baseURL) {
		return &ValidateLinkResult{
			Valid: false,
		}, nil
	}

	// Extract user ID from token (simplified)
	userID := s.extractUserID(link)
	if userID == "" {
		return &ValidateLinkResult{
			Valid: false,
		}, nil
	}

	return &ValidateLinkResult{
		Valid:  true,
		UserID: userID,
	}, nil
}

// generateToken generates a random token
func (s *Service) generateToken() string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return base64.URLEncoding.EncodeToString(hash[:])
}

// extractUserID extracts user ID from a deep link
func (s *Service) extractUserID(link string) string {
	// Simplified implementation - in real usage, you'd properly parse the URL
	// and extract the token parameter
	if len(link) > 10 {
		return link[len(link)-10:] // Simplified - just return last 10 chars
	}
	return ""
}

// hasPrefix is a helper function to check string prefix
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}