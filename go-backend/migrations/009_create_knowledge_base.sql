-- OctAi: Knowledge Base Migration
-- Creates the team-shared knowledge base tables for RAG-powered retrieval.
-- SQLite mode: uses FTS5 for BM25 text search.
-- PostgreSQL mode: extend with pgvector for semantic search.

CREATE TABLE IF NOT EXISTS kb_documents (
    id          TEXT PRIMARY KEY,
    team_id     TEXT NOT NULL,
    title       TEXT NOT NULL,
    content     TEXT NOT NULL,
    source_url  TEXT,
    metadata    TEXT,   -- JSON key-value pairs
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_kb_documents_team_id ON kb_documents(team_id);
CREATE INDEX IF NOT EXISTS idx_kb_documents_updated_at ON kb_documents(updated_at DESC);

CREATE TABLE IF NOT EXISTS kb_chunks (
    id          TEXT PRIMARY KEY,
    document_id TEXT NOT NULL REFERENCES kb_documents(id) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    chunk_index INTEGER NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_kb_chunks_document_id ON kb_chunks(document_id);

-- FTS5 virtual table for BM25 full-text search (SQLite only).
-- For PostgreSQL, replace with tsvector + GiST index or pg_trgm.
CREATE VIRTUAL TABLE IF NOT EXISTS kb_chunks_fts USING fts5(
    content,
    document_id UNINDEXED,
    chunk_id    UNINDEXED,
    title       UNINDEXED,
    tokenize    = 'porter unicode61'
);

-- Teams table for team-level knowledge base configuration.
CREATE TABLE IF NOT EXISTS kb_teams (
    team_id       TEXT PRIMARY KEY,
    kb_path       TEXT,       -- filesystem path for file-backed storage
    settings      TEXT,       -- JSON settings
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
