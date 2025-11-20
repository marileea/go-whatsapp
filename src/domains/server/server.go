package server

import "time"

type ServerStatus string

const (
	ServerStatusActive        ServerStatus = "active"
	ServerStatusMaintenance   ServerStatus = "maintenance"
	ServerStatusOffline       ServerStatus = "offline"
	ServerStatusDecommissioned ServerStatus = "decommissioned"
)

type SlotStatus string

const (
	SlotStatusAvailable  SlotStatus = "available"
	SlotStatusAllocated  SlotStatus = "allocated"
	SlotStatusReserved   SlotStatus = "reserved"
	SlotStatusMaintenance SlotStatus = "maintenance"
)

type AssignmentType string

const (
	AssignmentTypeDedicated AssignmentType = "dedicated"
	AssignmentTypeShared    AssignmentType = "shared"
)

type AssignmentStatus string

const (
	AssignmentStatusActive    AssignmentStatus = "active"
	AssignmentStatusInactive  AssignmentStatus = "inactive"
	AssignmentStatusMigrating AssignmentStatus = "migrating"
)

type ServerNode struct {
	ID            string                 `json:"id" db:"id"`
	ServerID      string                 `json:"server_id" db:"server_id"`
	Region        string                 `json:"region" db:"region"`
	Hostname      string                 `json:"hostname" db:"hostname"`
	IPAddress     *string                `json:"ip_address" db:"ip_address"`
	Status        ServerStatus           `json:"status" db:"status"`
	Capacity      int                    `json:"capacity" db:"capacity"`
	CurrentLoad   int                    `json:"current_load" db:"current_load"`
	LastHeartbeat *time.Time             `json:"last_heartbeat" db:"last_heartbeat"`
	Metadata      map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
}

type ServerSlot struct {
	ID           string      `json:"id" db:"id"`
	ServerNodeID string      `json:"server_node_id" db:"server_node_id"`
	SlotNumber   int         `json:"slot_number" db:"slot_number"`
	Status       SlotStatus  `json:"status" db:"status"`
	TenantID     *string     `json:"tenant_id" db:"tenant_id"`
	AllocatedAt  *time.Time  `json:"allocated_at" db:"allocated_at"`
	ReleasedAt   *time.Time  `json:"released_at" db:"released_at"`
	CreatedAt    time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at" db:"updated_at"`
}

type TenantServerAssignment struct {
	ID             string           `json:"id" db:"id"`
	TenantID       string           `json:"tenant_id" db:"tenant_id"`
	ServerNodeID   string           `json:"server_node_id" db:"server_node_id"`
	SlotID         *string          `json:"slot_id" db:"slot_id"`
	AssignmentType AssignmentType   `json:"assignment_type" db:"assignment_type"`
	Status         AssignmentStatus `json:"status" db:"status"`
	Priority       int              `json:"priority" db:"priority"`
	AssignedAt     time.Time        `json:"assigned_at" db:"assigned_at"`
	ReleasedAt     *time.Time       `json:"released_at" db:"released_at"`
	CreatedAt      time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at" db:"updated_at"`
}

type ServerMetric struct {
	ID           string                 `json:"id" db:"id"`
	ServerNodeID string                 `json:"server_node_id" db:"server_node_id"`
	MetricType   string                 `json:"metric_type" db:"metric_type"`
	MetricValue  float64                `json:"metric_value" db:"metric_value"`
	Unit         *string                `json:"unit" db:"unit"`
	Tags         map[string]interface{} `json:"tags" db:"tags"`
	RecordedAt   time.Time              `json:"recorded_at" db:"recorded_at"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
}

type CreateServerNodeRequest struct {
	ServerID  string                 `json:"server_id" validate:"required"`
	Region    string                 `json:"region" validate:"required"`
	Hostname  string                 `json:"hostname" validate:"required"`
	IPAddress *string                `json:"ip_address"`
	Capacity  int                    `json:"capacity" validate:"required,min=1"`
	Metadata  map[string]interface{} `json:"metadata"`
}

type UpdateServerNodeRequest struct {
	Status        *ServerStatus          `json:"status"`
	Capacity      *int                   `json:"capacity"`
	CurrentLoad   *int                   `json:"current_load"`
	LastHeartbeat *time.Time             `json:"last_heartbeat"`
	Metadata      *map[string]interface{} `json:"metadata"`
}

type AllocateSlotRequest struct {
	TenantID       string         `json:"tenant_id" validate:"required,uuid"`
	ServerNodeID   string         `json:"server_node_id" validate:"required,uuid"`
	AssignmentType AssignmentType `json:"assignment_type" validate:"required,oneof=dedicated shared"`
	Priority       int            `json:"priority"`
}

type RecordMetricRequest struct {
	ServerNodeID string                 `json:"server_node_id" validate:"required,uuid"`
	MetricType   string                 `json:"metric_type" validate:"required"`
	MetricValue  float64                `json:"metric_value" validate:"required"`
	Unit         *string                `json:"unit"`
	Tags         map[string]interface{} `json:"tags"`
	RecordedAt   *time.Time             `json:"recorded_at"`
}
