package middleware

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	domainAPIKey "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/apikey"
	domainServer "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/server"
	domainTenant "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/tenant"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/apikeyutil"
	auditlogger "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/audit"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/ratelimiter"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/tenantcontext"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

const (
	headerAPIKey        = "X-API-Key"
	headerAuthorization = "Authorization"
	bearerPrefix        = "bearer "
	resourceREST        = ratelimiter.ResourceREST
	auditResourceREST   = "rest_api"
	auditActionPrefix   = "REST"
)

type apiKeyRepository interface {
	GetByKeyHash(ctx context.Context, keyHash string) (*domainAPIKey.APIKey, error)
	UpdateLastUsed(ctx context.Context, id string) error
}

type tenantRepository interface {
	GetByID(ctx context.Context, id string) (*domainTenant.Tenant, error)
}

type serverAssignmentRepository interface {
	GetAssignmentsByTenantID(ctx context.Context, tenantID string) ([]*domainServer.TenantServerAssignment, error)
}

type APIKeyConfig struct {
	APIKeyRepo       apiKeyRepository
	TenantRepo       tenantRepository
	ServerRepo       serverAssignmentRepository
	RateLimiter      ratelimiter.RateLimiter
	AuditLogger      auditlogger.Logger
	ServerNodeID     string
	DefaultRateLimit int
	HashSalt         string
}

func APIKeyAuth(cfg APIKeyConfig) fiber.Handler {
	if cfg.APIKeyRepo == nil || cfg.TenantRepo == nil || cfg.ServerRepo == nil || cfg.RateLimiter == nil || cfg.AuditLogger == nil {
		panic("api key middleware: repositories, rate limiter, and audit logger are required")
	}
	if cfg.HashSalt == "" {
		panic("api key middleware: hashing salt is required")
	}
	if cfg.ServerNodeID == "" {
		panic("api key middleware: server node id is required")
	}
	if cfg.DefaultRateLimit <= 0 {
		cfg.DefaultRateLimit = 60
	}

	return func(c *fiber.Ctx) error {
		rawKey := extractAPIKey(c)
		if rawKey == "" {
			return respondWithError(c, fiber.StatusUnauthorized, "API_KEY_REQUIRED", "API key is required")
		}

		ctx := c.UserContext()
		if ctx == nil {
			ctx = context.Background()
		}
		hashedKey, err := apikeyutil.HashKey(rawKey, cfg.HashSalt)
		if err != nil {
			logrus.WithError(err).Error("failed to hash api key")
			return respondWithError(c, fiber.StatusInternalServerError, "API_KEY_INVALID", "API key validation failed")
		}

		apiKey, err := cfg.APIKeyRepo.GetByKeyHash(ctx, hashedKey)
		if err != nil {
			return respondWithError(c, fiber.StatusUnauthorized, "API_KEY_INVALID", "API key is invalid")
		}
		if apiKey.Status != domainAPIKey.APIKeyStatusActive {
			cfg.logFailure(apiKey.ID, apiKey.TenantID, "", c, fiber.StatusForbidden, fmt.Errorf("api key is not active"))
			return respondWithError(c, fiber.StatusForbidden, "API_KEY_REVOKED", "API key is not active")
		}
		if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
			cfg.logFailure(apiKey.ID, apiKey.TenantID, "", c, fiber.StatusUnauthorized, fmt.Errorf("api key expired"))
			return respondWithError(c, fiber.StatusUnauthorized, "API_KEY_EXPIRED", "API key has expired")
		}

		tenant, err := cfg.TenantRepo.GetByID(ctx, apiKey.TenantID)
		if err != nil {
			logrus.WithError(err).Error("failed to load tenant for api key")
			cfg.logFailure(apiKey.ID, apiKey.TenantID, "", c, fiber.StatusForbidden, fmt.Errorf("tenant lookup failed: %w", err))
			return respondWithError(c, fiber.StatusForbidden, "TENANT_NOT_FOUND", "Tenant is not authorized")
		}
		if tenant.Status != domainTenant.TenantStatusActive {
			cfg.logFailure(apiKey.ID, apiKey.TenantID, "", c, fiber.StatusForbidden, fmt.Errorf("tenant inactive"))
			return respondWithError(c, fiber.StatusForbidden, "TENANT_INACTIVE", "Tenant is not active")
		}

		assignment, err := cfg.resolveAssignment(ctx, tenant.ID)
		if err != nil {
			cfg.logFailure(apiKey.ID, tenant.ID, "", c, fiber.StatusForbidden, err)
			return respondWithError(c, fiber.StatusForbidden, "TENANT_NOT_ASSIGNED", err.Error())
		}
		assignmentID := assignment.ID

		limit := resolveTenantRateLimit(tenant, cfg.DefaultRateLimit)
		allowResult, err := cfg.RateLimiter.Allow(ctx, &ratelimiter.AllowRequest{
			TenantID:     tenant.ID,
			APIKeyID:     apiKey.ID,
			ResourceType: resourceREST,
			Limit:        limit,
		})
		if err != nil {
			logrus.WithError(err).Error("rate limiter failure")
			cfg.logFailure(apiKey.ID, tenant.ID, assignmentID, c, fiber.StatusInternalServerError, err)
			return respondWithError(c, fiber.StatusInternalServerError, "RATE_LIMIT_ERROR", "Unable to evaluate rate limit")
		}
		if !allowResult.Allowed {
			if allowResult.RetryAfter > 0 {
				c.Set(fiber.HeaderRetryAfter, fmt.Sprintf("%.0f", allowResult.RetryAfter.Seconds()))
			}
			cfg.logRequest(apiKey.ID, tenant.ID, assignmentID, c, fiber.StatusTooManyRequests, 0, fmt.Errorf("rate limit exceeded"))
			return respondWithError(c, fiber.StatusTooManyRequests, "RATE_LIMITED", "Too many requests")
		}

		tenantCtx := &domainTenant.TenantContext{
			TenantID:           tenant.ID,
			APIKeyID:           apiKey.ID,
			ServerAssignmentID: assignmentID,
			Scopes:             append([]string{}, apiKey.Scopes...),
			RateLimitPerMinute: limit,
		}
		ctxWithTenant := tenantcontext.WithTenant(ctx, tenantCtx)
		c.SetUserContext(ctxWithTenant)

		started := time.Now()
		if err := c.Next(); err != nil {
			cfg.logRequest(apiKey.ID, tenant.ID, assignmentID, c, c.Response().StatusCode(), time.Since(started), err)
			return err
		}

		statusCode := c.Response().StatusCode()
		if statusCode == 0 {
			statusCode = fiber.StatusOK
		}

		cfg.logRequest(apiKey.ID, tenant.ID, assignmentID, c, statusCode, time.Since(started), nil)

		go cfg.APIKeyRepo.UpdateLastUsed(context.Background(), apiKey.ID)

		return nil
	}
}

func extractAPIKey(c *fiber.Ctx) string {
	apiKey := strings.TrimSpace(string(c.Request().Header.Peek(headerAPIKey)))
	if apiKey != "" {
		return apiKey
	}

	authHeader := strings.TrimSpace(c.Get(headerAuthorization))
	if authHeader == "" {
		return ""
	}

	authLower := strings.ToLower(authHeader)
	if strings.HasPrefix(authLower, bearerPrefix) {
		return strings.TrimSpace(authHeader[len(bearerPrefix):])
	}

	return ""
}

func (cfg APIKeyConfig) resolveAssignment(ctx context.Context, tenantID string) (*domainServer.TenantServerAssignment, error) {
	assignments, err := cfg.ServerRepo.GetAssignmentsByTenantID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tenant assignment")
	}
	for _, assignment := range assignments {
		if assignment.ServerNodeID == cfg.ServerNodeID && assignment.Status == domainServer.AssignmentStatusActive {
			return assignment, nil
		}
	}
	return nil, fmt.Errorf("tenant is not assigned to this server")
}

func (cfg APIKeyConfig) logRequest(apiKeyID, tenantID, assignmentID string, c *fiber.Ctx, status int, latency time.Duration, err error) {
	entry := auditlogger.RequestEntry{
		TenantID:           tenantID,
		APIKeyID:           apiKeyID,
		ServerNodeID:       cfg.ServerNodeID,
		ServerAssignmentID: assignmentID,
		Method:             c.Method(),
		Path:               c.Path(),
		StatusCode:         status,
		Latency:            latency,
		IP:                 c.IP(),
		UserAgent:          c.Get(fiber.HeaderUserAgent),
		RequestID:          c.Get("X-Request-ID"),
		ResourceType:       auditResourceREST,
		Action:             fmt.Sprintf("%s %s", auditActionPrefix, c.Path()),
		Error:              err,
	}
	cfg.AuditLogger.Record(entry)
}

func (cfg APIKeyConfig) logFailure(apiKeyID, tenantID, assignmentID string, c *fiber.Ctx, status int, err error) {
	if apiKeyID == "" && tenantID == "" {
		return
	}
	cfg.logRequest(apiKeyID, tenantID, assignmentID, c, status, 0, err)
}

func respondWithError(c *fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(utils.ResponseData{
		Status:  status,
		Code:    code,
		Message: message,
	})
}

func resolveTenantRateLimit(tenant *domainTenant.Tenant, defaultLimit int) int {
	if defaultLimit <= 0 {
		defaultLimit = 60
	}
	if tenant == nil || tenant.Metadata == nil {
		return defaultLimit
	}

	if limit := parseInt(tenant.Metadata["rate_limit_per_minute"]); limit > 0 {
		return limit
	}

	if raw, ok := tenant.Metadata["rate_limit"].(map[string]interface{}); ok {
		if limit := parseInt(raw["requests_per_minute"]); limit > 0 {
			return limit
		}
	}

	return defaultLimit
}

func parseInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case string:
		val := strings.TrimSpace(v)
		if val == "" {
			return 0
		}
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return 0
}
