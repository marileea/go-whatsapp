package apikey

import (
	"context"
	"time"
)

type IAPIKeyRepository interface {
	Create(ctx context.Context, key *APIKey) error
	GetByID(ctx context.Context, id string) (*APIKey, error)
	GetByKeyHash(ctx context.Context, keyHash string) (*APIKey, error)
	ListByTenantID(ctx context.Context, tenantID string, limit, offset int) ([]*APIKey, error)
	Update(ctx context.Context, id string, req *UpdateAPIKeyRequest) error
	Delete(ctx context.Context, id string) error
	UpdateLastUsed(ctx context.Context, id string) error

	IncrementRateLimit(ctx context.Context, req *IncrementRateLimitRequest) (currentCount int, exceeded bool, err error)
	GetRateLimitCounter(ctx context.Context, tenantID string, apiKeyID *string, resourceType string, windowStart time.Time) (*RateLimitCounter, error)
	CleanupExpiredCounters(ctx context.Context, before time.Time) error
}
