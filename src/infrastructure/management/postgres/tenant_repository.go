package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/domains/tenant"
	"github.com/lib/pq"
)

type TenantRepository struct {
	db *sql.DB
}

func NewTenantRepository(db *sql.DB) *TenantRepository {
	return &TenantRepository{
		db: db,
	}
}

func (r *TenantRepository) Create(ctx context.Context, t *tenant.Tenant) error {
	metadataJSON, err := json.Marshal(t.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO tenants (id, name, status, tier, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err = r.db.QueryRowContext(ctx, query,
		t.ID, t.Name, t.Status, t.Tier, metadataJSON,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	return nil
}

func (r *TenantRepository) GetByID(ctx context.Context, id string) (*tenant.Tenant, error) {
	query := `
		SELECT id, name, status, tier, metadata, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`

	var t tenant.Tenant
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID, &t.Name, &t.Status, &t.Tier, &metadataJSON, &t.CreatedAt, &t.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tenant not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &t.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &t, nil
}

func (r *TenantRepository) List(ctx context.Context, limit, offset int) ([]*tenant.Tenant, error) {
	query := `
		SELECT id, name, status, tier, metadata, created_at, updated_at
		FROM tenants
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*tenant.Tenant
	for rows.Next() {
		var t tenant.Tenant
		var metadataJSON []byte

		err := rows.Scan(
			&t.ID, &t.Name, &t.Status, &t.Tier, &metadataJSON, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &t.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		tenants = append(tenants, &t)
	}

	return tenants, nil
}

func (r *TenantRepository) Update(ctx context.Context, id string, req *tenant.UpdateTenantRequest) error {
	query := "UPDATE tenants SET "
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

	if req.Tier != nil {
		query += fmt.Sprintf("tier = $%d, ", argIdx)
		args = append(args, *req.Tier)
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
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tenant not found")
	}

	return nil
}

func (r *TenantRepository) Delete(ctx context.Context, id string) error {
	query := "DELETE FROM tenants WHERE id = $1"

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tenant not found")
	}

	return nil
}

func (r *TenantRepository) GetTotalCount(ctx context.Context) (int64, error) {
	query := "SELECT COUNT(*) FROM tenants"

	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get total count: %w", err)
	}

	return count, nil
}

func (r *TenantRepository) CreateContact(ctx context.Context, contact *tenant.TenantContact) error {
	query := `
		INSERT INTO tenant_contact (id, tenant_id, contact_type, contact_value, is_primary, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		contact.ID, contact.TenantID, contact.ContactType, contact.ContactValue, contact.IsPrimary,
	).Scan(&contact.ID, &contact.CreatedAt, &contact.UpdatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return fmt.Errorf("tenant not found")
		}
		return fmt.Errorf("failed to create contact: %w", err)
	}

	return nil
}

func (r *TenantRepository) GetContactsByTenantID(ctx context.Context, tenantID string) ([]*tenant.TenantContact, error) {
	query := `
		SELECT id, tenant_id, contact_type, contact_value, is_primary, created_at, updated_at
		FROM tenant_contact
		WHERE tenant_id = $1
		ORDER BY is_primary DESC, created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts: %w", err)
	}
	defer rows.Close()

	var contacts []*tenant.TenantContact
	for rows.Next() {
		var c tenant.TenantContact
		err := rows.Scan(
			&c.ID, &c.TenantID, &c.ContactType, &c.ContactValue, &c.IsPrimary, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan contact: %w", err)
		}
		contacts = append(contacts, &c)
	}

	return contacts, nil
}

func (r *TenantRepository) UpdateContact(ctx context.Context, id string, contact *tenant.TenantContact) error {
	query := `
		UPDATE tenant_contact
		SET contact_type = $1, contact_value = $2, is_primary = $3, updated_at = NOW()
		WHERE id = $4
	`

	result, err := r.db.ExecContext(ctx, query,
		contact.ContactType, contact.ContactValue, contact.IsPrimary, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update contact: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("contact not found")
	}

	return nil
}

func (r *TenantRepository) DeleteContact(ctx context.Context, id string) error {
	query := "DELETE FROM tenant_contact WHERE id = $1"

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete contact: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("contact not found")
	}

	return nil
}
