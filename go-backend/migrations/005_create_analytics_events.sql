-- Migration: 005_create_analytics_events
-- Description: Create analytics_events table for event tracking and reporting

-- SQLite version
CREATE TABLE IF NOT EXISTS analytics_events (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    event_data TEXT, -- JSON stored as text in SQLite
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

-- Indexes for query performance
CREATE INDEX IF NOT EXISTS idx_analytics_events_organization_id ON analytics_events(organization_id);
CREATE INDEX IF NOT EXISTS idx_analytics_events_event_type ON analytics_events(event_type);
CREATE INDEX IF NOT EXISTS idx_analytics_events_created_at ON analytics_events(created_at);
CREATE INDEX IF NOT EXISTS idx_analytics_events_org_type ON analytics_events(organization_id, event_type);

-- PostgreSQL version (for production)
-- CREATE TABLE IF NOT EXISTS analytics_events (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
--     event_type VARCHAR(100) NOT NULL,
--     event_data JSONB,
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
-- );
--
-- -- Standard indexes
-- CREATE INDEX idx_analytics_events_organization_id ON analytics_events(organization_id);
-- CREATE INDEX idx_analytics_events_event_type ON analytics_events(event_type);
-- CREATE INDEX idx_analytics_events_created_at ON analytics_events(created_at DESC);
-- CREATE INDEX idx_analytics_events_org_type ON analytics_events(organization_id, event_type);
--
-- -- JSONB GIN index for querying event_data contents
-- CREATE INDEX idx_analytics_events_event_data ON analytics_events USING GIN (event_data);
--
-- -- Partial indexes for common event types
-- CREATE INDEX idx_analytics_events_page_views ON analytics_events(organization_id, created_at)
--     WHERE event_type = 'page_view';
--
-- CREATE INDEX idx_analytics_events_api_calls ON analytics_events(organization_id, created_at)
--     WHERE event_type = 'api_call';
--
-- -- Expression index for extracting common JSON fields
-- CREATE INDEX idx_analytics_events_user_id ON analytics_events((event_data->>'user_id'));
-- CREATE INDEX idx_analytics_events_session_id ON analytics_events((event_data->>'session_id'));
--
-- -- TimescaleDB hypertable extension (if available)
-- -- SELECT create_hypertable('analytics_events', 'created_at');
--
-- -- Continuous aggregates for common aggregations (TimescaleDB)
-- -- CREATE MATERIALIZED VIEW analytics_hourly
-- -- WITH (timescaledb.continuous) AS
-- -- SELECT
-- --     organization_id,
-- --     event_type,
-- --     time_bucket('1 hour', created_at) AS bucket,
-- --     COUNT(*) AS event_count
-- -- FROM analytics_events
-- -- GROUP BY organization_id, event_type, bucket;
