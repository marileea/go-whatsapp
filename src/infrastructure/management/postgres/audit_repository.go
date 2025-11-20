package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/domains/audit"
	"github.com/lib/pq"
)

type AuditRepository struct {
	db *sql.DB
}

func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{
		db: db,
	}
}

func (r *AuditRepository) CreateLog(ctx context.Context, log *audit.AuditLog) error {
	detailsJSON, err := json.Marshal(log.Details)
	if err != nil {
		return fmt.Errorf("failed to marshal details: %w", err)
	}

	query := `
		INSERT INTO audit_logs (id, tenant_id, api_key_id, server_node_id, action, resource_type, resource_id, actor_type, actor_id, status, ip_address, user_agent, request_id, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
		RETURNING id, created_at
	`

	err = r.db.QueryRowContext(ctx, query,
		log.ID, log.TenantID, log.APIKeyID, log.ServerNodeID, log.Action, log.ResourceType, log.ResourceID, log.ActorType, log.ActorID, log.Status, log.IPAddress, log.UserAgent, log.RequestID, detailsJSON,
	).Scan(&log.ID, &log.CreatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return fmt.Errorf("referenced entity not found")
		}
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

func (r *AuditRepository) GetLogByID(ctx context.Context, id string) (*audit.AuditLog, error) {
	query := `
		SELECT id, tenant_id, api_key_id, server_node_id, action, resource_type, resource_id, actor_type, actor_id, status, ip_address, user_agent, request_id, details, created_at
		FROM audit_logs
		WHERE id = $1
	`

	var log audit.AuditLog
	var detailsJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&log.ID, &log.TenantID, &log.APIKeyID, &log.ServerNodeID, &log.Action, &log.ResourceType, &log.ResourceID, &log.ActorType, &log.ActorID, &log.Status, &log.IPAddress, &log.UserAgent, &log.RequestID, &detailsJSON, &log.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("audit log not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}

	if len(detailsJSON) > 0 {
		if err := json.Unmarshal(detailsJSON, &log.Details); err != nil {
			return nil, fmt.Errorf("failed to unmarshal details: %w", err)
		}
	}

	return &log, nil
}

func (r *AuditRepository) ListLogs(ctx context.Context, filter *audit.AuditLogFilter) ([]*audit.AuditLog, error) {
	query := `
		SELECT id, tenant_id, api_key_id, server_node_id, action, resource_type, resource_id, actor_type, actor_id, status, ip_address, user_agent, request_id, details, created_at
		FROM audit_logs
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if filter.TenantID != nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", argIdx)
		args = append(args, *filter.TenantID)
		argIdx++
	}

	if filter.APIKeyID != nil {
		query += fmt.Sprintf(" AND api_key_id = $%d", argIdx)
		args = append(args, *filter.APIKeyID)
		argIdx++
	}

	if filter.ServerNodeID != nil {
		query += fmt.Sprintf(" AND server_node_id = $%d", argIdx)
		args = append(args, *filter.ServerNodeID)
		argIdx++
	}

	if filter.Action != nil {
		query += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, *filter.Action)
		argIdx++
	}

	if filter.ResourceType != nil {
		query += fmt.Sprintf(" AND resource_type = $%d", argIdx)
		args = append(args, *filter.ResourceType)
		argIdx++
	}

	if filter.ActorType != nil {
		query += fmt.Sprintf(" AND actor_type = $%d", argIdx)
		args = append(args, *filter.ActorType)
		argIdx++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filter.Status)
		argIdx++
	}

	if filter.StartTime != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *filter.StartTime)
		argIdx++
	}

	if filter.EndTime != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *filter.EndTime)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filter.Limit)
		argIdx++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
		argIdx++
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*audit.AuditLog
	for rows.Next() {
		var log audit.AuditLog
		var detailsJSON []byte

		err := rows.Scan(
			&log.ID, &log.TenantID, &log.APIKeyID, &log.ServerNodeID, &log.Action, &log.ResourceType, &log.ResourceID, &log.ActorType, &log.ActorID, &log.Status, &log.IPAddress, &log.UserAgent, &log.RequestID, &detailsJSON, &log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &log.Details); err != nil {
				return nil, fmt.Errorf("failed to unmarshal details: %w", err)
			}
		}

		logs = append(logs, &log)
	}

	return logs, nil
}

func (r *AuditRepository) GetRecentLogs(ctx context.Context, tenantID *string, limit int) ([]*audit.AuditLog, error) {
	query := `
		SELECT id, tenant_id, api_key_id, server_node_id, action, resource_type, resource_id, actor_type, actor_id, status, ip_address, user_agent, request_id, details, created_at
		FROM audit_logs
	`
	args := []interface{}{}

	if tenantID != nil {
		query += " WHERE tenant_id = $1"
		args = append(args, *tenantID)
	}

	query += " ORDER BY created_at DESC LIMIT " + fmt.Sprintf("%d", limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent logs: %w", err)
	}
	defer rows.Close()

	var logs []*audit.AuditLog
	for rows.Next() {
		var log audit.AuditLog
		var detailsJSON []byte

		err := rows.Scan(
			&log.ID, &log.TenantID, &log.APIKeyID, &log.ServerNodeID, &log.Action, &log.ResourceType, &log.ResourceID, &log.ActorType, &log.ActorID, &log.Status, &log.IPAddress, &log.UserAgent, &log.RequestID, &detailsJSON, &log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &log.Details); err != nil {
				return nil, fmt.Errorf("failed to unmarshal details: %w", err)
			}
		}

		logs = append(logs, &log)
	}

	return logs, nil
}

func (r *AuditRepository) CleanupOldLogs(ctx context.Context, before time.Time) error {
	query := "DELETE FROM audit_logs WHERE created_at < $1"

	result, err := r.db.ExecContext(ctx, query, before)
	if err != nil {
		return fmt.Errorf("failed to cleanup old logs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		fmt.Printf("Cleaned up %d old audit logs\n", rowsAffected)
	}

	return nil
}

func (r *AuditRepository) CreateBillingSnapshot(ctx context.Context, snapshot *audit.BillingSnapshot) error {
	metadataJSON, err := json.Marshal(snapshot.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO billing_snapshots (id, tenant_id, period_start, period_end, total_requests, total_messages_sent, total_media_bytes, total_compute_seconds, cost_amount, cost_currency, status, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err = r.db.QueryRowContext(ctx, query,
		snapshot.ID, snapshot.TenantID, snapshot.PeriodStart, snapshot.PeriodEnd, snapshot.TotalRequests, snapshot.TotalMessagesSent, snapshot.TotalMediaBytes, snapshot.TotalComputeSeconds, snapshot.CostAmount, snapshot.CostCurrency, snapshot.Status, metadataJSON,
	).Scan(&snapshot.ID, &snapshot.CreatedAt, &snapshot.UpdatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return fmt.Errorf("tenant not found")
		}
		return fmt.Errorf("failed to create billing snapshot: %w", err)
	}

	return nil
}

func (r *AuditRepository) GetBillingSnapshotByID(ctx context.Context, id string) (*audit.BillingSnapshot, error) {
	query := `
		SELECT id, tenant_id, period_start, period_end, total_requests, total_messages_sent, total_media_bytes, total_compute_seconds, cost_amount, cost_currency, status, metadata, created_at, updated_at
		FROM billing_snapshots
		WHERE id = $1
	`

	var snapshot audit.BillingSnapshot
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&snapshot.ID, &snapshot.TenantID, &snapshot.PeriodStart, &snapshot.PeriodEnd, &snapshot.TotalRequests, &snapshot.TotalMessagesSent, &snapshot.TotalMediaBytes, &snapshot.TotalComputeSeconds, &snapshot.CostAmount, &snapshot.CostCurrency, &snapshot.Status, &metadataJSON, &snapshot.CreatedAt, &snapshot.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("billing snapshot not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get billing snapshot: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &snapshot.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &snapshot, nil
}

func (r *AuditRepository) GetBillingSnapshotsByTenantID(ctx context.Context, tenantID string, startTime, endTime time.Time) ([]*audit.BillingSnapshot, error) {
	query := `
		SELECT id, tenant_id, period_start, period_end, total_requests, total_messages_sent, total_media_bytes, total_compute_seconds, cost_amount, cost_currency, status, metadata, created_at, updated_at
		FROM billing_snapshots
		WHERE tenant_id = $1 AND period_start >= $2 AND period_end <= $3
		ORDER BY period_start DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get billing snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []*audit.BillingSnapshot
	for rows.Next() {
		var snapshot audit.BillingSnapshot
		var metadataJSON []byte

		err := rows.Scan(
			&snapshot.ID, &snapshot.TenantID, &snapshot.PeriodStart, &snapshot.PeriodEnd, &snapshot.TotalRequests, &snapshot.TotalMessagesSent, &snapshot.TotalMediaBytes, &snapshot.TotalComputeSeconds, &snapshot.CostAmount, &snapshot.CostCurrency, &snapshot.Status, &metadataJSON, &snapshot.CreatedAt, &snapshot.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan billing snapshot: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &snapshot.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		snapshots = append(snapshots, &snapshot)
	}

	return snapshots, nil
}

func (r *AuditRepository) UpdateBillingSnapshot(ctx context.Context, id string, snapshot *audit.BillingSnapshot) error {
	metadataJSON, err := json.Marshal(snapshot.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var setClauses []string
	var args []interface{}
	argIdx := 1

	if snapshot.TotalRequests > 0 {
		setClauses = append(setClauses, fmt.Sprintf("total_requests = $%d", argIdx))
		args = append(args, snapshot.TotalRequests)
		argIdx++
	}

	if snapshot.TotalMessagesSent > 0 {
		setClauses = append(setClauses, fmt.Sprintf("total_messages_sent = $%d", argIdx))
		args = append(args, snapshot.TotalMessagesSent)
		argIdx++
	}

	if snapshot.TotalMediaBytes > 0 {
		setClauses = append(setClauses, fmt.Sprintf("total_media_bytes = $%d", argIdx))
		args = append(args, snapshot.TotalMediaBytes)
		argIdx++
	}

	if snapshot.TotalComputeSeconds > 0 {
		setClauses = append(setClauses, fmt.Sprintf("total_compute_seconds = $%d", argIdx))
		args = append(args, snapshot.TotalComputeSeconds)
		argIdx++
	}

	if snapshot.CostAmount != nil {
		setClauses = append(setClauses, fmt.Sprintf("cost_amount = $%d", argIdx))
		args = append(args, *snapshot.CostAmount)
		argIdx++
	}

	if snapshot.Status != "" {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, snapshot.Status)
		argIdx++
	}

	if len(metadataJSON) > 0 {
		setClauses = append(setClauses, fmt.Sprintf("metadata = $%d", argIdx))
		args = append(args, metadataJSON)
		argIdx++
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("no fields to update")
	}

	setClauses = append(setClauses, "updated_at = NOW()")

	query := fmt.Sprintf("UPDATE billing_snapshots SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update billing snapshot: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("billing snapshot not found")
	}

	return nil
}
