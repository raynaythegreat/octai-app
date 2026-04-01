-- Migration: 008_create_compliance
-- Description: Create compliance rules and check results tables

-- SQLite version

CREATE TABLE IF NOT EXISTS compliance_rules (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'warning',
    config TEXT,
    frameworks TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_type CHECK (type IN ('data_retention', 'access_control', 'encryption', 'password_policy', 'data_location')),
    CONSTRAINT valid_severity CHECK (severity IN ('info', 'warning', 'critical'))
);

CREATE TABLE IF NOT EXISTS compliance_check_results (
    id TEXT PRIMARY KEY,
    rule_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    status TEXT NOT NULL,
    findings TEXT,
    score INTEGER DEFAULT 0,
    max_score INTEGER DEFAULT 100,
    checked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    resolved_at DATETIME,
    resolution_notes TEXT,
    
    CONSTRAINT valid_status CHECK (status IN ('pass', 'fail', 'warning', 'skipped')),
    FOREIGN KEY (rule_id) REFERENCES compliance_rules(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS compliance_reports (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    generated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    summary TEXT NOT NULL,
    frameworks TEXT,
    period_start DATETIME,
    period_end DATETIME,
    
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS compliance_report_results (
    report_id TEXT NOT NULL,
    result_id TEXT NOT NULL,
    PRIMARY KEY (report_id, result_id),
    FOREIGN KEY (report_id) REFERENCES compliance_reports(id) ON DELETE CASCADE,
    FOREIGN KEY (result_id) REFERENCES compliance_check_results(id) ON DELETE CASCADE
);

-- Indexes for fast querying
CREATE INDEX IF NOT EXISTS idx_compliance_rules_org ON compliance_rules(organization_id);
CREATE INDEX IF NOT EXISTS idx_compliance_rules_type ON compliance_rules(type);
CREATE INDEX IF NOT EXISTS idx_compliance_rules_severity ON compliance_rules(severity);

CREATE INDEX IF NOT EXISTS idx_compliance_results_org ON compliance_check_results(organization_id);
CREATE INDEX IF NOT EXISTS idx_compliance_results_rule ON compliance_check_results(rule_id);
CREATE INDEX IF NOT EXISTS idx_compliance_results_status ON compliance_check_results(status);
CREATE INDEX IF NOT EXISTS idx_compliance_results_checked ON compliance_check_results(checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_compliance_results_resolved ON compliance_check_results(resolved_at);

CREATE INDEX IF NOT EXISTS idx_compliance_reports_org ON compliance_reports(organization_id);
CREATE INDEX IF NOT EXISTS idx_compliance_reports_generated ON compliance_reports(generated_at DESC);

-- PostgreSQL version (for production)
-- CREATE TABLE IF NOT EXISTS compliance_rules (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
--     name VARCHAR(255) NOT NULL,
--     description TEXT,
--     type VARCHAR(50) NOT NULL,
--     severity VARCHAR(20) NOT NULL DEFAULT 'warning',
--     config JSONB,
--     frameworks TEXT[],
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     
--     CONSTRAINT valid_type CHECK (type IN ('data_retention', 'access_control', 'encryption', 'password_policy', 'data_location')),
--     CONSTRAINT valid_severity CHECK (severity IN ('info', 'warning', 'critical'))
-- );
-- 
-- CREATE TABLE IF NOT EXISTS compliance_check_results (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     rule_id UUID NOT NULL REFERENCES compliance_rules(id) ON DELETE CASCADE,
--     organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
--     status VARCHAR(20) NOT NULL,
--     findings JSONB,
--     score INTEGER DEFAULT 0,
--     max_score INTEGER DEFAULT 100,
--     checked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     resolved_at TIMESTAMP WITH TIME ZONE,
--     resolution_notes TEXT,
--     
--     CONSTRAINT valid_status CHECK (status IN ('pass', 'fail', 'warning', 'skipped'))
-- );
-- 
-- CREATE TABLE IF NOT EXISTS compliance_reports (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
--     generated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     summary JSONB NOT NULL,
--     frameworks TEXT[],
--     period_start TIMESTAMP WITH TIME ZONE,
--     period_end TIMESTAMP WITH TIME ZONE
-- );
-- 
-- CREATE TABLE IF NOT EXISTS compliance_report_results (
--     report_id UUID NOT NULL REFERENCES compliance_reports(id) ON DELETE CASCADE,
--     result_id UUID NOT NULL REFERENCES compliance_check_results(id) ON DELETE CASCADE,
--     PRIMARY KEY (report_id, result_id)
-- );
-- 
-- -- Indexes
-- CREATE INDEX idx_compliance_rules_org ON compliance_rules(organization_id);
-- CREATE INDEX idx_compliance_rules_type ON compliance_rules(type);
-- CREATE INDEX idx_compliance_rules_severity ON compliance_rules(severity);
-- 
-- CREATE INDEX idx_compliance_results_org ON compliance_check_results(organization_id);
-- CREATE INDEX idx_compliance_results_rule ON compliance_check_results(rule_id);
-- CREATE INDEX idx_compliance_results_status ON compliance_check_results(status);
-- CREATE INDEX idx_compliance_results_checked ON compliance_check_results(checked_at DESC);
-- CREATE INDEX idx_compliance_results_resolved ON compliance_check_results(resolved_at) WHERE resolved_at IS NULL;
-- 
-- CREATE INDEX idx_compliance_reports_org ON compliance_reports(organization_id);
-- CREATE INDEX idx_compliance_reports_generated ON compliance_reports(generated_at DESC);
-- 
-- -- Trigger for updated_at on compliance_rules
-- CREATE TRIGGER update_compliance_rules_updated_at
--     BEFORE UPDATE ON compliance_rules
--     FOR EACH ROW
--     EXECUTE FUNCTION update_updated_at_column();

-- Sample data for default compliance rules (SOC 2, HIPAA, GDPR)
INSERT INTO compliance_rules (id, organization_id, name, description, type, severity, config, frameworks, created_at) VALUES
('soc2-data-retention', 'default', 'SOC 2 Data Retention Policy', 'Ensure data retention policy is configured and enforced', 'data_retention', 'warning', '{"retention_days": 365}', '["soc2"]', datetime('now')),
('soc2-access-control', 'default', 'SOC 2 Access Control', 'Verify RBAC is configured with appropriate permissions', 'access_control', 'critical', '{"rbac_enabled": true, "mfa_required": true}', '["soc2"]', datetime('now')),
('soc2-encryption', 'default', 'SOC 2 Encryption at Rest', 'Validate data encryption at rest', 'encryption', 'critical', '{"at_rest_enabled": true, "in_transit_enabled": true}', '["soc2"]', datetime('now')),
('hipaa-data-retention', 'default', 'HIPAA Data Retention', 'Healthcare data must be retained for 6 years minimum', 'data_retention', 'critical', '{"retention_days": 2190, "auto_delete": false}', '["hipaa"]', datetime('now')),
('hipaa-encryption', 'default', 'HIPAA Encryption Requirements', 'PHI must be encrypted at rest and in transit', 'encryption', 'critical', '{"at_rest_enabled": true, "in_transit_enabled": true, "algorithm": "AES-256"}', '["hipaa"]', datetime('now')),
('hipaa-access-control', 'default', 'HIPAA Access Control', 'Access to PHI must be strictly controlled and audited', 'access_control', 'critical', '{"rbac_enabled": true, "mfa_required": true, "session_timeout_min": 30}', '["hipaa"]', datetime('now')),
('hipaa-data-location', 'default', 'HIPAA Data Location', 'PHI must be stored in compliant regions', 'data_location', 'critical', '{"data_residency": true, "allowed_regions": ["us-east-1", "us-west-2"]}', '["hipaa"]', datetime('now')),
('gdpr-data-retention', 'default', 'GDPR Data Retention', 'Personal data must not be kept longer than necessary', 'data_retention', 'warning', '{"retention_days": 90, "auto_delete": true}', '["gdpr"]', datetime('now')),
('gdpr-data-location', 'default', 'GDPR Data Location', 'Personal data must remain in EU or approved regions', 'data_location', 'critical', '{"data_residency": true, "allowed_regions": ["eu-west-1", "eu-central-1"]}', '["gdpr"]', datetime('now')),
('gdpr-encryption', 'default', 'GDPR Encryption', 'Personal data must be protected with appropriate encryption', 'encryption', 'critical', '{"at_rest_enabled": true, "in_transit_enabled": true}', '["gdpr"]', datetime('now'));
