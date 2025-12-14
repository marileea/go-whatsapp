package httpdeeplink

import (
	"context"

	deeplinkservice "github.com/godeirect/go-whatsapp-multidevice-api/services/telegram_auth/internal/deeplink"
)

// Service interface for the HTTP transport
type Service interface {
	GenerateLink(context.Context, *GenerateLinkRequest) (*GenerateLinkResponse, error)
	ValidateLink(context.Context, *ValidateLinkRequest) (*ValidateLinkResponse, error)
}

// Controller handles HTTP requests for deep-link service
type Controller struct {
	deepLinkService deeplinkservice.DeepLinkService
}

// NewController creates a new deep-link HTTP controller
func NewController(deepLinkService deeplinkservice.DeepLinkService) *Controller {
	return &Controller{
		deepLinkService: deepLinkService,
	}
}

// GenerateLink generates a deep link with QR code
func (c *Controller) GenerateLink(ctx context.Context, req *GenerateLinkRequest) (*GenerateLinkResponse, error) {
	payload := &deeplinkservice.GenerateLinkPayload{
		UserID:      req.UserID,
		RedirectURL: req.RedirectURL,
	}

	result, err := c.deepLinkService.GenerateLink(ctx, payload)
	if err != nil {
		return nil, err
	}

	return &GenerateLinkResponse{
		Link:   result.Link,
		QRCode: result.QRCode,
	}, nil
}

// ValidateLink validates a deep link
func (c *Controller) ValidateLink(ctx context.Context, req *ValidateLinkRequest) (*ValidateLinkResponse, error) {
	payload := &deeplinkservice.ValidateLinkPayload{
		Link: req.Link,
	}

	result, err := c.deepLinkService.ValidateLink(ctx, payload)
	if err != nil {
		return nil, err
	}

	return &ValidateLinkResponse{
		Valid:  result.Valid,
		UserID: result.UserID,
	}, nil
}

// --- Request/Response structs ---

// GenerateLinkRequest is the request struct for GenerateLink
type GenerateLinkRequest struct {
	UserID      string
	RedirectURL string
}

// GenerateLinkResponse is the response struct for GenerateLink
type GenerateLinkResponse struct {
	Link   string
	QRCode string
}

// ValidateLinkRequest is the request struct for ValidateLink
type ValidateLinkRequest struct {
	Link string
}

// ValidateLinkResponse is the response struct for ValidateLink
type ValidateLinkResponse struct {
	Valid  bool
	UserID string
}