package postgres

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/apikey"
    "github.com/lib/pq"
)

type APIKeyRepository struct {
    db *sql.DB
}

func NewAPIKeyRepository(db *sql.DB) *APIKeyRepository {
    return &APIKeyRepository{
        db: db,
    }
}

func (r *APIKeyRepository) Create(ctx context.Context, key *apikey.APIKey) error {
    query := `
        INSERT INTO api_keys (id, tenant_id, name, key_hash, key_prefix, scopes, status, expires_at, last_used_at, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
        RETURNING id, created_at, updated_at
    `

    err := r.db.QueryRowContext(ctx, query,
        key.ID, key.TenantID, key.Name, key.KeyHash, key.KeyPrefix, pq.Array(key.Scopes), key.Status, key.ExpiresAt, key.LastUsedAt,
    ).Scan(&key.ID, &key.CreatedAt, &key.UpdatedAt)

    if err != nil {
        if pqErr, ok := err.(*pq.Error); ok {
            if pqErr.Code == "23505" {
                return fmt.Errorf("API key with this hash already exists")
            }
            if pqErr.Code == "23503" {
                return fmt.Errorf("tenant not found")
            }
        }
        return fmt.Errorf("failed to create API key: %w", err)
    }

    return nil
}

func (r *APIKeyRepository) GetByID(ctx context.Context, id string) (*apikey.APIKey, error) {
    query := `
        SELECT id, tenant_id, name, key_hash, key_prefix, scopes, status, expires_at, last_used_at, created_at, updated_at
        FROM api_keys
        WHERE id = $1
    `

    var key apikey.APIKey
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &key.ID, &key.TenantID, &key.Name, &key.KeyHash, &key.KeyPrefix, pq.Array(&key.Scopes), &key.Status, &key.ExpiresAt, &key.LastUsedAt, &key.CreatedAt, &key.UpdatedAt,
    )

    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("API key not found")
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get API key: %w", err)
    }

    return &key, nil
}

func (r *APIKeyRepository) GetByKeyHash(ctx context.Context, keyHash string) (*apikey.APIKey, error) {
    query := `
        SELECT id, tenant_id, name, key_hash, key_prefix, scopes, status, expires_at, last_used_at, created_at, updated_at
        FROM api_keys
        WHERE key_hash = $1
    `

    var key apikey.APIKey
    err := r.db.QueryRowContext(ctx, query, keyHash).Scan(
        &key.ID, &key.TenantID, &key.Name, &key.KeyHash, &key.KeyPrefix, pq.Array(&key.Scopes), &key.Status, &key.ExpiresAt, &key.LastUsedAt, &key.CreatedAt, &key.UpdatedAt,
    )

    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("API key not found")
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get API key: %w", err)
    }

    return &key, nil
}

func (r *APIKeyRepository) ListByTenantID(ctx context.Context, tenantID string, limit, offset int) ([]*apikey.APIKey, error) {
    query := `
        SELECT id, tenant_id, name, key_hash, key_prefix, scopes, status, expires_at, last_used_at, created_at, updated_at
        FROM api_keys
        WHERE tenant_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `

    rows, err := r.db.QueryContext(ctx, query, tenantID, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to list API keys: %w", err)
    }
    defer rows.Close()

    var keys []*apikey.APIKey
    for rows.Next() {
        var key apikey.APIKey
        err := rows.Scan(
            &key.ID, &key.TenantID, &key.Name, &key.KeyHash, &key.KeyPrefix, pq.Array(&key.Scopes), &key.Status, &key.ExpiresAt, &key.LastUsedAt, &key.CreatedAt, &key.UpdatedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan API key: %w", err)
        }
        keys = append(keys, &key)
    }

    return keys, nil
}

func (r *APIKeyRepository) Update(ctx context.Context, id string, req *apikey.UpdateAPIKeyRequest) error {
    query := "UPDATE api_keys SET "
    args := []interface{}{}
    argIdx := 1

    if req.Name != nil {
        query += fmt.Sprintf("name = $%d, ", argIdx)
        args = append(args, *req.Name)
        argIdx++
    }

    if req.Status != nil {
        query += fmt.Sprintf("status = $%d, ", argIdx)
        args = append(args, *req.Status)
        argIdx++
    }

    if req.ExpiresAt != nil {
        query += fmt.Sprintf("expires_at = $%d, ", argIdx)
        args = append(args, *req.ExpiresAt)
        argIdx++
    }

    if argIdx == 1 {
        return fmt.Errorf("no fields to update")
    }

    query += fmt.Sprintf("updated_at = NOW() WHERE id = $%d", argIdx)
    args = append(args, id)

    result, err := r.db.ExecContext(ctx, query, args...)
    if err != nil {
        return fmt.Errorf("failed to update API key: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }

    if rowsAffected == 0 {
        return fmt.Errorf("API key not found")
    }

    return nil
}

func (r *APIKeyRepository) Delete(ctx context.Context, id string) error {
    query := "DELETE FROM api_keys WHERE id = $1"

    result, err := r.db.ExecContext(ctx, query, id)
    if err != nil {
        return fmt.Errorf("failed to delete API key: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }

    if rowsAffected == 0 {
        return fmt.Errorf("API key not found")
    }

    return nil
}

func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id string) error {
    query := "UPDATE api_keys SET last_used_at = NOW(), updated_at = NOW() WHERE id = $1"

    _, err := r.db.ExecContext(ctx, query, id)
    if err != nil {
        return fmt.Errorf("failed to update last used: %w", err)
    }

    return nil
}

func (r *APIKeyRepository) IncrementRateLimit(ctx context.Context, req *apikey.IncrementRateLimitRequest) (currentCount int, exceeded bool, err error) {
    incrementBy := 1
    if req.IncrementBy > 0 {
        incrementBy = req.IncrementBy
    }

    windowStart := time.Now().Truncate(time.Duration(req.WindowDurationSeconds) * time.Second)
    if req.WindowStart != nil {
        windowStart = *req.WindowStart
    }

    query := `
        INSERT INTO rate_limit_counters (tenant_id, api_key_id, resource_type, window_start, window_duration_seconds, request_count, limit_value, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
        ON CONFLICT (tenant_id, api_key_id, resource_type, window_start)
        DO UPDATE SET request_count = rate_limit_counters.request_count + $6, limit_value = $7, updated_at = NOW()
        RETURNING request_count, limit_value
    `

    var limitValue int
    err = r.db.QueryRowContext(ctx, query,
        req.TenantID, req.APIKeyID, req.ResourceType, windowStart, req.WindowDurationSeconds, incrementBy, req.LimitValue,
    ).Scan(&currentCount, &limitValue)

    if err != nil {
        return 0, false, fmt.Errorf("failed to increment rate limit: %w", err)
    }

    exceeded = currentCount > limitValue
    return currentCount, exceeded, nil
}

func (r *APIKeyRepository) GetRateLimitCounter(ctx context.Context, tenantID string, apiKeyID *string, resourceType string, windowStart time.Time) (*apikey.RateLimitCounter, error) {
    query := `
        SELECT id, tenant_id, api_key_id, resource_type, window_start, window_duration_seconds, request_count, limit_value, created_at, updated_at
        FROM rate_limit_counters
        WHERE tenant_id = $1 AND resource_type = $2 AND window_start = $3
    `
    args := []interface{}{tenantID, resourceType, windowStart}

    if apiKeyID != nil {
        query += " AND api_key_id = $4"
        args = append(args, *apiKeyID)
    } else {
        query += " AND api_key_id IS NULL"
    }

    var counter apikey.RateLimitCounter
    err := r.db.QueryRowContext(ctx, query, args...).Scan(
        &counter.ID, &counter.TenantID, &counter.APIKeyID, &counter.ResourceType, &counter.WindowStart, &counter.WindowDurationSeconds, &counter.RequestCount, &counter.LimitValue, &counter.CreatedAt, &counter.UpdatedAt,
    )

    if err == sql.ErrNoRows {
        return nil, apikey.ErrRateLimitCounterNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get rate limit counter: %w", err)
    }

    return &counter, nil
}

func (r *APIKeyRepository) CleanupExpiredCounters(ctx context.Context, before time.Time) error {
    query := "DELETE FROM rate_limit_counters WHERE window_start < $1"

    result, err := r.db.ExecContext(ctx, query, before)
    if err != nil {
        return fmt.Errorf("failed to cleanup expired counters: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }

    if rowsAffected > 0 {
        fmt.Printf("Cleaned up %d expired rate limit counters\n", rowsAffected)
    }

    return nil
}
