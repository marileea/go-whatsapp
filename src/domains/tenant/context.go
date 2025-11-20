package tenant

import "context"

// TenantContext carries the authenticated tenant metadata extracted from the API key middleware.
type TenantContext struct {
    TenantID          string
    APIKeyID          string
    ServerAssignmentID string
    Scopes            []string
    RateLimitPerMinute int
}

// TenantProvider resolves the tenant context from the request context so usecases can remain tenant-aware
// without being coupled to the HTTP layer.
type TenantProvider interface {
    CurrentTenant(ctx context.Context) (*TenantContext, error)
}
