# Management Database Setup Guide

This document provides detailed information about the management database schema and how to set it up.

## Overview

The management database is an optional PostgreSQL database that enables multi-tenant operations, server capacity management, API key authentication, rate limiting, and audit logging for the WhatsApp API platform.

## Database Schema

### Core Tables

#### tenants
Stores customer/tenant information with tier-based access control.

- `id` (UUID) - Primary key
- `name` (VARCHAR) - Tenant name
- `status` (VARCHAR) - One of: active, suspended, cancelled
- `tier` (VARCHAR) - One of: free, basic, pro, enterprise
- `metadata` (JSONB) - Additional tenant metadata
- `created_at`, `updated_at` (TIMESTAMPTZ) - Timestamps

#### tenant_contact
Contact information for each tenant.

- `id` (UUID) - Primary key
- `tenant_id` (UUID) - Foreign key to tenants
- `contact_type` (VARCHAR) - One of: email, phone, billing, technical
- `contact_value` (VARCHAR) - The actual contact value
- `is_primary` (BOOLEAN) - Whether this is the primary contact
- `created_at`, `updated_at` (TIMESTAMPTZ) - Timestamps

#### api_keys
API keys with scopes and expiration for tenant authentication.

- `id` (UUID) - Primary key
- `tenant_id` (UUID) - Foreign key to tenants
- `name` (VARCHAR) - Human-readable key name
- `key_hash` (VARCHAR) - Hashed API key (UNIQUE)
- `key_prefix` (VARCHAR) - Visible prefix (e.g., "sk_test")
- `scopes` (TEXT[]) - Array of scopes/permissions
- `status` (VARCHAR) - One of: active, revoked, expired
- `expires_at` (TIMESTAMPTZ) - Optional expiration time
- `last_used_at` (TIMESTAMPTZ) - Last usage timestamp
- `created_at`, `updated_at` (TIMESTAMPTZ) - Timestamps

#### server_nodes
Infrastructure nodes that host WhatsApp connections.

- `id` (UUID) - Primary key
- `server_id` (VARCHAR) - Unique server identifier (UNIQUE)
- `region` (VARCHAR) - Datacenter/region (e.g., "us-east-1")
- `hostname` (VARCHAR) - Server hostname
- `ip_address` (INET) - Server IP address
- `status` (VARCHAR) - One of: active, maintenance, offline, decommissioned
- `capacity` (INTEGER) - Maximum slots/connections
- `current_load` (INTEGER) - Current number of allocated slots
- `last_heartbeat` (TIMESTAMPTZ) - Last health check
- `metadata` (JSONB) - Additional server metadata
- `created_at`, `updated_at` (TIMESTAMPTZ) - Timestamps

#### server_slots
Granular allocation units on each server node.

- `id` (UUID) - Primary key
- `server_node_id` (UUID) - Foreign key to server_nodes
- `slot_number` (INTEGER) - Slot number within the server
- `status` (VARCHAR) - One of: available, allocated, reserved, maintenance
- `tenant_id` (UUID) - Foreign key to tenants (nullable)
- `allocated_at`, `released_at` (TIMESTAMPTZ) - Allocation timestamps
- `created_at`, `updated_at` (TIMESTAMPTZ) - Timestamps
- UNIQUE constraint on (server_node_id, slot_number)

#### tenant_server_assignments
Maps tenants to server nodes (dedicated or shared).

- `id` (UUID) - Primary key
- `tenant_id` (UUID) - Foreign key to tenants
- `server_node_id` (UUID) - Foreign key to server_nodes
- `slot_id` (UUID) - Foreign key to server_slots (nullable)
- `assignment_type` (VARCHAR) - One of: dedicated, shared
- `status` (VARCHAR) - One of: active, inactive, migrating
- `priority` (INTEGER) - Assignment priority
- `assigned_at`, `released_at` (TIMESTAMPTZ) - Assignment timestamps
- `created_at`, `updated_at` (TIMESTAMPTZ) - Timestamps
- UNIQUE constraint on (tenant_id, server_node_id, assignment_type)

#### rate_limit_counters
Tracks rate limiting per tenant/API key and resource.

- `id` (UUID) - Primary key
- `tenant_id` (UUID) - Foreign key to tenants
- `api_key_id` (UUID) - Foreign key to api_keys (nullable)
- `resource_type` (VARCHAR) - Resource being rate-limited (e.g., "messages")
- `window_start` (TIMESTAMPTZ) - Start of the rate limit window
- `window_duration_seconds` (INTEGER) - Duration of the window
- `request_count` (INTEGER) - Number of requests in this window
- `limit_value` (INTEGER) - Maximum allowed requests
- `created_at`, `updated_at` (TIMESTAMPTZ) - Timestamps
- UNIQUE constraint on (tenant_id, api_key_id, resource_type, window_start)

#### audit_logs
Complete audit trail of all actions.

- `id` (UUID) - Primary key
- `tenant_id` (UUID) - Foreign key to tenants (nullable)
- `api_key_id` (UUID) - Foreign key to api_keys (nullable)
- `server_node_id` (UUID) - Foreign key to server_nodes (nullable)
- `action` (VARCHAR) - Action taken (e.g., "message.send")
- `resource_type` (VARCHAR) - Type of resource acted upon
- `resource_id` (VARCHAR) - ID of the resource
- `actor_type` (VARCHAR) - One of: user, system, api_key
- `actor_id` (VARCHAR) - ID of the actor
- `status` (VARCHAR) - One of: success, failure, error
- `ip_address` (INET) - IP address of the request
- `user_agent` (TEXT) - User agent string
- `request_id` (VARCHAR) - Correlation ID
- `details` (JSONB) - Additional details
- `created_at` (TIMESTAMPTZ) - Timestamp

#### server_metrics
Performance and health metrics for servers.

- `id` (UUID) - Primary key
- `server_node_id` (UUID) - Foreign key to server_nodes
- `metric_type` (VARCHAR) - Type of metric (e.g., "cpu_usage")
- `metric_value` (NUMERIC) - Metric value
- `unit` (VARCHAR) - Unit of measurement
- `tags` (JSONB) - Additional tags/labels
- `recorded_at` (TIMESTAMPTZ) - When the metric was recorded
- `created_at` (TIMESTAMPTZ) - Timestamp

#### billing_snapshots
Usage tracking for billing purposes.

- `id` (UUID) - Primary key
- `tenant_id` (UUID) - Foreign key to tenants
- `period_start`, `period_end` (TIMESTAMPTZ) - Billing period
- `total_requests` (INTEGER) - Total API requests
- `total_messages_sent` (INTEGER) - Total messages sent
- `total_media_bytes` (BIGINT) - Total media transferred
- `total_compute_seconds` (INTEGER) - Total compute time
- `cost_amount` (NUMERIC) - Calculated cost
- `cost_currency` (VARCHAR) - Currency code (e.g., "USD")
- `status` (VARCHAR) - One of: pending, finalized, paid, overdue
- `metadata` (JSONB) - Additional billing metadata
- `created_at`, `updated_at` (TIMESTAMPTZ) - Timestamps

## Setup Instructions

### 1. Create PostgreSQL Database

```bash
# Create database
createdb management

# Or using psql
psql -U postgres -c "CREATE DATABASE management;"
```

### 2. Configure Environment Variables

Create or update your `.env` file:

```bash
# Management Database Configuration
MANAGEMENT_DB_URI=postgres://user:password@localhost:5432/management?sslmode=disable
MANAGEMENT_DB_MAX_CONNS=25
MANAGEMENT_DB_MIN_CONNS=5
MANAGEMENT_DB_MAX_IDLE_TIME=15m
MANAGEMENT_DB_MAX_LIFETIME=1h

# Server Identity
SERVER_ID=server-001
SERVER_REGION=us-east-1
```

### 3. Run Migrations

Migrations are embedded in the application and run automatically on startup. Simply start the application:

```bash
cd src
./whatsapp rest
```

The first time you run the application, it will automatically:
1. Connect to the management database
2. Run all pending migrations
3. Create all tables and indexes
4. Set up triggers for updated_at columns

You can verify the migration by checking the logs:

```
INFO Current migration version: 1 (dirty: false)
```

### 4. Verify Installation

Connect to your database and verify tables were created:

```bash
psql -U postgres management -c "\dt"
```

You should see all the tables listed above.

## Repository Usage

### Go Code Examples

#### Tenant Operations

```go
import (
    "context"
    "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/tenant"
    "github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/management/postgres"
)

// Initialize repository
db, _ := postgres.NewDB(&postgres.DBConfig{
    URI:         "postgres://user:password@localhost:5432/management",
    MaxConns:    25,
    MinConns:    5,
    MaxIdleTime: "15m",
    MaxLifetime: "1h",
})
tenantRepo := postgres.NewTenantRepository(db)

// Create a tenant
newTenant := &tenant.Tenant{
    ID:     "uuid-here",
    Name:   "Acme Corp",
    Status: tenant.TenantStatusActive,
    Tier:   tenant.TenantTierPro,
    Metadata: map[string]interface{}{
        "industry": "technology",
    },
}
err := tenantRepo.Create(context.Background(), newTenant)

// Get tenant by ID
tenant, err := tenantRepo.GetByID(context.Background(), "uuid-here")
```

#### API Key Operations

```go
import (
    "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/apikey"
    "github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/management/postgres"
)

keyRepo := postgres.NewAPIKeyRepository(db)

// Create API key
newKey := &apikey.APIKey{
    ID:        "key-uuid",
    TenantID:  "tenant-uuid",
    Name:      "Production Key",
    KeyHash:   "hashed-value",
    KeyPrefix: "sk_live",
    Scopes:    []string{"messages:send", "messages:read"},
    Status:    apikey.APIKeyStatusActive,
}
err := keyRepo.Create(context.Background(), newKey)

// Check rate limit
req := &apikey.IncrementRateLimitRequest{
    TenantID:              "tenant-uuid",
    ResourceType:          "messages",
    WindowDurationSeconds: 60,
    LimitValue:            100,
}
count, exceeded, err := keyRepo.IncrementRateLimit(context.Background(), req)
```

#### Server Management

```go
import (
    "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/server"
    "github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/management/postgres"
)

serverRepo := postgres.NewServerRepository(db)

// Register server node
node := &server.ServerNode{
    ID:         "node-uuid",
    ServerID:   "server-001",
    Region:     "us-east-1",
    Hostname:   "whatsapp-01.example.com",
    Status:     server.ServerStatusActive,
    Capacity:   100,
    CurrentLoad: 0,
}
err := serverRepo.CreateNode(context.Background(), node)

// Allocate slot to tenant
slot := &server.ServerSlot{
    ID:           "slot-uuid",
    ServerNodeID: "node-uuid",
    SlotNumber:   1,
    Status:       server.SlotStatusAvailable,
}
err = serverRepo.CreateSlot(context.Background(), slot)

err = serverRepo.AllocateSlot(context.Background(), "slot-uuid", "tenant-uuid")
```

#### Audit Logging

```go
import (
    "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/audit"
    "github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/management/postgres"
)

auditRepo := postgres.NewAuditRepository(db)

// Create audit log
log := &audit.AuditLog{
    ID:           "log-uuid",
    TenantID:     stringPtr("tenant-uuid"),
    Action:       "message.send",
    ResourceType: "message",
    ResourceID:   stringPtr("msg-123"),
    ActorType:    audit.ActorTypeAPIKey,
    Status:       audit.AuditStatusSuccess,
    Details: map[string]interface{}{
        "to":      "+1234567890",
        "message": "Hello",
    },
}
err := auditRepo.CreateLog(context.Background(), log)

// Query audit logs
filter := &audit.AuditLogFilter{
    TenantID: stringPtr("tenant-uuid"),
    Limit:    50,
    Offset:   0,
}
logs, err := auditRepo.ListLogs(context.Background(), filter)
```

## Maintenance

### Backup

```bash
pg_dump -U postgres management > management_backup.sql
```

### Restore

```bash
psql -U postgres management < management_backup.sql
```

### Cleanup Old Data

The repository provides cleanup methods:

```go
// Cleanup old audit logs (older than 90 days)
err := auditRepo.CleanupOldLogs(ctx, time.Now().AddDate(0, 0, -90))

// Cleanup expired rate limit counters (older than 24 hours)
err := keyRepo.CleanupExpiredCounters(ctx, time.Now().Add(-24*time.Hour))
```

## Performance Considerations

- All foreign keys and frequently queried columns have indexes
- JSONB columns use GIN indexes where appropriate
- `updated_at` columns are automatically maintained by triggers
- Connection pooling is configured via environment variables
- Rate limit counters use ON CONFLICT for efficient upserts

## Security

- API keys are stored as hashed values, never plain text
- Foreign key constraints ensure referential integrity
- CASCADE deletes are used carefully to maintain data consistency
- All timestamps use TIMESTAMPTZ for timezone awareness
- INET type for IP addresses provides validation and efficient storage

## Testing

The repository includes comprehensive unit tests using sqlmock:

```bash
cd src
go test ./infrastructure/management/postgres -v
go test ./infrastructure/management/migrations -v
```

## Troubleshooting

### Migration fails with "database is dirty"

This means a previous migration failed mid-way. To fix:

```sql
-- Check current version
SELECT * FROM schema_migrations;

-- Reset dirty flag (careful!)
UPDATE schema_migrations SET dirty = false WHERE version = X;
```

### Connection pool exhausted

Increase the pool size:

```bash
export MANAGEMENT_DB_MAX_CONNS=50
export MANAGEMENT_DB_MIN_CONNS=10
```

### Slow queries

Enable query logging in PostgreSQL:

```sql
ALTER DATABASE management SET log_statement = 'all';
```

Then review the logs and add indexes as needed.
