-- Migration: 002_create_memberships
-- Description: Create memberships table for organization membership management

-- SQLite version
CREATE TABLE IF NOT EXISTS memberships (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member',
    invited_by TEXT,
    invited_at DATETIME,
    joined_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    UNIQUE(organization_id, user_id)
);

-- Indexes for query performance
CREATE INDEX IF NOT EXISTS idx_memberships_organization_id ON memberships(organization_id);
CREATE INDEX IF NOT EXISTS idx_memberships_user_id ON memberships(user_id);
CREATE INDEX IF NOT EXISTS idx_memberships_role ON memberships(role);

-- PostgreSQL version (for production)
-- CREATE TYPE membership_role AS ENUM ('owner', 'admin', 'member', 'viewer');
--
-- CREATE TABLE IF NOT EXISTS memberships (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
--     user_id UUID NOT NULL,
--     role membership_role NOT NULL DEFAULT 'member',
--     invited_by UUID,
--     invited_at TIMESTAMP WITH TIME ZONE,
--     joined_at TIMESTAMP WITH TIME ZONE,
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     UNIQUE(organization_id, user_id)
-- );
--
-- CREATE INDEX idx_memberships_organization_id ON memberships(organization_id);
-- CREATE INDEX idx_memberships_user_id ON memberships(user_id);
-- CREATE INDEX idx_memberships_role ON memberships(role);
--
-- -- Add foreign key to users table (assumes users table exists)
-- -- ALTER TABLE memberships ADD CONSTRAINT fk_memberships_user_id
-- --     FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
-- -- ALTER TABLE memberships ADD CONSTRAINT fk_memberships_invited_by
-- --     FOREIGN KEY (invited_by) REFERENCES users(id) ON DELETE SET NULL;
