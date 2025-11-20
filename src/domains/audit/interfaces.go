package audit

import (
	"context"
	"time"
)

type IAuditRepository interface {
	CreateLog(ctx context.Context, log *AuditLog) error
	GetLogByID(ctx context.Context, id string) (*AuditLog, error)
	ListLogs(ctx context.Context, filter *AuditLogFilter) ([]*AuditLog, error)
	GetRecentLogs(ctx context.Context, tenantID *string, limit int) ([]*AuditLog, error)
	CleanupOldLogs(ctx context.Context, before time.Time) error

	CreateBillingSnapshot(ctx context.Context, snapshot *BillingSnapshot) error
	GetBillingSnapshotByID(ctx context.Context, id string) (*BillingSnapshot, error)
	GetBillingSnapshotsByTenantID(ctx context.Context, tenantID string, startTime, endTime time.Time) ([]*BillingSnapshot, error)
	UpdateBillingSnapshot(ctx context.Context, id string, snapshot *BillingSnapshot) error
}
