-- Rollback script for management database schema

DROP TRIGGER IF EXISTS update_billing_snapshots_updated_at ON billing_snapshots;
DROP TRIGGER IF EXISTS update_rate_limit_counters_updated_at ON rate_limit_counters;
DROP TRIGGER IF EXISTS update_tenant_server_assignments_updated_at ON tenant_server_assignments;
DROP TRIGGER IF EXISTS update_server_slots_updated_at ON server_slots;
DROP TRIGGER IF EXISTS update_server_nodes_updated_at ON server_nodes;
DROP TRIGGER IF EXISTS update_api_keys_updated_at ON api_keys;
DROP TRIGGER IF EXISTS update_tenant_contact_updated_at ON tenant_contact;
DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS billing_snapshots;
DROP TABLE IF EXISTS server_metrics;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS rate_limit_counters;
DROP TABLE IF EXISTS tenant_server_assignments;
DROP TABLE IF EXISTS server_slots;
DROP TABLE IF EXISTS server_nodes;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS tenant_contact;
DROP TABLE IF EXISTS tenants;
