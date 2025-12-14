# Telegram Authentication Service - Quick Start

## Overview
This is a production-ready Goa-style microservice for handling Telegram authentication, deep link generation, and webhook processing.

## Project Structure
```
services/telegram_auth/
├── cmd/server/main.go          # Application entry point
├── config/config.go             # Configuration management
├── design/design.go             # Goa-style API design
├── internal/                    # Business logic
│   ├── auth/                    # JWT authentication service
│   ├── deeplink/                # Deep link & QR code generation
│   ├── webhook/                 # Telegram webhook handling
│   └── logger/                  # Structured logging
├── gen/http/*/server/           # HTTP transport layer
├── Dockerfile                   # Container build configuration
├── .env                         # Environment configuration
├── .env.example                 # Environment template
├── go.mod                       # Go module definition
└── README.md                    # Comprehensive documentation
```

## Quick Test
```bash
cd services/telegram_auth

# Start the service
TELEGRAM_AUTH_TELEGRAM_BOT_TOKEN=test \
TELEGRAM_AUTH_JWT_SECRET=test_secret_32_chars_long \
TELEGRAM_AUTH_MYSQL_DSN=test \
./server

# In another terminal, test the endpoints:
curl http://localhost:8080/health
```

## Docker Deployment
```bash
# From project root
docker-compose up -d --build
```

## Key Features
✅ JWT token validation, refresh, and logout
✅ Deep link generation with QR codes
✅ Telegram webhook processing
✅ Environment-based configuration with validation
✅ Graceful shutdown with proper cleanup
✅ Structured logging with context
✅ Health check endpoint
✅ Docker multi-stage build
✅ MySQL integration
✅ CORS support
✅ Production-ready error handling

## API Endpoints
- `GET /health` - Service health check
- `POST /auth/validate` - Validate JWT token
- `POST /auth/refresh` - Refresh access token
- `POST /auth/logout` - Revoke token
- `POST /deeplink/generate` - Generate authentication deep link
- `POST /deeplink/validate` - Validate deep link
- `POST /webhook/telegram` - Handle Telegram updates
- `POST /webhook/verify` - Verify webhook configuration