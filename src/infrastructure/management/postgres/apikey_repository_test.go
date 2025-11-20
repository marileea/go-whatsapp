package postgres

import (
    "context"
    "testing"
    "time"

    "github.com/DATA-DOG/go-sqlmock"
    "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/apikey"
    "github.com/lib/pq"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestAPIKeyRepository_Create(t *testing.T) {
    db, mock, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()

    repo := NewAPIKeyRepository(db)
    ctx := context.Background()
    now := time.Now()

    key := &apikey.APIKey{
        ID:        "key-id",
        TenantID:  "tenant-id",
        Name:      "Test Key",
        KeyHash:   "hash123",
        KeyPrefix: "sk_test",
        Scopes:    []string{"read", "write"},
        Status:    apikey.APIKeyStatusActive,
    }

    mock.ExpectQuery(`INSERT INTO api_keys`).
        WithArgs(key.ID, key.TenantID, key.Name, key.KeyHash, key.KeyPrefix, pq.Array(key.Scopes), key.Status, key.ExpiresAt, key.LastUsedAt).
        WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
            AddRow(key.ID, now, now))

    err = repo.Create(ctx, key)
    assert.NoError(t, err)
    assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAPIKeyRepository_GetByKeyHash(t *testing.T) {
    db, mock, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()

    repo := NewAPIKeyRepository(db)
    ctx := context.Background()
    now := time.Now()
    keyHash := "hash123"

    mock.ExpectQuery(`SELECT .+ FROM api_keys WHERE key_hash = \$1`).
        WithArgs(keyHash).
        WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name", "key_hash", "key_prefix", "scopes", "status", "expires_at", "last_used_at", "created_at", "updated_at"}).
            AddRow("key-id", "tenant-id", "Test Key", keyHash, "sk_test", pq.Array([]string{"read", "write"}), apikey.APIKeyStatusActive, nil, nil, now, now))

    result, err := repo.GetByKeyHash(ctx, keyHash)
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, keyHash, result.KeyHash)
    assert.Equal(t, "Test Key", result.Name)
    assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAPIKeyRepository_IncrementRateLimit(t *testing.T) {
    db, mock, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()

    repo := NewAPIKeyRepository(db)
    ctx := context.Background()

    req := &apikey.IncrementRateLimitRequest{
        TenantID:              "tenant-id",
        ResourceType:          "messages",
        WindowDurationSeconds: 60,
        LimitValue:            100,
        IncrementBy:           2,
    }

    mock.ExpectQuery(`INSERT INTO rate_limit_counters`).
        WithArgs(req.TenantID, req.APIKeyID, req.ResourceType, sqlmock.AnyArg(), req.WindowDurationSeconds, req.IncrementBy, req.LimitValue).
        WillReturnRows(sqlmock.NewRows([]string{"request_count", "limit_value"}).
            AddRow(2, 100))

    currentCount, exceeded, err := repo.IncrementRateLimit(ctx, req)
    assert.NoError(t, err)
    assert.Equal(t, 2, currentCount)
    assert.False(t, exceeded)
    assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAPIKeyRepository_UpdateLastUsed(t *testing.T) {
    db, mock, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()

    repo := NewAPIKeyRepository(db)
    ctx := context.Background()
    keyID := "key-id"

    mock.ExpectExec(`UPDATE api_keys SET last_used_at = NOW\(\), updated_at = NOW\(\) WHERE id = \$1`).
        WithArgs(keyID).
        WillReturnResult(sqlmock.NewResult(0, 1))

    err = repo.UpdateLastUsed(ctx, keyID)
    assert.NoError(t, err)
    assert.NoError(t, mock.ExpectationsWereMet())
}
