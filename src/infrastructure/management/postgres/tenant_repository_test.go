package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/domains/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTenantTest(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *TenantRepository) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewTenantRepository(db)
	return db, mock, repo
}

func TestTenantRepository_Create(t *testing.T) {
	db, mock, repo := setupTenantTest(t)
	defer db.Close()

	ctx := context.Background()
	now := time.Now()

	testTenant := &tenant.Tenant{
		ID:       "123e4567-e89b-12d3-a456-426614174000",
		Name:     "Test Tenant",
		Status:   tenant.TenantStatusActive,
		Tier:     tenant.TenantTierPro,
		Metadata: map[string]interface{}{"key": "value"},
	}

	mock.ExpectQuery(`INSERT INTO tenants`).
		WithArgs(testTenant.ID, testTenant.Name, testTenant.Status, testTenant.Tier, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(testTenant.ID, now, now))

	err := repo.Create(ctx, testTenant)
	assert.NoError(t, err)
	assert.Equal(t, now.Unix(), testTenant.CreatedAt.Unix())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantRepository_GetByID(t *testing.T) {
	db, mock, repo := setupTenantTest(t)
	defer db.Close()

	ctx := context.Background()
	now := time.Now()
	tenantID := "123e4567-e89b-12d3-a456-426614174000"

	mock.ExpectQuery(`SELECT .+ FROM tenants WHERE id = \$1`).
		WithArgs(tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status", "tier", "metadata", "created_at", "updated_at"}).
			AddRow(tenantID, "Test Tenant", tenant.TenantStatusActive, tenant.TenantTierPro, []byte(`{"key":"value"}`), now, now))

	result, err := repo.GetByID(ctx, tenantID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, tenantID, result.ID)
	assert.Equal(t, "Test Tenant", result.Name)
	assert.Equal(t, tenant.TenantStatusActive, result.Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantRepository_List(t *testing.T) {
	db, mock, repo := setupTenantTest(t)
	defer db.Close()

	ctx := context.Background()
	now := time.Now()

	mock.ExpectQuery(`SELECT .+ FROM tenants ORDER BY created_at DESC LIMIT \$1 OFFSET \$2`).
		WithArgs(10, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status", "tier", "metadata", "created_at", "updated_at"}).
			AddRow("id1", "Tenant 1", tenant.TenantStatusActive, tenant.TenantTierPro, []byte(`{}`), now, now).
			AddRow("id2", "Tenant 2", tenant.TenantStatusActive, tenant.TenantTierBasic, []byte(`{}`), now, now))

	results, err := repo.List(ctx, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "Tenant 1", results[0].Name)
	assert.Equal(t, "Tenant 2", results[1].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantRepository_Update(t *testing.T) {
	db, mock, repo := setupTenantTest(t)
	defer db.Close()

	ctx := context.Background()
	tenantID := "123e4567-e89b-12d3-a456-426614174000"
	newName := "Updated Tenant"

	req := &tenant.UpdateTenantRequest{
		Name: &newName,
	}

	mock.ExpectExec(`UPDATE tenants SET name = \$1, updated_at = NOW\(\) WHERE id = \$2`).
		WithArgs(newName, tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Update(ctx, tenantID, req)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantRepository_Delete(t *testing.T) {
	db, mock, repo := setupTenantTest(t)
	defer db.Close()

	ctx := context.Background()
	tenantID := "123e4567-e89b-12d3-a456-426614174000"

	mock.ExpectExec(`DELETE FROM tenants WHERE id = \$1`).
		WithArgs(tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(ctx, tenantID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantRepository_CreateContact(t *testing.T) {
	db, mock, repo := setupTenantTest(t)
	defer db.Close()

	ctx := context.Background()
	now := time.Now()

	contact := &tenant.TenantContact{
		ID:           "contact-id",
		TenantID:     "tenant-id",
		ContactType:  tenant.ContactTypeEmail,
		ContactValue: "test@example.com",
		IsPrimary:    true,
	}

	mock.ExpectQuery(`INSERT INTO tenant_contact`).
		WithArgs(contact.ID, contact.TenantID, contact.ContactType, contact.ContactValue, contact.IsPrimary).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(contact.ID, now, now))

	err := repo.CreateContact(ctx, contact)
	assert.NoError(t, err)
	assert.Equal(t, now.Unix(), contact.CreatedAt.Unix())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantRepository_GetContactsByTenantID(t *testing.T) {
	db, mock, repo := setupTenantTest(t)
	defer db.Close()

	ctx := context.Background()
	now := time.Now()
	tenantID := "tenant-id"

	mock.ExpectQuery(`SELECT .+ FROM tenant_contact WHERE tenant_id = \$1`).
		WithArgs(tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "contact_type", "contact_value", "is_primary", "created_at", "updated_at"}).
			AddRow("contact1", tenantID, tenant.ContactTypeEmail, "test@example.com", true, now, now))

	results, err := repo.GetContactsByTenantID(ctx, tenantID)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "test@example.com", results[0].ContactValue)
	assert.NoError(t, mock.ExpectationsWereMet())
}
