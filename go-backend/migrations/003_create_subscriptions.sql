-- Migration: 003_create_subscriptions
-- Description: Create subscriptions table for billing and plan management

-- SQLite version
CREATE TABLE IF NOT EXISTS subscriptions (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL UNIQUE,
    plan_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    stripe_customer_id TEXT,
    stripe_subscription_id TEXT UNIQUE,
    current_period_start DATETIME,
    current_period_end DATETIME,
    cancel_at_period_end INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

-- Indexes for query performance
CREATE INDEX IF NOT EXISTS idx_subscriptions_organization_id ON subscriptions(organization_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);
CREATE INDEX IF NOT EXISTS idx_subscriptions_stripe_customer_id ON subscriptions(stripe_customer_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_stripe_subscription_id ON subscriptions(stripe_subscription_id);

-- PostgreSQL version (for production)
-- CREATE TYPE subscription_status AS ENUM ('active', 'past_due', 'canceled', 'incomplete', 'trialing');
-- CREATE TYPE plan_id AS ENUM ('free', 'starter', 'pro', 'enterprise');
--
-- CREATE TABLE IF NOT EXISTS subscriptions (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     organization_id UUID NOT NULL UNIQUE REFERENCES organizations(id) ON DELETE CASCADE,
--     plan_id plan_id NOT NULL,
--     status subscription_status NOT NULL DEFAULT 'active',
--     stripe_customer_id VARCHAR(255),
--     stripe_subscription_id VARCHAR(255) UNIQUE,
--     current_period_start TIMESTAMP WITH TIME ZONE,
--     current_period_end TIMESTAMP WITH TIME ZONE,
--     cancel_at_period_end BOOLEAN DEFAULT FALSE,
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
-- );
--
-- CREATE INDEX idx_subscriptions_organization_id ON subscriptions(organization_id);
-- CREATE INDEX idx_subscriptions_status ON subscriptions(status);
-- CREATE INDEX idx_subscriptions_stripe_customer_id ON subscriptions(stripe_customer_id);
-- CREATE INDEX idx_subscriptions_stripe_subscription_id ON subscriptions(stripe_subscription_id);
-- CREATE INDEX idx_subscriptions_current_period_end ON subscriptions(current_period_end);
--
-- -- Trigger for updated_at (reuses function from 001_create_organizations.sql)
-- CREATE TRIGGER update_subscriptions_updated_at
--     BEFORE UPDATE ON subscriptions
--     FOR EACH ROW
--     EXECUTE FUNCTION update_updated_at_column();
