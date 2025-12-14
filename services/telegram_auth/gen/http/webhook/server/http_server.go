package httpwebhook

import (
	"context"

	webhookservice "github.com/godeirect/go-whatsapp-multidevice-api/services/telegram_auth/internal/webhook"
)

// Service interface for the HTTP transport
type Service interface {
	TelegramUpdate(context.Context, *TelegramUpdateRequest) (*TelegramUpdateResponse, error)
	VerifyWebhook(context.Context, *VerifyWebhookRequest) (*VerifyWebhookResponse, error)
}

// Controller handles HTTP requests for webhook service
type Controller struct {
	webhookService webhookservice.WebhookService
}

// NewController creates a new webhook HTTP controller
func NewController(webhookService webhookservice.WebhookService) *Controller {
	return &Controller{
		webhookService: webhookService,
	}
}

// TelegramUpdate handles incoming Telegram updates
func (c *Controller) TelegramUpdate(ctx context.Context, req *TelegramUpdateRequest) (*TelegramUpdateResponse, error) {
	payload := &webhookservice.TelegramUpdatePayload{
		UpdateID:      req.UpdateID,
		Message:       req.Message,
		CallbackQuery: req.CallbackQuery,
	}

	result, err := c.webhookService.TelegramUpdate(ctx, payload)
	if err != nil {
		return nil, err
	}

	return &TelegramUpdateResponse{
		Success:  result.Success,
		Response: result.Response,
	}, nil
}

// VerifyWebhook verifies a webhook URL
func (c *Controller) VerifyWebhook(ctx context.Context, req *VerifyWebhookRequest) (*VerifyWebhookResponse, error) {
	payload := &webhookservice.VerifyWebhookPayload{
		URL: req.URL,
	}

	result, err := c.webhookService.VerifyWebhook(ctx, payload)
	if err != nil {
		return nil, err
	}

	return &VerifyWebhookResponse{
		Verified: result.Verified,
		Result:   result.Result,
	}, nil
}

// --- Request/Response structs ---

// TelegramUpdateRequest is the request struct for TelegramUpdate
type TelegramUpdateRequest struct {
	UpdateID      int
	Message       map[string]interface{}
	CallbackQuery map[string]interface{}
}

// TelegramUpdateResponse is the response struct for TelegramUpdate
type TelegramUpdateResponse struct {
	Success  bool
	Response string
}

// VerifyWebhookRequest is the request struct for VerifyWebhook
type VerifyWebhookRequest struct {
	URL string
}

// VerifyWebhookResponse is the response struct for VerifyWebhook
type VerifyWebhookResponse struct {
	Verified bool
	Result   string
}