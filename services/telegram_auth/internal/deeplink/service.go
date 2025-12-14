package deeplinkservice

import (
	"context"
)

// GenerateLinkPayload is the payload for the GenerateLink method
type GenerateLinkPayload struct {
	UserID      string
	RedirectURL string
}

// GenerateLinkResult is the result for the GenerateLink method
type GenerateLinkResult struct {
	Link    string
	QRCode  string
}

// ValidateLinkPayload is the payload for the ValidateLink method
type ValidateLinkPayload struct {
	Link string
}

// ValidateLinkResult is the result for the ValidateLink method
type ValidateLinkResult struct {
	Valid  bool
	UserID string
}

// DeepLinkService is the interface for the deep link service
type DeepLinkService interface {
	GenerateLink(context.Context, *GenerateLinkPayload) (*GenerateLinkResult, error)
	ValidateLink(context.Context, *ValidateLinkPayload) (*ValidateLinkResult, error)
}