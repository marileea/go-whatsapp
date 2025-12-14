package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds the service configuration
type Config struct {
	// Server configuration
	Port         int           `envconfig:"PORT" default:"8080"`
	Env          string        `envconfig:"ENV" default:"development"`
	ShutdownTime time.Duration `envconfig:"SHUTDOWN_TIME" default:"30s"`

	// Telegram configuration
	TelegramBotToken string `envconfig:"TELEGRAM_BOT_TOKEN" required:"true"`

	// Deep link configuration
	DeepLinkBaseURL string `envconfig:"DEEPLINK_BASE_URL" default:"https://example.com/auth"`

	// JWT configuration
	JWTSecret    string        `envconfig:"JWT_SECRET" required:"true"`
	TokenTTL     time.Duration `envconfig:"TOKEN_TTL" default:"24h"`
	RefreshTTL   time.Duration `envconfig:"REFRESH_TOKEN_TTL" default:"168h"`
	RefreshRatio int           `envconfig:"REFRESH_RATIO" default:"3"`

	// Database configuration
	MySQLDSN string `envconfig:"MYSQL_DSN" required:"true"`

	// Logging configuration
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`

	// Webhook configuration
	WebhookURL    string        `envconfig:"WEBHOOK_URL"`
	WebhookSecret string        `envconfig:"WEBHOOK_SECRET"`
	WebhookTimeout time.Duration `envconfig:"WEBHOOK_TIMEOUT" default:"30s"`
}

// Load loads configuration from environment variables and optionally from a config file
func Load() (*Config, error) {
	var cfg Config
	envPrefix := "TELEGRAM_AUTH"

	// Load from environment variables
	if err := envconfig.Process(envPrefix, &cfg); err != nil {
		return nil, err
	}

	// Validate required fields
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validateConfig validates the required configuration fields
func validateConfig(cfg *Config) error {
	if cfg.TelegramBotToken == "" {
		return Error("TELEGRAM_BOT_TOKEN", "telegram bot token is required")
	}

	if cfg.JWTSecret == "" {
		return Error("JWT_SECRET", "jwt secret is required")
	}

	if cfg.MySQLDSN == "" {
		return Error("MYSQL_DSN", "mysql dsn is required")
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		return Error("PORT", "port must be between 1 and 65535")
	}

	if cfg.TokenTTL < time.Minute {
		return Error("TOKEN_TTL", "token ttl must be at least 1 minute")
	}

	if cfg.RefreshTTL < time.Hour {
		return Error("REFRESH_TOKEN_TTL", "refresh token ttl must be at least 1 hour")
	}

	return nil
}

// Error creates a configuration error
func Error(field, message string) error {
	return &ConfigError{
		Field:   field,
		Message: message,
	}
}

// ConfigError represents a configuration validation error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return "config error: " + e.Field + " - " + e.Message
}

// IsConfigError checks if an error is a configuration error
func IsConfigError(err error) bool {
	_, ok := err.(*ConfigError)
	return ok
}

// GetEnvOrDefault returns environment variable value or default
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvAsInt returns environment variable as integer or default
func GetEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetEnvAsDuration returns environment variable as duration or default
func GetEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// GetEnvAsBool returns environment variable as boolean or default
func GetEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultValue
}