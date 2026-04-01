-- Migration: 004_create_usage_records
-- Description: Create usage_records table for metering and billing

-- SQLite version
CREATE TABLE IF NOT EXISTS usage_records (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    metric_type TEXT NOT NULL,
    metric_value REAL NOT NULL,
    recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

-- Indexes for query performance
CREATE INDEX IF NOT EXISTS idx_usage_records_organization_id ON usage_records(organization_id);
CREATE INDEX IF NOT EXISTS idx_usage_records_recorded_at ON usage_records(recorded_at);
CREATE INDEX IF NOT EXISTS idx_usage_records_metric_type ON usage_records(metric_type);
CREATE INDEX IF NOT EXISTS idx_usage_records_org_metric ON usage_records(organization_id, metric_type);

-- PostgreSQL version (for production)
-- CREATE TYPE metric_type AS ENUM ('api_calls', 'storage_bytes', 'compute_hours', 'bandwidth_bytes', 'seats');
--
-- CREATE TABLE IF NOT EXISTS usage_records (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
--     metric_type metric_type NOT NULL,
--     metric_value DECIMAL(20, 6) NOT NULL,
--     recorded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
-- ) PARTITION BY RANGE (recorded_at);
--
-- -- Create monthly partitions (example for January 2024)
-- CREATE TABLE usage_records_2024_01 PARTITION OF usage_records
--     FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
--
-- CREATE TABLE usage_records_2024_02 PARTITION OF usage_records
--     FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
--
-- -- Auto-create future partitions with a scheduled job (pg_cron or similar)
-- -- Example:
-- -- SELECT cron.schedule(
-- --     'create_monthly_partition',
-- --     '0 0 1 * *',
-- --     $$
-- --     SELECT create_usage_partition_for_next_month();
-- --     $$
-- -- );
--
-- -- Indexes (created on each partition automatically)
-- CREATE INDEX idx_usage_records_organization_id ON usage_records(organization_id);
-- CREATE INDEX idx_usage_records_recorded_at ON usage_records(recorded_at);
-- CREATE INDEX idx_usage_records_metric_type ON usage_records(metric_type);
-- CREATE INDEX idx_usage_records_org_metric ON usage_records(organization_id, metric_type, recorded_at);
--
-- -- Aggregation table for faster queries on summarized data
-- CREATE TABLE IF NOT EXISTS usage_summaries (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
--     metric_type metric_type NOT NULL,
--     total_value DECIMAL(20, 6) NOT NULL,
--     period_start TIMESTAMP WITH TIME ZONE NOT NULL,
--     period_end TIMESTAMP WITH TIME ZONE NOT NULL,
--     UNIQUE(organization_id, metric_type, period_start)
-- );
--
-- CREATE INDEX idx_usage_summaries_org_period ON usage_summaries(organization_id, period_start, period_end);
