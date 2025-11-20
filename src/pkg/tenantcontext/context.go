package tenantcontext

import (
    "context"

    domainTenant "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/tenant"
    pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
)

type contextKey struct{}

// WithTenant injects the authenticated tenant context into the provided context so downstream usecases
// can remain transport-agnostic.
func WithTenant(ctx context.Context, tenantCtx *domainTenant.TenantContext) context.Context {
    if ctx == nil {
        ctx = context.Background()
    }
    return context.WithValue(ctx, contextKey{}, tenantCtx)
}

// ContextProvider implements domainTenant.TenantProvider by pulling the tenant context out of a standard
// Go context. This keeps usecases decoupled from Fiber while still ensuring we reject calls that somehow
// bypass the API key middleware.
type ContextProvider struct{}

func NewContextProvider() domainTenant.TenantProvider {
    return &ContextProvider{}
}

func (p *ContextProvider) CurrentTenant(ctx context.Context) (*domainTenant.TenantContext, error) {
    if ctx == nil {
        return nil, pkgError.ContextError("missing request context")
    }

    value := ctx.Value(contextKey{})
    tenantCtx, ok := value.(*domainTenant.TenantContext)
    if !ok || tenantCtx == nil {
        return nil, pkgError.AuthError("tenant context is not set")
    }

    return tenantCtx, nil
}
