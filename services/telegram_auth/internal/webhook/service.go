package webhookservice

import (
	"context"
)

// TelegramUpdatePayload is the payload for the TelegramUpdate method
type TelegramUpdatePayload struct {
	UpdateID      int
	Message       map[string]interface{}
	CallbackQuery map[string]interface{}
}

// TelegramUpdateResult is the result for the TelegramUpdate method
type TelegramUpdateResult struct {
	Success  bool
	Response string
}

// VerifyWebhookPayload is the payload for the VerifyWebhook method
type VerifyWebhookPayload struct {
	URL string
}

// VerifyWebhookResult is the result for the VerifyWebhook method
type VerifyWebhookResult struct {
	Verified bool
	Result   string
}

// WebhookService is the interface for the webhook service
type WebhookService interface {
	TelegramUpdate(context.Context, *TelegramUpdatePayload) (*TelegramUpdateResult, error)
	VerifyWebhook(context.Context, *VerifyWebhookPayload) (*VerifyWebhookResult, error)
}