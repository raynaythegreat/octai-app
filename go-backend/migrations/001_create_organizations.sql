-- Migration: 001_create_organizations
-- Description: Create organizations table for multi-tenant SaaS

-- SQLite version
CREATE TABLE IF NOT EXISTS organizations (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for query performance
CREATE INDEX IF NOT EXISTS idx_organizations_slug ON organizations(slug);
CREATE INDEX IF NOT EXISTS idx_organizations_created_at ON organizations(created_at);

-- PostgreSQL version (for production)
-- CREATE TABLE IF NOT EXISTS organizations (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     name VARCHAR(255) NOT NULL,
--     slug VARCHAR(100) NOT NULL UNIQUE,
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
-- );
-- 
-- CREATE INDEX idx_organizations_slug ON organizations(slug);
-- CREATE INDEX idx_organizations_created_at ON organizations(created_at DESC);
--
-- -- Trigger for updated_at
-- CREATE OR REPLACE FUNCTION update_updated_at_column()
-- RETURNS TRIGGER AS $$
-- BEGIN
--     NEW.updated_at = NOW();
--     RETURN NEW;
-- END;
-- $$ LANGUAGE plpgsql;
--
-- CREATE TRIGGER update_organizations_updated_at
--     BEFORE UPDATE ON organizations
--     FOR EACH ROW
--     EXECUTE FUNCTION update_updated_at_column();
