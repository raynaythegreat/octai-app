-- OctAi: Workflow Engine Migration
-- Stores workflow definitions and execution history.

CREATE TABLE IF NOT EXISTS workflows (
    id          TEXT PRIMARY KEY,
    team_id     TEXT,
    org_id      TEXT REFERENCES organizations(id) ON DELETE SET NULL,
    name        TEXT NOT NULL,
    description TEXT,
    definition  TEXT NOT NULL,   -- JSON WorkflowDefinition
    status      TEXT NOT NULL DEFAULT 'active', -- active | paused | archived
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_workflows_team_id   ON workflows(team_id);
CREATE INDEX IF NOT EXISTS idx_workflows_org_id    ON workflows(org_id);
CREATE INDEX IF NOT EXISTS idx_workflows_status    ON workflows(status);

CREATE TABLE IF NOT EXISTS workflow_runs (
    id          TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    status      TEXT NOT NULL,   -- pending | running | completed | failed | canceled
    node_states TEXT,            -- JSON map[nodeID]NodeRunState
    started_at  DATETIME,
    ended_at    DATETIME,
    error       TEXT
);

CREATE INDEX IF NOT EXISTS idx_workflow_runs_workflow  ON workflow_runs(workflow_id);
CREATE INDEX IF NOT EXISTS idx_workflow_runs_started   ON workflow_runs(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_workflow_runs_status    ON workflow_runs(status);
