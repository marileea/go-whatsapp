package webhookservice

import (
    "context"
    "fmt"
)

// Service represents the webhook service implementation
type Service struct {
    botToken string
    webhookURL string
}

// New creates a new webhook service
func New(botToken, webhookURL string) *Service {
    return &Service{
        botToken:   botToken,
        webhookURL: webhookURL,
    }
}

// TelegramUpdate handles incoming Telegram updates
func (s *Service) TelegramUpdate(ctx context.Context, payload *TelegramUpdatePayload) (*TelegramUpdateResult, error) {
    // Log the update
    fmt.Printf("Received Telegram update: %d\n", payload.UpdateID)

    // Handle callback query
    if payload.CallbackQuery != nil {
        return s.handleCallbackQuery(payload.CallbackQuery)
    }

    // Handle regular message
    if payload.Message != nil {
        return s.handleMessage(payload.Message)
    }

    return &TelegramUpdateResult{
        Success:  true,
        Response: "Update processed successfully",
    }, nil
}

// VerifyWebhook verifies a webhook URL with Telegram
func (s *Service) VerifyWebhook(ctx context.Context, payload *VerifyWebhookPayload) (*VerifyWebhookResult, error) {
    // In a real implementation, you would use the Telegram Bot API to set the webhook
    // For now, we just simulate the verification
    webhookURL := payload.URL
    if webhookURL == "" {
        webhookURL = s.webhookURL
    }

    if webhookURL == "" {
        return &VerifyWebhookResult{
            Verified: false,
            Result:   "No webhook URL provided",
        }, nil
    }

    // Simulate webhook verification
    // In production, you would make an HTTP request to Telegram's API
    if !isValidURL(webhookURL) {
        return &VerifyWebhookResult{
            Verified: false,
            Result:   "Invalid webhook URL format",
        }, nil
    }

    return &VerifyWebhookResult{
        Verified: true,
        Result:   fmt.Sprintf("Webhook verified successfully at %s", webhookURL),
    }, nil
}

// handleCallbackQuery handles Telegram callback queries
func (s *Service) handleCallbackQuery(callbackQuery map[string]interface{}) (*TelegramUpdateResult, error) {
    data, ok := callbackQuery["data"].(string)
    if !ok {
        return &TelegramUpdateResult{
            Success:  false,
            Response: "Invalid callback query data",
        }, nil
    }

    fmt.Printf("Processing callback query: %s\n", data)

    return &TelegramUpdateResult{
        Success:  true,
        Response: "Callback query processed",
    }, nil
}

// handleMessage handles Telegram messages
func (s *Service) handleMessage(message map[string]interface{}) (*TelegramUpdateResult, error) {
    text, ok := message["text"].(string)
    if !ok {
        text = ""
    }

    chatID, ok := message["chat"].(map[string]interface{})["id"].(float64)
    if !ok {
        return &TelegramUpdateResult{
            Success:  false,
            Response: "Invalid chat ID",
        }, nil
    }

    fmt.Printf("Received message: %s from chat: %.0f\n", text, chatID)

    // In a real implementation, you would process the message and potentially send a response
    responseText := fmt.Sprintf("Received your message: %s", text)
    
    return &TelegramUpdateResult{
        Success:  true,
        Response: responseText,
    }, nil
}

// isValidURL checks if a URL is valid
func isValidURL(url string) bool {
    if url == "" {
        return false
    }

    // Basic URL validation
    if len(url) < 10 {
        return false
    }

    if url[:8] != "https://" && url[:7] != "http://" {
        return false
    }

    return true
}