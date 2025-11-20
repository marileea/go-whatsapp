package tenant

import "context"

type ITenantRepository interface {
	Create(ctx context.Context, tenant *Tenant) error
	GetByID(ctx context.Context, id string) (*Tenant, error)
	List(ctx context.Context, limit, offset int) ([]*Tenant, error)
	Update(ctx context.Context, id string, req *UpdateTenantRequest) error
	Delete(ctx context.Context, id string) error
	GetTotalCount(ctx context.Context) (int64, error)

	CreateContact(ctx context.Context, contact *TenantContact) error
	GetContactsByTenantID(ctx context.Context, tenantID string) ([]*TenantContact, error)
	UpdateContact(ctx context.Context, id string, contact *TenantContact) error
	DeleteContact(ctx context.Context, id string) error
}
