package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/domains/server"
	"github.com/lib/pq"
)

type ServerRepository struct {
	db *sql.DB
}

func NewServerRepository(db *sql.DB) *ServerRepository {
	return &ServerRepository{
		db: db,
	}
}

func (r *ServerRepository) CreateNode(ctx context.Context, node *server.ServerNode) error {
	metadataJSON, err := json.Marshal(node.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO server_nodes (id, server_id, region, hostname, ip_address, status, capacity, current_load, last_heartbeat, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err = r.db.QueryRowContext(ctx, query,
		node.ID, node.ServerID, node.Region, node.Hostname, node.IPAddress, node.Status, node.Capacity, node.CurrentLoad, node.LastHeartbeat, metadataJSON,
	).Scan(&node.ID, &node.CreatedAt, &node.UpdatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return fmt.Errorf("server with this server_id already exists")
		}
		return fmt.Errorf("failed to create server node: %w", err)
	}

	return nil
}

func (r *ServerRepository) GetNodeByID(ctx context.Context, id string) (*server.ServerNode, error) {
	query := `
		SELECT id, server_id, region, hostname, ip_address, status, capacity, current_load, last_heartbeat, metadata, created_at, updated_at
		FROM server_nodes
		WHERE id = $1
	`

	var node server.ServerNode
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&node.ID, &node.ServerID, &node.Region, &node.Hostname, &node.IPAddress, &node.Status, &node.Capacity, &node.CurrentLoad, &node.LastHeartbeat, &metadataJSON, &node.CreatedAt, &node.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("server node not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get server node: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &node.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &node, nil
}

func (r *ServerRepository) GetNodeByServerID(ctx context.Context, serverID string) (*server.ServerNode, error) {
	query := `
		SELECT id, server_id, region, hostname, ip_address, status, capacity, current_load, last_heartbeat, metadata, created_at, updated_at
		FROM server_nodes
		WHERE server_id = $1
	`

	var node server.ServerNode
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, serverID).Scan(
		&node.ID, &node.ServerID, &node.Region, &node.Hostname, &node.IPAddress, &node.Status, &node.Capacity, &node.CurrentLoad, &node.LastHeartbeat, &metadataJSON, &node.CreatedAt, &node.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("server node not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get server node: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &node.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &node, nil
}

func (r *ServerRepository) ListNodes(ctx context.Context, limit, offset int) ([]*server.ServerNode, error) {
	query := `
		SELECT id, server_id, region, hostname, ip_address, status, capacity, current_load, last_heartbeat, metadata, created_at, updated_at
		FROM server_nodes
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list server nodes: %w", err)
	}
	defer rows.Close()

	var nodes []*server.ServerNode
	for rows.Next() {
		var node server.ServerNode
		var metadataJSON []byte

		err := rows.Scan(
			&node.ID, &node.ServerID, &node.Region, &node.Hostname, &node.IPAddress, &node.Status, &node.Capacity, &node.CurrentLoad, &node.LastHeartbeat, &metadataJSON, &node.CreatedAt, &node.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan server node: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &node.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		nodes = append(nodes, &node)
	}

	return nodes, nil
}

func (r *ServerRepository) UpdateNode(ctx context.Context, id string, req *server.UpdateServerNodeRequest) error {
	query := "UPDATE server_nodes SET "
	args := []interface{}{}
	argIdx := 1

	if req.Status != nil {
		query += fmt.Sprintf("status = $%d, ", argIdx)
		args = append(args, *req.Status)
		argIdx++
	}

	if req.Capacity != nil {
		query += fmt.Sprintf("capacity = $%d, ", argIdx)
		args = append(args, *req.Capacity)
		argIdx++
	}

	if req.CurrentLoad != nil {
		query += fmt.Sprintf("current_load = $%d, ", argIdx)
		args = append(args, *req.CurrentLoad)
		argIdx++
	}

	if req.LastHeartbeat != nil {
		query += fmt.Sprintf("last_heartbeat = $%d, ", argIdx)
		args = append(args, *req.LastHeartbeat)
		argIdx++
	}

	if req.Metadata != nil {
		metadataJSON, err := json.Marshal(*req.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		query += fmt.Sprintf("metadata = $%d, ", argIdx)
		args = append(args, metadataJSON)
		argIdx++
	}

	if argIdx == 1 {
		return fmt.Errorf("no fields to update")
	}

	query += fmt.Sprintf("updated_at = NOW() WHERE id = $%d", argIdx)
	args = append(args, id)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update server node: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("server node not found")
	}

	return nil
}

func (r *ServerRepository) DeleteNode(ctx context.Context, id string) error {
	query := "DELETE FROM server_nodes WHERE id = $1"

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete server node: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("server node not found")
	}

	return nil
}

func (r *ServerRepository) UpdateHeartbeat(ctx context.Context, serverID string) error {
	query := "UPDATE server_nodes SET last_heartbeat = NOW(), updated_at = NOW() WHERE server_id = $1"

	result, err := r.db.ExecContext(ctx, query, serverID)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("server node not found")
	}

	return nil
}

func (r *ServerRepository) CreateSlot(ctx context.Context, slot *server.ServerSlot) error {
	query := `
		INSERT INTO server_slots (id, server_node_id, slot_number, status, tenant_id, allocated_at, released_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		slot.ID, slot.ServerNodeID, slot.SlotNumber, slot.Status, slot.TenantID, slot.AllocatedAt, slot.ReleasedAt,
	).Scan(&slot.ID, &slot.CreatedAt, &slot.UpdatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				return fmt.Errorf("slot already exists for this server node and slot number")
			}
			if pqErr.Code == "23503" {
				return fmt.Errorf("server node not found")
			}
		}
		return fmt.Errorf("failed to create slot: %w", err)
	}

	return nil
}

func (r *ServerRepository) GetSlotByID(ctx context.Context, id string) (*server.ServerSlot, error) {
	query := `
		SELECT id, server_node_id, slot_number, status, tenant_id, allocated_at, released_at, created_at, updated_at
		FROM server_slots
		WHERE id = $1
	`

	var slot server.ServerSlot
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&slot.ID, &slot.ServerNodeID, &slot.SlotNumber, &slot.Status, &slot.TenantID, &slot.AllocatedAt, &slot.ReleasedAt, &slot.CreatedAt, &slot.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("slot not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get slot: %w", err)
	}

	return &slot, nil
}

func (r *ServerRepository) GetSlotsByServerNodeID(ctx context.Context, serverNodeID string) ([]*server.ServerSlot, error) {
	query := `
		SELECT id, server_node_id, slot_number, status, tenant_id, allocated_at, released_at, created_at, updated_at
		FROM server_slots
		WHERE server_node_id = $1
		ORDER BY slot_number ASC
	`

	rows, err := r.db.QueryContext(ctx, query, serverNodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get slots: %w", err)
	}
	defer rows.Close()

	var slots []*server.ServerSlot
	for rows.Next() {
		var slot server.ServerSlot
		err := rows.Scan(
			&slot.ID, &slot.ServerNodeID, &slot.SlotNumber, &slot.Status, &slot.TenantID, &slot.AllocatedAt, &slot.ReleasedAt, &slot.CreatedAt, &slot.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan slot: %w", err)
		}
		slots = append(slots, &slot)
	}

	return slots, nil
}

func (r *ServerRepository) GetAvailableSlots(ctx context.Context, serverNodeID string) ([]*server.ServerSlot, error) {
	query := `
		SELECT id, server_node_id, slot_number, status, tenant_id, allocated_at, released_at, created_at, updated_at
		FROM server_slots
		WHERE server_node_id = $1 AND status = 'available'
		ORDER BY slot_number ASC
	`

	rows, err := r.db.QueryContext(ctx, query, serverNodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get available slots: %w", err)
	}
	defer rows.Close()

	var slots []*server.ServerSlot
	for rows.Next() {
		var slot server.ServerSlot
		err := rows.Scan(
			&slot.ID, &slot.ServerNodeID, &slot.SlotNumber, &slot.Status, &slot.TenantID, &slot.AllocatedAt, &slot.ReleasedAt, &slot.CreatedAt, &slot.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan slot: %w", err)
		}
		slots = append(slots, &slot)
	}

	return slots, nil
}

func (r *ServerRepository) AllocateSlot(ctx context.Context, slotID, tenantID string) error {
	now := time.Now()
	query := `
		UPDATE server_slots
		SET status = 'allocated', tenant_id = $1, allocated_at = $2, updated_at = NOW()
		WHERE id = $3 AND status = 'available'
	`

	result, err := r.db.ExecContext(ctx, query, tenantID, now, slotID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return fmt.Errorf("tenant not found")
		}
		return fmt.Errorf("failed to allocate slot: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("slot not found or not available")
	}

	return nil
}

func (r *ServerRepository) ReleaseSlot(ctx context.Context, slotID string) error {
	now := time.Now()
	query := `
		UPDATE server_slots
		SET status = 'available', tenant_id = NULL, released_at = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, now, slotID)
	if err != nil {
		return fmt.Errorf("failed to release slot: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("slot not found")
	}

	return nil
}

func (r *ServerRepository) UpdateSlotStatus(ctx context.Context, slotID string, status server.SlotStatus) error {
	query := "UPDATE server_slots SET status = $1, updated_at = NOW() WHERE id = $2"

	result, err := r.db.ExecContext(ctx, query, status, slotID)
	if err != nil {
		return fmt.Errorf("failed to update slot status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("slot not found")
	}

	return nil
}

func (r *ServerRepository) CreateAssignment(ctx context.Context, assignment *server.TenantServerAssignment) error {
	query := `
		INSERT INTO tenant_server_assignments (id, tenant_id, server_node_id, slot_id, assignment_type, status, priority, assigned_at, released_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		assignment.ID, assignment.TenantID, assignment.ServerNodeID, assignment.SlotID, assignment.AssignmentType, assignment.Status, assignment.Priority, assignment.AssignedAt, assignment.ReleasedAt,
	).Scan(&assignment.ID, &assignment.CreatedAt, &assignment.UpdatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				return fmt.Errorf("assignment already exists for this tenant and server")
			}
			if pqErr.Code == "23503" {
				return fmt.Errorf("tenant or server node not found")
			}
		}
		return fmt.Errorf("failed to create assignment: %w", err)
	}

	return nil
}

func (r *ServerRepository) GetAssignmentByID(ctx context.Context, id string) (*server.TenantServerAssignment, error) {
	query := `
		SELECT id, tenant_id, server_node_id, slot_id, assignment_type, status, priority, assigned_at, released_at, created_at, updated_at
		FROM tenant_server_assignments
		WHERE id = $1
	`

	var assignment server.TenantServerAssignment
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&assignment.ID, &assignment.TenantID, &assignment.ServerNodeID, &assignment.SlotID, &assignment.AssignmentType, &assignment.Status, &assignment.Priority, &assignment.AssignedAt, &assignment.ReleasedAt, &assignment.CreatedAt, &assignment.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("assignment not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment: %w", err)
	}

	return &assignment, nil
}

func (r *ServerRepository) GetAssignmentsByTenantID(ctx context.Context, tenantID string) ([]*server.TenantServerAssignment, error) {
	query := `
		SELECT id, tenant_id, server_node_id, slot_id, assignment_type, status, priority, assigned_at, released_at, created_at, updated_at
		FROM tenant_server_assignments
		WHERE tenant_id = $1
		ORDER BY priority DESC, assigned_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignments: %w", err)
	}
	defer rows.Close()

	var assignments []*server.TenantServerAssignment
	for rows.Next() {
		var assignment server.TenantServerAssignment
		err := rows.Scan(
			&assignment.ID, &assignment.TenantID, &assignment.ServerNodeID, &assignment.SlotID, &assignment.AssignmentType, &assignment.Status, &assignment.Priority, &assignment.AssignedAt, &assignment.ReleasedAt, &assignment.CreatedAt, &assignment.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan assignment: %w", err)
		}
		assignments = append(assignments, &assignment)
	}

	return assignments, nil
}

func (r *ServerRepository) GetAssignmentsByServerNodeID(ctx context.Context, serverNodeID string) ([]*server.TenantServerAssignment, error) {
	query := `
		SELECT id, tenant_id, server_node_id, slot_id, assignment_type, status, priority, assigned_at, released_at, created_at, updated_at
		FROM tenant_server_assignments
		WHERE server_node_id = $1
		ORDER BY assigned_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, serverNodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignments: %w", err)
	}
	defer rows.Close()

	var assignments []*server.TenantServerAssignment
	for rows.Next() {
		var assignment server.TenantServerAssignment
		err := rows.Scan(
			&assignment.ID, &assignment.TenantID, &assignment.ServerNodeID, &assignment.SlotID, &assignment.AssignmentType, &assignment.Status, &assignment.Priority, &assignment.AssignedAt, &assignment.ReleasedAt, &assignment.CreatedAt, &assignment.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan assignment: %w", err)
		}
		assignments = append(assignments, &assignment)
	}

	return assignments, nil
}

func (r *ServerRepository) UpdateAssignmentStatus(ctx context.Context, id string, status server.AssignmentStatus) error {
	query := "UPDATE tenant_server_assignments SET status = $1, updated_at = NOW() WHERE id = $2"

	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update assignment status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("assignment not found")
	}

	return nil
}

func (r *ServerRepository) ReleaseAssignment(ctx context.Context, id string) error {
	now := time.Now()
	query := "UPDATE tenant_server_assignments SET status = 'inactive', released_at = $1, updated_at = NOW() WHERE id = $2"

	result, err := r.db.ExecContext(ctx, query, now, id)
	if err != nil {
		return fmt.Errorf("failed to release assignment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("assignment not found")
	}

	return nil
}

func (r *ServerRepository) RecordMetric(ctx context.Context, metric *server.ServerMetric) error {
	tagsJSON, err := json.Marshal(metric.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		INSERT INTO server_metrics (id, server_node_id, metric_type, metric_value, unit, tags, recorded_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id, created_at
	`

	err = r.db.QueryRowContext(ctx, query,
		metric.ID, metric.ServerNodeID, metric.MetricType, metric.MetricValue, metric.Unit, tagsJSON, metric.RecordedAt,
	).Scan(&metric.ID, &metric.CreatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return fmt.Errorf("server node not found")
		}
		return fmt.Errorf("failed to record metric: %w", err)
	}

	return nil
}

func (r *ServerRepository) GetMetrics(ctx context.Context, serverNodeID string, metricType *string, startTime, endTime time.Time) ([]*server.ServerMetric, error) {
	query := `
		SELECT id, server_node_id, metric_type, metric_value, unit, tags, recorded_at, created_at
		FROM server_metrics
		WHERE server_node_id = $1 AND recorded_at >= $2 AND recorded_at <= $3
	`
	args := []interface{}{serverNodeID, startTime, endTime}

	if metricType != nil {
		query += " AND metric_type = $4"
		args = append(args, *metricType)
	}

	query += " ORDER BY recorded_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}
	defer rows.Close()

	var metrics []*server.ServerMetric
	for rows.Next() {
		var metric server.ServerMetric
		var tagsJSON []byte

		err := rows.Scan(
			&metric.ID, &metric.ServerNodeID, &metric.MetricType, &metric.MetricValue, &metric.Unit, &tagsJSON, &metric.RecordedAt, &metric.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}

		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &metric.Tags); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
			}
		}

		metrics = append(metrics, &metric)
	}

	return metrics, nil
}

func (r *ServerRepository) GetLatestMetrics(ctx context.Context, serverNodeID string, limit int) ([]*server.ServerMetric, error) {
	query := `
		SELECT id, server_node_id, metric_type, metric_value, unit, tags, recorded_at, created_at
		FROM server_metrics
		WHERE server_node_id = $1
		ORDER BY recorded_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, serverNodeID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest metrics: %w", err)
	}
	defer rows.Close()

	var metrics []*server.ServerMetric
	for rows.Next() {
		var metric server.ServerMetric
		var tagsJSON []byte

		err := rows.Scan(
			&metric.ID, &metric.ServerNodeID, &metric.MetricType, &metric.MetricValue, &metric.Unit, &tagsJSON, &metric.RecordedAt, &metric.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}

		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &metric.Tags); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
			}
		}

		metrics = append(metrics, &metric)
	}

	return metrics, nil
}
