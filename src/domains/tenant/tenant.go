package tenant

import "time"

type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusCancelled TenantStatus = "cancelled"
)

type TenantTier string

const (
	TenantTierFree       TenantTier = "free"
	TenantTierBasic      TenantTier = "basic"
	TenantTierPro        TenantTier = "pro"
	TenantTierEnterprise TenantTier = "enterprise"
)

type ContactType string

const (
	ContactTypeEmail     ContactType = "email"
	ContactTypePhone     ContactType = "phone"
	ContactTypeBilling   ContactType = "billing"
	ContactTypeTechnical ContactType = "technical"
)

type Tenant struct {
	ID        string                 `json:"id" db:"id"`
	Name      string                 `json:"name" db:"name"`
	Status    TenantStatus           `json:"status" db:"status"`
	Tier      TenantTier             `json:"tier" db:"tier"`
	Metadata  map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" db:"updated_at"`
}

type TenantContact struct {
	ID           string      `json:"id" db:"id"`
	TenantID     string      `json:"tenant_id" db:"tenant_id"`
	ContactType  ContactType `json:"contact_type" db:"contact_type"`
	ContactValue string      `json:"contact_value" db:"contact_value"`
	IsPrimary    bool        `json:"is_primary" db:"is_primary"`
	CreatedAt    time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at" db:"updated_at"`
}

type CreateTenantRequest struct {
	Name     string                 `json:"name" validate:"required"`
	Tier     TenantTier             `json:"tier" validate:"required,oneof=free basic pro enterprise"`
	Metadata map[string]interface{} `json:"metadata"`
}

type UpdateTenantRequest struct {
	Name     *string                 `json:"name"`
	Status   *TenantStatus           `json:"status"`
	Tier     *TenantTier             `json:"tier"`
	Metadata *map[string]interface{} `json:"metadata"`
}

type CreateContactRequest struct {
	TenantID     string      `json:"tenant_id" validate:"required,uuid"`
	ContactType  ContactType `json:"contact_type" validate:"required,oneof=email phone billing technical"`
	ContactValue string      `json:"contact_value" validate:"required"`
	IsPrimary    bool        `json:"is_primary"`
}
