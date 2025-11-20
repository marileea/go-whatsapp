package middleware

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	domainAPIKey "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/apikey"
	domainServer "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/server"
	domainTenant "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/tenant"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/apikeyutil"
	auditlogger "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/audit"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/ratelimiter"
	tenantcontext "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/tenantcontext"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyAuth_AllowsRequestWithValidKey(t *testing.T) {
	salt := "unit-test-salt"
	rawKey := "sk_live_unit_test_key"
	hashed, err := apikeyutil.HashKey(rawKey, salt)
	require.NoError(t, err)

	repo := &fakeAPIKeyRepo{
		key: &domainAPIKey.APIKey{
			ID:        "api-key-id",
			TenantID:  "tenant-id",
			Name:      "Primary",
			KeyHash:   hashed,
			KeyPrefix: "sk_live",
			Scopes:    []string{"messages:send"},
			Status:    domainAPIKey.APIKeyStatusActive,
		},
		expectedHash: hashed,
	}

	tenantRepo := &fakeTenantRepo{
		tenant: &domainTenant.Tenant{
			ID:       "tenant-id",
			Name:     "Acme",
			Status:   domainTenant.TenantStatusActive,
			Metadata: map[string]interface{}{"rate_limit_per_minute": float64(120)},
		},
	}

	serverRepo := &fakeServerRepo{
		assignments: []*domainServer.TenantServerAssignment{
			{
				ID:           "assign-1",
				TenantID:     "tenant-id",
				ServerNodeID: "server-node",
				Status:       domainServer.AssignmentStatusActive,
			},
		},
	}

	limiter := &fakeLimiter{result: &ratelimiter.AllowResult{Allowed: true, Remaining: 10}}
	auditLog := &fakeAuditLogger{}

	cfg := APIKeyConfig{
		APIKeyRepo:       repo,
		TenantRepo:       tenantRepo,
		ServerRepo:       serverRepo,
		RateLimiter:      limiter,
		AuditLogger:      auditLog,
		ServerNodeID:     "server-node",
		DefaultRateLimit: 60,
		HashSalt:         salt,
	}

	app := fiber.New()
	app.Use(APIKeyAuth(cfg))
	app.Get("/app/status", func(c *fiber.Ctx) error {
		provider := tenantcontext.NewContextProvider()
		tenantCtx, err := provider.CurrentTenant(c.UserContext())
		require.NoError(t, err)
		require.Equal(t, "tenant-id", tenantCtx.TenantID)
		require.Equal(t, "api-key-id", tenantCtx.APIKeyID)
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/app/status", nil)
	req.Header.Set("X-API-Key", rawKey)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	require.Eventually(t, func() bool {
		return len(auditLog.entries) == 1
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, fiber.StatusOK, auditLog.entries[0].StatusCode)
	require.Equal(t, "assign-1", auditLog.entries[0].ServerAssignmentID)
}

func TestAPIKeyAuth_RevokedKey(t *testing.T) {
	salt := "unit-test-salt"
	rawKey := "sk_live_revoked"
	hashed, err := apikeyutil.HashKey(rawKey, salt)
	require.NoError(t, err)

	repo := &fakeAPIKeyRepo{
		key: &domainAPIKey.APIKey{
			ID:        "api-key-id",
			TenantID:  "tenant-id",
			KeyHash:   hashed,
			KeyPrefix: "sk_live",
			Status:    domainAPIKey.APIKeyStatusRevoked,
		},
		expectedHash: hashed,
	}
	cfg := APIKeyConfig{
		APIKeyRepo:       repo,
		TenantRepo:       &fakeTenantRepo{},
		ServerRepo:       &fakeServerRepo{},
		RateLimiter:      &fakeLimiter{result: &ratelimiter.AllowResult{Allowed: true}},
		AuditLogger:      &fakeAuditLogger{},
		ServerNodeID:     "server-node",
		DefaultRateLimit: 60,
		HashSalt:         salt,
	}

	app := fiber.New()
	app.Use(APIKeyAuth(cfg))
	app.Get("/app/status", func(c *fiber.Ctx) error {
		t.Fatalf("handler should not be invoked for revoked key")
		return nil
	})

	req := httptest.NewRequest("GET", "/app/status", nil)
	req.Header.Set("X-API-Key", rawKey)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestAPIKeyAuth_RateLimited(t *testing.T) {
	salt := "unit-test-salt"
	rawKey := "sk_live_limit"
	hashed, err := apikeyutil.HashKey(rawKey, salt)
	require.NoError(t, err)

	repo := &fakeAPIKeyRepo{
		key: &domainAPIKey.APIKey{
			ID:        "api-key-id",
			TenantID:  "tenant-id",
			KeyHash:   hashed,
			KeyPrefix: "sk_live",
			Status:    domainAPIKey.APIKeyStatusActive,
		},
		expectedHash: hashed,
	}
	tenantRepo := &fakeTenantRepo{
		tenant: &domainTenant.Tenant{
			ID:       "tenant-id",
			Status:   domainTenant.TenantStatusActive,
			Metadata: map[string]interface{}{},
		},
	}
	serverRepo := &fakeServerRepo{
		assignments: []*domainServer.TenantServerAssignment{
			{
				ID:           "assign-1",
				TenantID:     "tenant-id",
				ServerNodeID: "server-node",
				Status:       domainServer.AssignmentStatusActive,
			},
		},
	}

	limiter := &fakeLimiter{result: &ratelimiter.AllowResult{Allowed: false, RetryAfter: 3 * time.Second}}
	auditLog := &fakeAuditLogger{}

	cfg := APIKeyConfig{
		APIKeyRepo:       repo,
		TenantRepo:       tenantRepo,
		ServerRepo:       serverRepo,
		RateLimiter:      limiter,
		AuditLogger:      auditLog,
		ServerNodeID:     "server-node",
		DefaultRateLimit: 60,
		HashSalt:         salt,
	}

	app := fiber.New()
	app.Use(APIKeyAuth(cfg))
	app.Get("/app/status", func(c *fiber.Ctx) error {
		t.Fatalf("handler should not be invoked when rate limited")
		return nil
	})

	req := httptest.NewRequest("GET", "/app/status", nil)
	req.Header.Set("X-API-Key", rawKey)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode)
	require.Equal(t, "3", resp.Header.Get("Retry-After"))

	require.Eventually(t, func() bool {
		return len(auditLog.entries) == 1
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, fiber.StatusTooManyRequests, auditLog.entries[0].StatusCode)
}

// --- fakes ---

type fakeAPIKeyRepo struct {
	key          *domainAPIKey.APIKey
	expectedHash string
}

func (f *fakeAPIKeyRepo) Create(context.Context, *domainAPIKey.APIKey) error { return nil }
func (f *fakeAPIKeyRepo) GetByID(context.Context, string) (*domainAPIKey.APIKey, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeAPIKeyRepo) GetByKeyHash(ctx context.Context, hash string) (*domainAPIKey.APIKey, error) {
	if f.expectedHash != "" && hash != f.expectedHash {
		return nil, errors.New("not found")
	}
	return f.key, nil
}
func (f *fakeAPIKeyRepo) ListByTenantID(context.Context, string, int, int) ([]*domainAPIKey.APIKey, error) {
	return nil, nil
}
func (f *fakeAPIKeyRepo) Update(context.Context, string, *domainAPIKey.UpdateAPIKeyRequest) error {
	return nil
}
func (f *fakeAPIKeyRepo) Delete(context.Context, string) error         { return nil }
func (f *fakeAPIKeyRepo) UpdateLastUsed(context.Context, string) error { return nil }
func (f *fakeAPIKeyRepo) IncrementRateLimit(context.Context, *domainAPIKey.IncrementRateLimitRequest) (int, bool, error) {
	return 0, false, nil
}
func (f *fakeAPIKeyRepo) GetRateLimitCounter(context.Context, string, *string, string, time.Time) (*domainAPIKey.RateLimitCounter, error) {
	return nil, domainAPIKey.ErrRateLimitCounterNotFound
}
func (f *fakeAPIKeyRepo) CleanupExpiredCounters(context.Context, time.Time) error { return nil }

type fakeTenantRepo struct {
	tenant *domainTenant.Tenant
}

func (f *fakeTenantRepo) Create(context.Context, *domainTenant.Tenant) error { return nil }
func (f *fakeTenantRepo) GetByID(context.Context, string) (*domainTenant.Tenant, error) {
	if f.tenant == nil {
		return nil, errors.New("tenant not found")
	}
	return f.tenant, nil
}
func (f *fakeTenantRepo) List(context.Context, int, int) ([]*domainTenant.Tenant, error) {
	return nil, nil
}
func (f *fakeTenantRepo) Update(context.Context, string, *domainTenant.UpdateTenantRequest) error {
	return nil
}
func (f *fakeTenantRepo) Delete(context.Context, string) error         { return nil }
func (f *fakeTenantRepo) GetTotalCount(context.Context) (int64, error) { return 0, nil }
func (f *fakeTenantRepo) CreateContact(context.Context, *domainTenant.TenantContact) error {
	return nil
}
func (f *fakeTenantRepo) GetContactsByTenantID(context.Context, string) ([]*domainTenant.TenantContact, error) {
	return nil, nil
}
func (f *fakeTenantRepo) UpdateContact(context.Context, string, *domainTenant.TenantContact) error {
	return nil
}
func (f *fakeTenantRepo) DeleteContact(context.Context, string) error { return nil }

type fakeServerRepo struct {
	assignments []*domainServer.TenantServerAssignment
}

func (f *fakeServerRepo) GetAssignmentsByTenantID(context.Context, string) ([]*domainServer.TenantServerAssignment, error) {
	if f.assignments == nil {
		return nil, errors.New("no assignments")
	}
	return f.assignments, nil
}

type fakeLimiter struct {
	result *ratelimiter.AllowResult
	err    error
}

func (f *fakeLimiter) Allow(context.Context, *ratelimiter.AllowRequest) (*ratelimiter.AllowResult, error) {
	return f.result, f.err
}

func (f *fakeLimiter) Shutdown(context.Context) error { return nil }

type fakeAuditLogger struct {
	entries []auditlogger.RequestEntry
}

func (f *fakeAuditLogger) Record(entry auditlogger.RequestEntry) {
	f.entries = append(f.entries, entry)
}

func (f *fakeAuditLogger) Close(context.Context) error { return nil }
