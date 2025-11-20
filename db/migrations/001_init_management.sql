-- Management Database Schema
-- This schema handles multi-tenant management, server capacity, API keys, rate limiting, and audit logging

-- ===========================
-- Tenants and Contact Info
-- ===========================

CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'cancelled')),
    tier VARCHAR(50) NOT NULL DEFAULT 'free' CHECK (tier IN ('free', 'basic', 'pro', 'enterprise')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_tenants_status ON tenants(status);
CREATE INDEX idx_tenants_tier ON tenants(tier);
CREATE INDEX idx_tenants_created_at ON tenants(created_at);

CREATE TABLE IF NOT EXISTS tenant_contact (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    contact_type VARCHAR(50) NOT NULL CHECK (contact_type IN ('email', 'phone', 'billing', 'technical')),
    contact_value VARCHAR(255) NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenant_contact_tenant_id ON tenant_contact(tenant_id);
CREATE INDEX idx_tenant_contact_type ON tenant_contact(tenant_id, contact_type);

-- ===========================
-- API Keys
-- ===========================

CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    key_prefix VARCHAR(20) NOT NULL,
    scopes TEXT[] NOT NULL DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked', 'expired')),
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_tenant_id ON api_keys(tenant_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_key_prefix ON api_keys(key_prefix);
CREATE INDEX idx_api_keys_status ON api_keys(status);

-- ===========================
-- Server Nodes and Slots
-- ===========================

CREATE TABLE IF NOT EXISTS server_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id VARCHAR(255) NOT NULL UNIQUE,
    region VARCHAR(100) NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    ip_address INET,
    status VARCHAR(50) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'maintenance', 'offline', 'decommissioned')),
    capacity INTEGER NOT NULL DEFAULT 100,
    current_load INTEGER NOT NULL DEFAULT 0,
    last_heartbeat TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_server_nodes_server_id ON server_nodes(server_id);
CREATE INDEX idx_server_nodes_region ON server_nodes(region);
CREATE INDEX idx_server_nodes_status ON server_nodes(status);
CREATE INDEX idx_server_nodes_last_heartbeat ON server_nodes(last_heartbeat);

CREATE TABLE IF NOT EXISTS server_slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_node_id UUID NOT NULL REFERENCES server_nodes(id) ON DELETE CASCADE,
    slot_number INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'allocated', 'reserved', 'maintenance')),
    tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    allocated_at TIMESTAMPTZ,
    released_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(server_node_id, slot_number)
);

CREATE INDEX idx_server_slots_server_node_id ON server_slots(server_node_id);
CREATE INDEX idx_server_slots_status ON server_slots(status);
CREATE INDEX idx_server_slots_tenant_id ON server_slots(tenant_id);

-- ===========================
-- Tenant-Server Assignments
-- ===========================

CREATE TABLE IF NOT EXISTS tenant_server_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    server_node_id UUID NOT NULL REFERENCES server_nodes(id) ON DELETE CASCADE,
    slot_id UUID REFERENCES server_slots(id) ON DELETE SET NULL,
    assignment_type VARCHAR(50) NOT NULL CHECK (assignment_type IN ('dedicated', 'shared')),
    status VARCHAR(50) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'migrating')),
    priority INTEGER NOT NULL DEFAULT 0,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    released_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, server_node_id, assignment_type)
);

CREATE INDEX idx_tenant_server_assignments_tenant_id ON tenant_server_assignments(tenant_id);
CREATE INDEX idx_tenant_server_assignments_server_node_id ON tenant_server_assignments(server_node_id);
CREATE INDEX idx_tenant_server_assignments_status ON tenant_server_assignments(status);
CREATE INDEX idx_tenant_server_assignments_slot_id ON tenant_server_assignments(slot_id);

-- ===========================
-- Rate Limit Counters
-- ===========================

CREATE TABLE IF NOT EXISTS rate_limit_counters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    api_key_id UUID REFERENCES api_keys(id) ON DELETE CASCADE,
    resource_type VARCHAR(100) NOT NULL,
    window_start TIMESTAMPTZ NOT NULL,
    window_duration_seconds INTEGER NOT NULL,
    request_count INTEGER NOT NULL DEFAULT 0,
    limit_value INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, api_key_id, resource_type, window_start)
);

CREATE INDEX idx_rate_limit_counters_tenant_id ON rate_limit_counters(tenant_id);
CREATE INDEX idx_rate_limit_counters_api_key_id ON rate_limit_counters(api_key_id);
CREATE INDEX idx_rate_limit_counters_window ON rate_limit_counters(window_start, window_duration_seconds);
CREATE INDEX idx_rate_limit_counters_resource_type ON rate_limit_counters(resource_type);

-- ===========================
-- Audit Logs
-- ===========================

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    api_key_id UUID REFERENCES api_keys(id) ON DELETE SET NULL,
    server_node_id UUID REFERENCES server_nodes(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255),
    actor_type VARCHAR(50) NOT NULL CHECK (actor_type IN ('user', 'system', 'api_key')),
    actor_id VARCHAR(255),
    status VARCHAR(50) NOT NULL CHECK (status IN ('success', 'failure', 'error')),
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(255),
    details JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_tenant_id ON audit_logs(tenant_id, created_at DESC);
CREATE INDEX idx_audit_logs_api_key_id ON audit_logs(api_key_id, created_at DESC);
CREATE INDEX idx_audit_logs_server_node_id ON audit_logs(server_node_id, created_at DESC);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource_type ON audit_logs(resource_type);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_request_id ON audit_logs(request_id);

-- ===========================
-- Server Metrics
-- ===========================

CREATE TABLE IF NOT EXISTS server_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_node_id UUID NOT NULL REFERENCES server_nodes(id) ON DELETE CASCADE,
    metric_type VARCHAR(100) NOT NULL,
    metric_value NUMERIC NOT NULL,
    unit VARCHAR(50),
    tags JSONB DEFAULT '{}'::jsonb,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_server_metrics_server_node_id ON server_metrics(server_node_id, recorded_at DESC);
CREATE INDEX idx_server_metrics_metric_type ON server_metrics(metric_type, recorded_at DESC);
CREATE INDEX idx_server_metrics_recorded_at ON server_metrics(recorded_at DESC);

-- ===========================
-- Billing Snapshots
-- ===========================

CREATE TABLE IF NOT EXISTS billing_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    total_requests INTEGER NOT NULL DEFAULT 0,
    total_messages_sent INTEGER NOT NULL DEFAULT 0,
    total_media_bytes BIGINT NOT NULL DEFAULT 0,
    total_compute_seconds INTEGER NOT NULL DEFAULT 0,
    cost_amount NUMERIC(10, 2),
    cost_currency VARCHAR(10) DEFAULT 'USD',
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'finalized', 'paid', 'overdue')),
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_billing_snapshots_tenant_id ON billing_snapshots(tenant_id, period_start DESC);
CREATE INDEX idx_billing_snapshots_period ON billing_snapshots(period_start, period_end);
CREATE INDEX idx_billing_snapshots_status ON billing_snapshots(status);

-- ===========================
-- Triggers for updated_at
-- ===========================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_tenant_contact_updated_at BEFORE UPDATE ON tenant_contact FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_api_keys_updated_at BEFORE UPDATE ON api_keys FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_server_nodes_updated_at BEFORE UPDATE ON server_nodes FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_server_slots_updated_at BEFORE UPDATE ON server_slots FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_tenant_server_assignments_updated_at BEFORE UPDATE ON tenant_server_assignments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_rate_limit_counters_updated_at BEFORE UPDATE ON rate_limit_counters FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_billing_snapshots_updated_at BEFORE UPDATE ON billing_snapshots FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
