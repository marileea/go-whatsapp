package design

// Design defines the API design for the Telegram Authentication Service
// This file provides the service specification that would normally be used with Goa generation tools

// API Design Notes:
// - The design follows Goa-style service definitions with explicit API, service, and method declarations
// - This structure allows for future integration with Goa code generation tools
// - Currently, the service is implemented manually following Goa conventions
//
// Services included:
// 1. Auth - JWT token management (validate, refresh, logout)
// 2. Deep Link - QR code generation and validation
// 3. Webhook - Telegram bot webhook handling
//
// Each service provides clear payloads, results, and HTTP transports
//
// To regenerate with Goa (when Go 1.24+ is available):
// 1. Install Goa: go install goa.design/goa/v3/cmd/goa@v3
// 2. Generate: goa gen design/design.go

var DesignInfo = map[string]string{
	"service":  "telegram-auth",
	"version":  "1.0.0",
	"protocol": "HTTP/JSON",
	"format":   "REST",
}