package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type ComplianceReport struct {
	ID               string                   `json:"id"`
	OrganizationID   string                   `json:"organization_id"`
	OrganizationName string                   `json:"organization_name,omitempty"`
	GeneratedAt      time.Time                `json:"generated_at"`
	CheckResults     []*ComplianceCheckResult `json:"check_results"`
	Summary          ReportSummary            `json:"summary"`
	Frameworks       []ComplianceFramework    `json:"frameworks,omitempty"`
	PeriodStart      *time.Time               `json:"period_start,omitempty"`
	PeriodEnd        *time.Time               `json:"period_end,omitempty"`
}

type ReportSummary struct {
	TotalChecks    int                                      `json:"total_checks"`
	PassedChecks   int                                      `json:"passed_checks"`
	FailedChecks   int                                      `json:"failed_checks"`
	WarningChecks  int                                      `json:"warning_checks"`
	SkippedChecks  int                                      `json:"skipped_checks"`
	OverallScore   int                                      `json:"overall_score"`
	MaxScore       int                                      `json:"max_score"`
	CompliancePct  float64                                  `json:"compliance_pct"`
	CriticalIssues int                                      `json:"critical_issues"`
	WarningIssues  int                                      `json:"warning_issues"`
	ByType         map[RuleType]TypeSummary                 `json:"by_type"`
	ByFramework    map[ComplianceFramework]FrameworkSummary `json:"by_framework,omitempty"`
}

type TypeSummary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Warning int `json:"warning"`
	Score   int `json:"score"`
}

type FrameworkSummary struct {
	Framework     ComplianceFramework `json:"framework"`
	TotalChecks   int                 `json:"total_checks"`
	PassedChecks  int                 `json:"passed_checks"`
	CompliancePct float64             `json:"compliance_pct"`
	Status        CheckStatus         `json:"status"`
}

type ReportGenerator struct {
	registry *CheckerRegistry
	store    ReportStore
}

type ReportStore interface {
	GetOrganization(ctx context.Context, orgID string) (string, error)
	GetRules(ctx context.Context, orgID string, frameworks []ComplianceFramework) ([]ComplianceRule, error)
	SaveReport(ctx context.Context, report *ComplianceReport) error
	GetHistoricalReports(ctx context.Context, orgID string, limit int) ([]*ComplianceReport, error)
}

func NewReportGenerator(registry *CheckerRegistry, store ReportStore) *ReportGenerator {
	return &ReportGenerator{
		registry: registry,
		store:    store,
	}
}

func (g *ReportGenerator) GenerateReport(ctx context.Context, orgID string, frameworks []ComplianceFramework, ruleTypes []RuleType) (*ComplianceReport, error) {
	rules, err := g.store.GetRules(ctx, orgID, frameworks)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}

	filteredRules := rules
	if len(ruleTypes) > 0 {
		typeSet := make(map[RuleType]bool)
		for _, t := range ruleTypes {
			typeSet[t] = true
		}
		filteredRules = make([]ComplianceRule, 0)
		for _, rule := range rules {
			if typeSet[rule.Type] {
				filteredRules = append(filteredRules, rule)
			}
		}
	}

	results, err := g.registry.CheckAll(ctx, orgID, filteredRules)
	if err != nil {
		return nil, fmt.Errorf("failed to run checks: %w", err)
	}

	orgName, _ := g.store.GetOrganization(ctx, orgID)

	report := &ComplianceReport{
		ID:               generateReportID(),
		OrganizationID:   orgID,
		OrganizationName: orgName,
		GeneratedAt:      time.Now(),
		CheckResults:     results,
		Frameworks:       frameworks,
	}

	report.Summary = g.calculateSummary(results, filteredRules)

	if err := g.store.SaveReport(ctx, report); err != nil {
		return nil, fmt.Errorf("failed to save report: %w", err)
	}

	return report, nil
}

func (g *ReportGenerator) calculateSummary(results []*ComplianceCheckResult, rules []ComplianceRule) ReportSummary {
	summary := ReportSummary{
		TotalChecks: len(results),
		ByType:      make(map[RuleType]TypeSummary),
		ByFramework: make(map[ComplianceFramework]FrameworkSummary),
	}

	ruleMap := make(map[string]ComplianceRule)
	for _, rule := range rules {
		ruleMap[rule.ID] = rule
	}

	for _, result := range results {
		switch result.Status {
		case CheckStatusPass:
			summary.PassedChecks++
		case CheckStatusFail:
			summary.FailedChecks++
		case CheckStatusWarning:
			summary.WarningChecks++
		case CheckStatusSkipped:
			summary.SkippedChecks++
		}

		summary.OverallScore += result.Score
		summary.MaxScore += result.MaxScore

		for _, finding := range result.Findings {
			if finding.Severity == SeverityCritical {
				summary.CriticalIssues++
			} else if finding.Severity == SeverityWarning {
				summary.WarningIssues++
			}
		}

		if rule, ok := ruleMap[result.RuleID]; ok {
			typeSummary := summary.ByType[rule.Type]
			typeSummary.Total++
			typeSummary.Score += result.Score
			switch result.Status {
			case CheckStatusPass:
				typeSummary.Passed++
			case CheckStatusFail:
				typeSummary.Failed++
			case CheckStatusWarning:
				typeSummary.Warning++
			}
			summary.ByType[rule.Type] = typeSummary

			for _, framework := range rule.Frameworks {
				fwSummary := summary.ByFramework[framework]
				fwSummary.Framework = framework
				fwSummary.TotalChecks++
				if result.Status == CheckStatusPass {
					fwSummary.PassedChecks++
				}
				summary.ByFramework[framework] = fwSummary
			}
		}
	}

	if summary.MaxScore > 0 {
		summary.CompliancePct = float64(summary.OverallScore) / float64(summary.MaxScore) * 100
	}

	for framework, fwSummary := range summary.ByFramework {
		if fwSummary.TotalChecks > 0 {
			fwSummary.CompliancePct = float64(fwSummary.PassedChecks) / float64(fwSummary.TotalChecks) * 100
			if fwSummary.CompliancePct >= 90 {
				fwSummary.Status = CheckStatusPass
			} else if fwSummary.CompliancePct >= 70 {
				fwSummary.Status = CheckStatusWarning
			} else {
				fwSummary.Status = CheckStatusFail
			}
		}
		summary.ByFramework[framework] = fwSummary
	}

	return summary
}

func (g *ReportGenerator) ExportToJSON(ctx context.Context, report *ComplianceReport) ([]byte, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal report: %w", err)
	}
	return data, nil
}

func (g *ReportGenerator) ExportToPDF(ctx context.Context, report *ComplianceReport) ([]byte, error) {
	pdfContent := generatePDFContent(report)
	return []byte(pdfContent), nil
}

func generatePDFContent(report *ComplianceReport) string {
	content := fmt.Sprintf(`
COMPLIANCE REPORT
=================

Organization: %s
Report ID: %s
Generated: %s

SUMMARY
-------
Total Checks: %d
Passed: %d
Failed: %d
Warnings: %d
Compliance Score: %.1f%%
Critical Issues: %d
Warning Issues: %d

CHECK RESULTS
-------------
`, report.OrganizationName, report.ID, report.GeneratedAt.Format(time.RFC3339),
		report.Summary.TotalChecks, report.Summary.PassedChecks, report.Summary.FailedChecks,
		report.Summary.WarningChecks, report.Summary.CompliancePct,
		report.Summary.CriticalIssues, report.Summary.WarningIssues)

	for _, result := range report.CheckResults {
		status := string(result.Status)
		content += fmt.Sprintf("\n[%s] Rule: %s (Score: %d/%d)\n", status, result.RuleID, result.Score, result.MaxScore)
		for _, finding := range result.Findings {
			content += fmt.Sprintf("  - [%s] %s: %s\n", finding.Severity, finding.Title, finding.Description)
		}
	}

	if len(report.Summary.ByFramework) > 0 {
		content += "\nFRAMEWORK COMPLIANCE\n--------------------\n"
		for framework, summary := range report.Summary.ByFramework {
			content += fmt.Sprintf("%s: %.1f%% (%d/%d passed)\n", framework, summary.CompliancePct, summary.PassedChecks, summary.TotalChecks)
		}
	}

	return content
}

func (g *ReportGenerator) GetHistoricalTrend(ctx context.Context, orgID string, days int) ([]*ComplianceReport, error) {
	reports, err := g.store.GetHistoricalReports(ctx, orgID, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical reports: %w", err)
	}
	return reports, nil
}

type TrendData struct {
	Date           string  `json:"date"`
	CompliancePct  float64 `json:"compliance_pct"`
	PassedChecks   int     `json:"passed_checks"`
	FailedChecks   int     `json:"failed_checks"`
	CriticalIssues int     `json:"critical_issues"`
}

func (g *ReportGenerator) GetTrendData(ctx context.Context, orgID string, days int) ([]TrendData, error) {
	reports, err := g.GetHistoricalTrend(ctx, orgID, days)
	if err != nil {
		return nil, err
	}

	trend := make([]TrendData, 0, len(reports))
	for _, report := range reports {
		trend = append(trend, TrendData{
			Date:           report.GeneratedAt.Format("2006-01-02"),
			CompliancePct:  report.Summary.CompliancePct,
			PassedChecks:   report.Summary.PassedChecks,
			FailedChecks:   report.Summary.FailedChecks,
			CriticalIssues: report.Summary.CriticalIssues,
		})
	}
	return trend, nil
}

func generateReportID() string {
	return fmt.Sprintf("rpt-%d", time.Now().UnixNano())
}
