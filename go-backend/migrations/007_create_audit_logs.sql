-- Migration: 007_create_audit_logs
-- Description: Create audit_logs table for comprehensive audit logging with GDPR compliance

-- SQLite version
CREATE TABLE IF NOT EXISTS audit_logs (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    changes TEXT,
    ip_address TEXT,
    user_agent TEXT,
    country TEXT,
    country_code TEXT,
    region TEXT,
    city TEXT,
    latitude TEXT,
    longitude TEXT,
    status TEXT NOT NULL DEFAULT 'success',
    error_message TEXT,
    metadata TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

-- Indexes for query performance
CREATE INDEX IF NOT EXISTS idx_audit_logs_organization_id ON audit_logs(organization_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_type ON audit_logs(resource_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_id ON audit_logs(resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_org_created ON audit_logs(organization_id, created_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_ip_address ON audit_logs(ip_address);

-- Audit retention settings per organization
CREATE TABLE IF NOT EXISTS audit_retention_settings (
    organization_id TEXT PRIMARY KEY,
    retention_days INTEGER NOT NULL DEFAULT 90,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

-- PostgreSQL version (for production)
-- CREATE TYPE audit_action AS ENUM (
--     'create', 'read', 'update', 'delete',
--     'login', 'logout', 'login_failed', 
--     'password_change', 'password_reset',
--     'api_key_create', 'api_key_revoke',
--     'export', 'import', 'share', 'unshare',
--     'archive', 'restore', 'access', 'configure',
--     'enable', 'disable'
-- );
-- 
-- CREATE TYPE audit_resource_type AS ENUM (
--     'organization', 'membership', 'subscription',
--     'message', 'agent', 'channel', 'session',
--     'api_key', 'user', 'billing', 'settings',
--     'skill', 'tool', 'webhook', 'integration',
--     'document', 'report', 'audit_log'
-- );
-- 
-- CREATE TYPE audit_status AS ENUM ('success', 'failure', 'denied');
-- 
-- CREATE TABLE IF NOT EXISTS audit_logs (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
--     user_id UUID NOT NULL,
--     action audit_action NOT NULL,
--     resource_type audit_resource_type NOT NULL,
--     resource_id VARCHAR(255) NOT NULL,
--     changes JSONB,
--     ip_address INET,
--     user_agent VARCHAR(500),
--     country VARCHAR(100),
--     country_code CHAR(2),
--     region VARCHAR(100),
--     city VARCHAR(100),
--     latitude DECIMAL(10, 8),
--     longitude DECIMAL(11, 8),
--     status audit_status NOT NULL DEFAULT 'success',
--     error_message TEXT,
--     metadata JSONB,
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
-- );
-- 
-- -- Primary indexes
-- CREATE INDEX idx_audit_logs_organization_id ON audit_logs(organization_id);
-- CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
-- CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
-- CREATE INDEX idx_audit_logs_org_created ON audit_logs(organization_id, created_at DESC);
-- 
-- -- Filter indexes
-- CREATE INDEX idx_audit_logs_action ON audit_logs(action);
-- CREATE INDEX idx_audit_logs_resource_type ON audit_logs(resource_type);
-- CREATE INDEX idx_audit_logs_resource_id ON audit_logs(resource_id);
-- CREATE INDEX idx_audit_logs_status ON audit_logs(status);
-- CREATE INDEX idx_audit_logs_ip_address ON audit_logs(ip_address);
-- 
-- -- Composite indexes for common query patterns
-- CREATE INDEX idx_audit_logs_org_user ON audit_logs(organization_id, user_id, created_at DESC);
-- CREATE INDEX idx_audit_logs_org_resource ON audit_logs(organization_id, resource_type, resource_id, created_at DESC);
-- 
-- -- JSONB GIN index for changes field
-- CREATE INDEX idx_audit_logs_changes ON audit_logs USING GIN (changes);
-- 
-- -- Partitioning for large-scale deployments (optional)
-- -- CREATE TABLE audit_logs (
-- --     ...
-- -- ) PARTITION BY RANGE (created_at);
-- --
-- -- CREATE TABLE audit_logs_2024_q1 PARTITION OF audit_logs
-- --     FOR VALUES FROM ('2024-01-01') TO ('2024-04-01');
-- 
-- -- Audit retention settings
-- CREATE TABLE IF NOT EXISTS audit_retention_settings (
--     organization_id UUID PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
--     retention_days INTEGER NOT NULL DEFAULT 90 CHECK (retention_days >= 30 AND retention_days <= 2555),
--     updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
-- );
-- 
-- -- Trigger for updated_at
-- CREATE TRIGGER update_audit_retention_updated_at
--     BEFORE UPDATE ON audit_retention_settings
--     FOR EACH ROW
--     EXECUTE FUNCTION update_updated_at_column();
-- 
-- -- GDPR cleanup function
-- CREATE OR REPLACE FUNCTION cleanup_audit_logs(p_org_id UUID DEFAULT NULL)
-- RETURNS BIGINT AS $$
-- DECLARE
--     v_deleted BIGINT := 0;
--     v_retention INTEGER;
--     v_org RECORD;
-- BEGIN
--     IF p_org_id IS NULL THEN
--         FOR v_org IN SELECT id FROM organizations LOOP
--             SELECT COALESCE(
--                 (SELECT retention_days FROM audit_retention_settings WHERE organization_id = v_org.id),
--                 90
--             ) INTO v_retention;
--             
--             DELETE FROM audit_logs 
--             WHERE organization_id = v_org.id 
--             AND created_at < NOW() - (v_retention || ' days')::INTERVAL;
--             
--             GET DIAGNOSTICS v_deleted = ROW_COUNT;
--         END LOOP;
--     ELSE
--         SELECT COALESCE(
--             (SELECT retention_days FROM audit_retention_settings WHERE organization_id = p_org_id),
--             90
--         ) INTO v_retention;
--         
--         DELETE FROM audit_logs 
--         WHERE organization_id = p_org_id 
--         AND created_at < NOW() - (v_retention || ' days')::INTERVAL;
--         
--         GET DIAGNOSTICS v_deleted = ROW_COUNT;
--     END IF;
--     
--     RETURN v_deleted;
-- END;
-- $$ LANGUAGE plpgsql;
-- 
-- -- Schedule cleanup job (requires pg_cron extension)
-- -- SELECT cron.schedule('cleanup_audit_logs', '0 2 * * *', 'SELECT cleanup_audit_logs()');
