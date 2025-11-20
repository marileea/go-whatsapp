package auditlogger

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	domainAudit "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/audit"
	"github.com/stretchr/testify/require"
)

func TestAsyncLoggerRecordsStatuses(t *testing.T) {
	repo := &capturingAuditRepo{}
	logger := NewLogger(repo, WithQueueSize(4), WithWorkerCount(1), WithServerNodeID("server-node"))
	t.Cleanup(func() {
		require.NoError(t, logger.Close(context.Background()))
	})

	logger.Record(RequestEntry{
		TenantID:           "tenant-1",
		APIKeyID:           "key-1",
		ServerAssignmentID: "assign-1",
		Method:             "GET",
		Path:               "/app/status",
		StatusCode:         200,
		Latency:            12 * time.Millisecond,
	})
	logger.Record(RequestEntry{
		TenantID:   "tenant-1",
		APIKeyID:   "key-1",
		Method:     "POST",
		Path:       "/send/text",
		StatusCode: 404,
		Latency:    5 * time.Millisecond,
	})
	logger.Record(RequestEntry{
		TenantID:   "tenant-1",
		APIKeyID:   "key-1",
		Method:     "GET",
		Path:       "/app/devices",
		StatusCode: 500,
		Error:      errors.New("boom"),
	})

	require.Eventually(t, func() bool {
		return repo.Count() == 3
	}, time.Second, 10*time.Millisecond)

	logs := repo.Logs()
	require.Equal(t, domainAudit.AuditStatusSuccess, logs[0].Status)
	require.Equal(t, "assign-1", logs[0].Details["server_assignment_id"])

	require.Equal(t, domainAudit.AuditStatusFailure, logs[1].Status)
	require.Equal(t, "POST /send/text", logs[1].Action)

	require.Equal(t, domainAudit.AuditStatusError, logs[2].Status)
	require.Equal(t, "server-node", *logs[2].ServerNodeID)
}

// capturingAuditRepo is a minimal in-memory repository used for testing.
type capturingAuditRepo struct {
	mu   sync.Mutex
	logs []*domainAudit.AuditLog
}

func (r *capturingAuditRepo) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.logs)
}

func (r *capturingAuditRepo) Logs() []*domainAudit.AuditLog {
	r.mu.Lock()
	defer r.mu.Unlock()
	copied := make([]*domainAudit.AuditLog, len(r.logs))
	copy(copied, r.logs)
	return copied
}

func (r *capturingAuditRepo) CreateLog(_ context.Context, log *domainAudit.AuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, log)
	return nil
}

func (r *capturingAuditRepo) GetLogByID(context.Context, string) (*domainAudit.AuditLog, error) {
	return nil, nil
}
func (r *capturingAuditRepo) ListLogs(context.Context, *domainAudit.AuditLogFilter) ([]*domainAudit.AuditLog, error) {
	return nil, nil
}
func (r *capturingAuditRepo) GetRecentLogs(context.Context, *string, int) ([]*domainAudit.AuditLog, error) {
	return nil, nil
}
func (r *capturingAuditRepo) CleanupOldLogs(context.Context, time.Time) error { return nil }
func (r *capturingAuditRepo) CreateBillingSnapshot(context.Context, *domainAudit.BillingSnapshot) error {
	return nil
}
func (r *capturingAuditRepo) GetBillingSnapshotByID(context.Context, string) (*domainAudit.BillingSnapshot, error) {
	return nil, nil
}
func (r *capturingAuditRepo) GetBillingSnapshotsByTenantID(context.Context, string, time.Time, time.Time) ([]*domainAudit.BillingSnapshot, error) {
	return nil, nil
}
func (r *capturingAuditRepo) UpdateBillingSnapshot(context.Context, string, *domainAudit.BillingSnapshot) error {
	return nil
}
