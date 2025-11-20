package server

import (
	"context"
	"time"
)

type IServerRepository interface {
	CreateNode(ctx context.Context, node *ServerNode) error
	GetNodeByID(ctx context.Context, id string) (*ServerNode, error)
	GetNodeByServerID(ctx context.Context, serverID string) (*ServerNode, error)
	ListNodes(ctx context.Context, limit, offset int) ([]*ServerNode, error)
	UpdateNode(ctx context.Context, id string, req *UpdateServerNodeRequest) error
	DeleteNode(ctx context.Context, id string) error
	UpdateHeartbeat(ctx context.Context, serverID string) error

	CreateSlot(ctx context.Context, slot *ServerSlot) error
	GetSlotByID(ctx context.Context, id string) (*ServerSlot, error)
	GetSlotsByServerNodeID(ctx context.Context, serverNodeID string) ([]*ServerSlot, error)
	GetAvailableSlots(ctx context.Context, serverNodeID string) ([]*ServerSlot, error)
	AllocateSlot(ctx context.Context, slotID, tenantID string) error
	ReleaseSlot(ctx context.Context, slotID string) error
	UpdateSlotStatus(ctx context.Context, slotID string, status SlotStatus) error

	CreateAssignment(ctx context.Context, assignment *TenantServerAssignment) error
	GetAssignmentByID(ctx context.Context, id string) (*TenantServerAssignment, error)
	GetAssignmentsByTenantID(ctx context.Context, tenantID string) ([]*TenantServerAssignment, error)
	GetAssignmentsByServerNodeID(ctx context.Context, serverNodeID string) ([]*TenantServerAssignment, error)
	UpdateAssignmentStatus(ctx context.Context, id string, status AssignmentStatus) error
	ReleaseAssignment(ctx context.Context, id string) error

	RecordMetric(ctx context.Context, metric *ServerMetric) error
	GetMetrics(ctx context.Context, serverNodeID string, metricType *string, startTime, endTime time.Time) ([]*ServerMetric, error)
	GetLatestMetrics(ctx context.Context, serverNodeID string, limit int) ([]*ServerMetric, error)
}
