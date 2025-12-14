# Telegram Authentication Service

A Goa-style microservice for handling Telegram authentication, deep link generation, and webhook processing.

## Features

- **JWT-based Authentication**: Token validation, refresh, and logout
- **Deep Link Generation**: Generate QR-coded authentication links
- **Webhook Integration**: Handle Telegram bot webhooks and updates
- **Configurable**: Environment-based configuration with validation
- **Graceful Shutdown**: Proper service lifecycle management
- **Docker Support**: Multi-stage build with optimized runtime

## Architecture

This service follows a clean architecture pattern:

- **`cmd/server/`**: Application entry point with server bootstrap
- **`config/`**: Configuration management with environment variable support
- **`internal/`**: Business logic and services
  - `auth/`: JWT authentication service
  - `deeplink/`: Deep link generation and validation
  - `webhook/`: Telegram webhook handling
  - `logger/`: Structured logging with context support
- **`gen/`**: HTTP transport layer (controllers and handlers)
  - `http/auth/server/`: Auth HTTP endpoints
  - `http/deep_link/server/`: Deep link HTTP endpoints
  - `http/webhook/server/`: Webhook HTTP endpoints

## API Endpoints

### Authentication Service

- `POST /auth/validate` - Validate JWT token
- `POST /auth/refresh` - Refresh access token
- `POST /auth/logout` - Revoke token

### Deep Link Service

- `POST /deeplink/generate` - Generate authentication deep link with QR code
- `POST /deeplink/validate` - Validate deep link
- `GET /deeplink/validate/{link}` - Validate deep link via GET

### Webhook Service

- `POST /webhook/telegram` - Handle Telegram bot updates
- `POST /webhook/verify` - Verify webhook configuration

### Health Check

- `GET /health` - Service health status

## Configuration

The service is configured through environment variables. Copy `.env.example` to `.env` and update the values:

```bash
cp .env.example .env
```

### Required Configuration

- **`TELEGRAM_BOT_TOKEN`**: Your Telegram bot token from @BotFather
- **`JWT_SECRET`**: Secret key for signing JWT tokens
- **`MYSQL_DSN`**: MySQL connection string

### Optional Configuration

- **`PORT`**: Server port (default: 8080)
- **`ENV`**: Environment (default: development)
- **`DEEPLINK_BASE_URL`**: Base URL for deep links (default: https://example.com/auth)
- **`LOG_LEVEL`**: Logging level (debug, info, warn, error)
- **`WEBHOOK_URL`**: Webhook endpoint URL

## Bootstrap Instructions

### Prerequisites

- Go 1.22.2 or later
- MySQL 8.0 or later

### Local Development

1. **Clone and setup**

```bash
# Navigate to service directory
cd services/telegram_auth

# Copy environment configuration
cp .env.example .env

# Edit .env with your values
vim .env
```

2. **Install dependencies**

```bash
go mod tidy
```

3. **Run the service**

```bash
# Build and run
go build -o server cmd/server/main.go
./server

# Or run directly
go run cmd/server/main.go
```

4. **Test the service**

```bash
# Health check
curl http://localhost:8080/health

# Test token validation (requires valid JWT token)
curl -X POST http://localhost:8080/auth/validate \\
  -H "Content-Type: application/json" \\
  -d '{"token": "your-jwt-token"}'
```

### Database Setup

1. **Create MySQL database**

```sql
CREATE DATABASE telegram_auth;
CREATE USER 'telegram_user'@'%' IDENTIFIED BY 'telegram_password';
GRANT ALL PRIVILEGES ON telegram_auth.* TO 'telegram_user'@'%';
FLUSH PRIVILEGES;
```

2. **Update connection string in `.env`**

```
MYSQL_DSN=telegram_user:telegram_password@tcp(localhost:3306)/telegram_auth
```

## Docker Deployment

### Using Docker Compose

The service includes Docker support with MySQL integration:

```bash
# From project root
docker-compose up -d --build
```

This will start:
- **telegram_auth** service on port 8080
- **mysql** database on port 3306
- **whatsapp_go** service on port 3000 (existing service)

### Manual Docker Build

```bash
# Build the image
docker build -t telegram-auth:latest .

# Run the container
docker run -d \\
  --name telegram-auth \\
  -p 8080:8080 \\
  --env-file .env \\
  telegram-auth:latest
```

### Docker Environment

Create a `.env` file in the service directory with your configuration:

```bash
# Service Configuration
PORT=8080
ENV=production
LOG_LEVEL=info

# Required Configuration
TELEGRAM_BOT_TOKEN=your_bot_token_here
JWT_SECRET=your_jwt_secret_here
MYSQL_DSN=telegram_user:telegram_password@tcp(mysql:3306)/telegram_auth

# Optional Configuration
DEEPLINK_BASE_URL=https://yourapp.example.com/auth
WEBHOOK_URL=https://yourapp.example.com/webhook
WEBHOOK_SECRET=your_webhook_secret
```

## Usage Examples

### Generate Deep Link

```bash
curl -X POST http://localhost:8080/deeplink/generate \\
  -H "Content-Type: application/json" \\
  -d '{
    "user_id": "user123",
    "redirect_url": "https://yourapp.example.com/redirect"
  }'
```

Response:
```json
{
  "link": "https://yourapp.example.com/auth/telegram/auth?token=abc123&redirect=https://yourapp.example.com/redirect",
  "qr_code": "iVBORw0KGgoAAAANSUhEUgAA..."
}
```

### Validate Token

```bash
curl -X POST http://localhost:8080/auth/validate \\
  -H "Content-Type: application/json" \\
  -d '{
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }'
```

Response:
```json
{
  "valid": true,
  "user_id": "user123",
  "expires_at": "2024-12-15T10:30:00Z"
}
```

### Handle Telegram Webhook

```bash
curl -X POST http://localhost:8080/webhook/telegram \\
  -H "Content-Type: application/json" \\
  -d '{
    "update_id": 12345,
    "message": {
      "message_id": 678,
      "chat": {
        "id": 123456789,
        "type": "private"
      },
      "text": "Hello from Telegram!"
    }
  }'
```

Response:
```json
{
  "success": true,
  "response": "Received your message: Hello from Telegram!"
}
```

## Development

### Adding New Endpoints

1. **Define service interface** in `internal/{service}/service.go`
2. **Implement service** in `internal/{service}/service_impl.go`
3. **Create HTTP controller** in `gen/http/{service}/server/http_server.go`
4. **Add routes** in `cmd/server/main.go`

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/auth
```

### Code Style

```bash
# Format code
go fmt ./...

# Vet code
go vet ./...

# Import organization
goimports -w .
```

## Monitoring

### Health Checks

The service provides a health check endpoint:

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "ok",
  "service": "telegram-auth",
  "timestamp": "2024-12-14T02:00:00Z"
}
```

### Logging

Logs are structured in JSON format and include:
- Request IDs for tracing
- Request/response logging
- Error tracking with context
- Performance metrics

### Metrics

Monitor these key metrics:
- Request rate and latency
- Error rates by endpoint
- Token validation success/failure rates
- Database connection health

## Troubleshooting

### Common Issues

1. **Port already in use**
   ```bash
   # Change PORT in .env or kill process using port 8080
   lsof -i :8080
   kill -9 <PID>
   ```

2. **MySQL connection failed**
   ```bash
   # Check MySQL is running and accessible
   mysql -u telegram_user -p telegram_password -h localhost -e "SELECT 1;"
   ```

3. **Invalid Telegram bot token**
   ```bash
   # Test token with Telegram API
   curl "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getMe"
   ```

### Debug Mode

Enable debug logging:
```bash
LOG_LEVEL=debug go run cmd/server/main.go
```

### Production Deployment

1. Set `ENV=production`
2. Use secure `JWT_SECRET` (32+ characters)
3. Enable HTTPS (configure reverse proxy)
4. Set up monitoring and alerting
5. Configure log aggregation
6. Set appropriate resource limits

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Ensure tests pass: `go test ./...`
6. Submit a pull request

## License

This service is part of the Go WhatsApp Multidevice API project. See the main project LICENSE file for details.