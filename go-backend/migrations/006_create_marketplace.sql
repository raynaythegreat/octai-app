-- Migration: 006_create_marketplace
-- Description: Create marketplace tables for skill listings, reviews, purchases, and versions

-- SQLite version

CREATE TABLE IF NOT EXISTS skill_listings (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    author TEXT NOT NULL,
    version TEXT NOT NULL DEFAULT '1.0.0',
    category TEXT NOT NULL DEFAULT 'utility',
    tags TEXT NOT NULL DEFAULT '[]',
    price REAL NOT NULL DEFAULT 0,
    rating REAL NOT NULL DEFAULT 0,
    downloads INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS skill_reviews (
    id TEXT PRIMARY KEY,
    skill_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    rating INTEGER NOT NULL CHECK(rating >= 1 AND rating <= 5),
    comment TEXT NOT NULL DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (skill_id) REFERENCES skill_listings(id) ON DELETE CASCADE,
    UNIQUE(skill_id, user_id)
);

CREATE TABLE IF NOT EXISTS skill_purchases (
    id TEXT PRIMARY KEY,
    skill_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    purchased_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (skill_id) REFERENCES skill_listings(id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    UNIQUE(skill_id, organization_id)
);

CREATE TABLE IF NOT EXISTS skill_versions (
    id TEXT PRIMARY KEY,
    skill_id TEXT NOT NULL,
    version TEXT NOT NULL,
    changelog TEXT NOT NULL DEFAULT '',
    download_url TEXT NOT NULL DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (skill_id) REFERENCES skill_listings(id) ON DELETE CASCADE,
    UNIQUE(skill_id, version)
);

-- Indexes for skill_listings
CREATE INDEX IF NOT EXISTS idx_skill_listings_category ON skill_listings(category);
CREATE INDEX IF NOT EXISTS idx_skill_listings_author ON skill_listings(author);
CREATE INDEX IF NOT EXISTS idx_skill_listings_rating ON skill_listings(rating DESC);
CREATE INDEX IF NOT EXISTS idx_skill_listings_downloads ON skill_listings(downloads DESC);
CREATE INDEX IF NOT EXISTS idx_skill_listings_price ON skill_listings(price);
CREATE INDEX IF NOT EXISTS idx_skill_listings_created_at ON skill_listings(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_skill_listings_name ON skill_listings(name);

-- Indexes for skill_reviews
CREATE INDEX IF NOT EXISTS idx_skill_reviews_skill_id ON skill_reviews(skill_id);
CREATE INDEX IF NOT EXISTS idx_skill_reviews_user_id ON skill_reviews(user_id);
CREATE INDEX IF NOT EXISTS idx_skill_reviews_rating ON skill_reviews(rating);

-- Indexes for skill_purchases
CREATE INDEX IF NOT EXISTS idx_skill_purchases_skill_id ON skill_purchases(skill_id);
CREATE INDEX IF NOT EXISTS idx_skill_purchases_org_id ON skill_purchases(organization_id);
CREATE INDEX IF NOT EXISTS idx_skill_purchases_purchased_at ON skill_purchases(purchased_at DESC);

-- Indexes for skill_versions
CREATE INDEX IF NOT EXISTS idx_skill_versions_skill_id ON skill_versions(skill_id);
CREATE INDEX IF NOT EXISTS idx_skill_versions_created_at ON skill_versions(created_at DESC);

-- Full-text search index for SQLite (using FTS5 virtual table)
CREATE VIRTUAL TABLE IF NOT EXISTS skill_listings_fts USING fts5(
    name,
    description,
    author,
    tags,
    content=skill_listings,
    content_rowid=rowid
);

-- Triggers to keep FTS index in sync
CREATE TRIGGER IF NOT EXISTS skill_listings_ai AFTER INSERT ON skill_listings BEGIN
    INSERT INTO skill_listings_fts(rowid, name, description, author, tags)
    VALUES (NEW.rowid, NEW.name, NEW.description, NEW.author, NEW.tags);
END;

CREATE TRIGGER IF NOT EXISTS skill_listings_ad AFTER DELETE ON skill_listings BEGIN
    INSERT INTO skill_listings_fts(skill_listings_fts, rowid, name, description, author, tags)
    VALUES('delete', OLD.rowid, OLD.name, OLD.description, OLD.author, OLD.tags);
END;

CREATE TRIGGER IF NOT EXISTS skill_listings_au AFTER UPDATE ON skill_listings BEGIN
    INSERT INTO skill_listings_fts(skill_listings_fts, rowid, name, description, author, tags)
    VALUES('delete', OLD.rowid, OLD.name, OLD.description, OLD.author, OLD.tags);
    INSERT INTO skill_listings_fts(rowid, name, description, author, tags)
    VALUES (NEW.rowid, NEW.name, NEW.description, NEW.author, NEW.tags);
END;

-- PostgreSQL version (for production)
-- CREATE TYPE skill_category AS ENUM ('automation', 'communication', 'data', 'integration', 'ai', 'utility');
--
-- CREATE TABLE IF NOT EXISTS skill_listings (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     name VARCHAR(255) NOT NULL,
--     description TEXT NOT NULL DEFAULT '',
--     author VARCHAR(255) NOT NULL,
--     version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
--     category skill_category NOT NULL DEFAULT 'utility',
--     tags TEXT[] NOT NULL DEFAULT '{}',
--     price DECIMAL(10, 2) NOT NULL DEFAULT 0 CHECK(price >= 0),
--     rating DECIMAL(3, 2) NOT NULL DEFAULT 0 CHECK(rating >= 0 AND rating <= 5),
--     downloads BIGINT NOT NULL DEFAULT 0,
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
-- );
--
-- CREATE TABLE IF NOT EXISTS skill_reviews (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     skill_id UUID NOT NULL REFERENCES skill_listings(id) ON DELETE CASCADE,
--     user_id UUID NOT NULL,
--     rating SMALLINT NOT NULL CHECK(rating >= 1 AND rating <= 5),
--     comment TEXT NOT NULL DEFAULT '',
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     UNIQUE(skill_id, user_id)
-- );
--
-- CREATE TABLE IF NOT EXISTS skill_purchases (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     skill_id UUID NOT NULL REFERENCES skill_listings(id) ON DELETE CASCADE,
--     organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
--     purchased_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     UNIQUE(skill_id, organization_id)
-- );
--
-- CREATE TABLE IF NOT EXISTS skill_versions (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     skill_id UUID NOT NULL REFERENCES skill_listings(id) ON DELETE CASCADE,
--     version VARCHAR(50) NOT NULL,
--     changelog TEXT NOT NULL DEFAULT '',
--     download_url TEXT NOT NULL DEFAULT '',
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     UNIQUE(skill_id, version)
-- );
--
-- -- Indexes for skill_listings
-- CREATE INDEX idx_skill_listings_category ON skill_listings(category);
-- CREATE INDEX idx_skill_listings_author ON skill_listings(author);
-- CREATE INDEX idx_skill_listings_rating ON skill_listings(rating DESC);
-- CREATE INDEX idx_skill_listings_downloads ON skill_listings(downloads DESC);
-- CREATE INDEX idx_skill_listings_price ON skill_listings(price);
-- CREATE INDEX idx_skill_listings_created_at ON skill_listings(created_at DESC);
-- CREATE INDEX idx_skill_listings_name ON skill_listings(name);
--
-- -- Full-text search index for PostgreSQL (using GIN)
-- CREATE INDEX idx_skill_listings_search ON skill_listings 
--     USING GIN(to_tsvector('english', name || ' ' || description || ' ' || author));
--
-- -- Indexes for skill_reviews
-- CREATE INDEX idx_skill_reviews_skill_id ON skill_reviews(skill_id);
-- CREATE INDEX idx_skill_reviews_user_id ON skill_reviews(user_id);
-- CREATE INDEX idx_skill_reviews_rating ON skill_reviews(rating);
--
-- -- Indexes for skill_purchases
-- CREATE INDEX idx_skill_purchases_skill_id ON skill_purchases(skill_id);
-- CREATE INDEX idx_skill_purchases_org_id ON skill_purchases(organization_id);
-- CREATE INDEX idx_skill_purchases_purchased_at ON skill_purchases(purchased_at DESC);
--
-- -- Indexes for skill_versions
-- CREATE INDEX idx_skill_versions_skill_id ON skill_versions(skill_id);
-- CREATE INDEX idx_skill_versions_created_at ON skill_versions(created_at DESC);
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
-- CREATE TRIGGER update_skill_listings_updated_at
--     BEFORE UPDATE ON skill_listings
--     FOR EACH ROW
--     EXECUTE FUNCTION update_updated_at_column();
--
-- CREATE TRIGGER update_skill_reviews_updated_at
--     BEFORE UPDATE ON skill_reviews
--     FOR EACH ROW
--     EXECUTE FUNCTION update_updated_at_column();
