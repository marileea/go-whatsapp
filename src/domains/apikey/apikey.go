package apikey

import (
    "context"
    "errors"
    "time"
)

type APIKeyStatus string

const (
    APIKeyStatusActive  APIKeyStatus = "active"
    APIKeyStatusRevoked APIKeyStatus = "revoked"
    APIKeyStatusExpired APIKeyStatus = "expired"
)

var ErrRateLimitCounterNotFound = errors.New("rate limit counter not found")

// IAPIKeyUsecase describes the public contract for API key management capabilities that the
// REST layer can rely on when creating, rotating, disabling, or inspecting API keys.
type IAPIKeyUsecase interface {
    CreateKey(ctx context.Context, req *CreateAPIKeyRequest) (*CreateAPIKeyResponse, error)
    RotateKey(ctx context.Context, apiKeyID string) (*CreateAPIKeyResponse, error)
    DisableKey(ctx context.Context, apiKeyID string) error
    DescribeKey(ctx context.Context, apiKeyID string) (*APIKey, error)
    ListKeys(ctx context.Context, tenantID string, limit, offset int) ([]*APIKey, error)
}

type APIKey struct {
    ID         string       `json:"id" db:"id"`
    TenantID   string       `json:"tenant_id" db:"tenant_id"`
    Name       string       `json:"name" db:"name"`
    KeyHash    string       `json:"-" db:"key_hash"`
    KeyPrefix  string       `json:"key_prefix" db:"key_prefix"`
    Scopes     []string     `json:"scopes" db:"scopes"`
    Status     APIKeyStatus `json:"status" db:"status"`
    ExpiresAt  *time.Time   `json:"expires_at" db:"expires_at"`
    LastUsedAt *time.Time   `json:"last_used_at" db:"last_used_at"`
    CreatedAt  time.Time    `json:"created_at" db:"created_at"`
    UpdatedAt  time.Time    `json:"updated_at" db:"updated_at"`
}

type RateLimitCounter struct {
    ID                    string    `json:"id" db:"id"`
    TenantID              string    `json:"tenant_id" db:"tenant_id"`
    APIKeyID              *string   `json:"api_key_id" db:"api_key_id"`
    ResourceType          string    `json:"resource_type" db:"resource_type"`
    WindowStart           time.Time `json:"window_start" db:"window_start"`
    WindowDurationSeconds int       `json:"window_duration_seconds" db:"window_duration_seconds"`
    RequestCount          int       `json:"request_count" db:"request_count"`
    LimitValue            int       `json:"limit_value" db:"limit_value"`
    CreatedAt             time.Time `json:"created_at" db:"created_at"`
    UpdatedAt             time.Time `json:"updated_at" db:"updated_at"`
}

type CreateAPIKeyRequest struct {
    TenantID  string     `json:"tenant_id" validate:"required,uuid"`
    Name      string     `json:"name" validate:"required"`
    Scopes    []string   `json:"scopes" validate:"required,min=1"`
    ExpiresAt *time.Time `json:"expires_at"`
}

type CreateAPIKeyResponse struct {
    APIKey   *APIKey `json:"api_key"`
    PlainKey string  `json:"plain_key"`
}

type UpdateAPIKeyRequest struct {
    Name      *string       `json:"name"`
    Status    *APIKeyStatus `json:"status"`
    ExpiresAt *time.Time    `json:"expires_at"`
}

type IncrementRateLimitRequest struct {
    TenantID              string     `json:"tenant_id" validate:"required,uuid"`
    APIKeyID              *string    `json:"api_key_id"`
    ResourceType          string     `json:"resource_type" validate:"required"`
    WindowDurationSeconds int        `json:"window_duration_seconds" validate:"required,min=1"`
    LimitValue            int        `json:"limit_value" validate:"required,min=1"`
    IncrementBy           int        `json:"increment_by"`
    WindowStart           *time.Time `json:"window_start"`
}
