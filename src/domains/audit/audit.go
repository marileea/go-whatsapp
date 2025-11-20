package audit

import "time"

type ActorType string

const (
	ActorTypeUser   ActorType = "user"
	ActorTypeSystem ActorType = "system"
	ActorTypeAPIKey ActorType = "api_key"
)

type AuditStatus string

const (
	AuditStatusSuccess AuditStatus = "success"
	AuditStatusFailure AuditStatus = "failure"
	AuditStatusError   AuditStatus = "error"
)

type AuditLog struct {
	ID           string                 `json:"id" db:"id"`
	TenantID     *string                `json:"tenant_id" db:"tenant_id"`
	APIKeyID     *string                `json:"api_key_id" db:"api_key_id"`
	ServerNodeID *string                `json:"server_node_id" db:"server_node_id"`
	Action       string                 `json:"action" db:"action"`
	ResourceType string                 `json:"resource_type" db:"resource_type"`
	ResourceID   *string                `json:"resource_id" db:"resource_id"`
	ActorType    ActorType              `json:"actor_type" db:"actor_type"`
	ActorID      *string                `json:"actor_id" db:"actor_id"`
	Status       AuditStatus            `json:"status" db:"status"`
	IPAddress    *string                `json:"ip_address" db:"ip_address"`
	UserAgent    *string                `json:"user_agent" db:"user_agent"`
	RequestID    *string                `json:"request_id" db:"request_id"`
	Details      map[string]interface{} `json:"details" db:"details"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
}

type BillingSnapshot struct {
	ID                  string                 `json:"id" db:"id"`
	TenantID            string                 `json:"tenant_id" db:"tenant_id"`
	PeriodStart         time.Time              `json:"period_start" db:"period_start"`
	PeriodEnd           time.Time              `json:"period_end" db:"period_end"`
	TotalRequests       int                    `json:"total_requests" db:"total_requests"`
	TotalMessagesSent   int                    `json:"total_messages_sent" db:"total_messages_sent"`
	TotalMediaBytes     int64                  `json:"total_media_bytes" db:"total_media_bytes"`
	TotalComputeSeconds int                    `json:"total_compute_seconds" db:"total_compute_seconds"`
	CostAmount          *float64               `json:"cost_amount" db:"cost_amount"`
	CostCurrency        string                 `json:"cost_currency" db:"cost_currency"`
	Status              string                 `json:"status" db:"status"`
	Metadata            map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt           time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at" db:"updated_at"`
}

type CreateAuditLogRequest struct {
	TenantID     *string                `json:"tenant_id"`
	APIKeyID     *string                `json:"api_key_id"`
	ServerNodeID *string                `json:"server_node_id"`
	Action       string                 `json:"action" validate:"required"`
	ResourceType string                 `json:"resource_type" validate:"required"`
	ResourceID   *string                `json:"resource_id"`
	ActorType    ActorType              `json:"actor_type" validate:"required,oneof=user system api_key"`
	ActorID      *string                `json:"actor_id"`
	Status       AuditStatus            `json:"status" validate:"required,oneof=success failure error"`
	IPAddress    *string                `json:"ip_address"`
	UserAgent    *string                `json:"user_agent"`
	RequestID    *string                `json:"request_id"`
	Details      map[string]interface{} `json:"details"`
}

type AuditLogFilter struct {
	TenantID     *string
	APIKeyID     *string
	ServerNodeID *string
	Action       *string
	ResourceType *string
	ActorType    *ActorType
	Status       *AuditStatus
	StartTime    *time.Time
	EndTime      *time.Time
	Limit        int
	Offset       int
}

type CreateBillingSnapshotRequest struct {
	TenantID            string                 `json:"tenant_id" validate:"required,uuid"`
	PeriodStart         time.Time              `json:"period_start" validate:"required"`
	PeriodEnd           time.Time              `json:"period_end" validate:"required"`
	TotalRequests       int                    `json:"total_requests"`
	TotalMessagesSent   int                    `json:"total_messages_sent"`
	TotalMediaBytes     int64                  `json:"total_media_bytes"`
	TotalComputeSeconds int                    `json:"total_compute_seconds"`
	CostAmount          *float64               `json:"cost_amount"`
	CostCurrency        string                 `json:"cost_currency"`
	Metadata            map[string]interface{} `json:"metadata"`
}
