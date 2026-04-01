package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/compliance"
	"github.com/raynaythegreat/octai-app/pkg/tenant"
)

type ComplianceHandler struct {
	registry    *compliance.CheckerRegistry
	reportGen   *compliance.ReportGenerator
	remediation *compliance.RemediationService
	store       *MemoryComplianceStore
	tenantStore tenant.TenantStore
}

type MemoryComplianceStore struct {
	orgConfigs   map[string]map[string]json.RawMessage
	checkResults map[string]*compliance.ComplianceCheckResult
	rules        map[string]*compliance.ComplianceRule
	reports      map[string]*compliance.ComplianceReport
	memberships  map[string][]map[string]interface{}
	dataRegions  map[string]string
	orgNames     map[string]string
}

func NewMemoryComplianceStore() *MemoryComplianceStore {
	return &MemoryComplianceStore{
		orgConfigs:   make(map[string]map[string]json.RawMessage),
		checkResults: make(map[string]*compliance.ComplianceCheckResult),
		rules:        make(map[string]*compliance.ComplianceRule),
		reports:      make(map[string]*compliance.ComplianceReport),
		memberships:  make(map[string][]map[string]interface{}),
		dataRegions:  make(map[string]string),
		orgNames:     make(map[string]string),
	}
}

func (s *MemoryComplianceStore) GetOrgConfig(ctx context.Context, orgID string, configType string) (json.RawMessage, error) {
	orgConfigs, ok := s.orgConfigs[orgID]
	if !ok {
		return nil, fmt.Errorf("org config not found")
	}
	config, ok := orgConfigs[configType]
	if !ok {
		return nil, fmt.Errorf("config type not found")
	}
	return config, nil
}

func (s *MemoryComplianceStore) GetMemberships(ctx context.Context, orgID string) ([]map[string]interface{}, error) {
	return s.memberships[orgID], nil
}

func (s *MemoryComplianceStore) GetDataRegion(ctx context.Context, orgID string) (string, error) {
	return s.dataRegions[orgID], nil
}

func (s *MemoryComplianceStore) GetCheckResult(ctx context.Context, resultID string) (*compliance.ComplianceCheckResult, error) {
	result, ok := s.checkResults[resultID]
	if !ok {
		return nil, fmt.Errorf("result not found")
	}
	return result, nil
}

func (s *MemoryComplianceStore) UpdateCheckResult(ctx context.Context, result *compliance.ComplianceCheckResult) error {
	s.checkResults[result.ID] = result
	return nil
}

func (s *MemoryComplianceStore) GetOpenIssues(ctx context.Context, orgID string) ([]*compliance.ComplianceCheckResult, error) {
	var issues []*compliance.ComplianceCheckResult
	for _, result := range s.checkResults {
		if result.OrganizationID == orgID && !result.IsResolved() && result.Status != compliance.CheckStatusPass {
			issues = append(issues, result)
		}
	}
	return issues, nil
}

func (s *MemoryComplianceStore) GetRule(ctx context.Context, ruleID string) (*compliance.ComplianceRule, error) {
	rule, ok := s.rules[ruleID]
	if !ok {
		return nil, fmt.Errorf("rule not found")
	}
	return rule, nil
}

func (s *MemoryComplianceStore) GetOrganization(ctx context.Context, orgID string) (string, error) {
	name, ok := s.orgNames[orgID]
	if !ok {
		return orgID, nil
	}
	return name, nil
}

func (s *MemoryComplianceStore) GetRules(ctx context.Context, orgID string, frameworks []compliance.ComplianceFramework) ([]compliance.ComplianceRule, error) {
	rules := []compliance.ComplianceRule{}
	rules = append(rules, compliance.DefaultSOC2Rules()...)
	rules = append(rules, compliance.DefaultHIPAARules()...)
	rules = append(rules, compliance.DefaultGDPRRules()...)

	if len(frameworks) == 0 {
		return rules, nil
	}

	frameworkSet := make(map[compliance.ComplianceFramework]bool)
	for _, f := range frameworks {
		frameworkSet[f] = true
	}

	filtered := []compliance.ComplianceRule{}
	for _, rule := range rules {
		for _, f := range rule.Frameworks {
			if frameworkSet[f] {
				filtered = append(filtered, rule)
				break
			}
		}
	}
	return filtered, nil
}

func (s *MemoryComplianceStore) SaveReport(ctx context.Context, report *compliance.ComplianceReport) error {
	s.reports[report.ID] = report
	return nil
}

func (s *MemoryComplianceStore) GetHistoricalReports(ctx context.Context, orgID string, limit int) ([]*compliance.ComplianceReport, error) {
	var reports []*compliance.ComplianceReport
	for _, report := range s.reports {
		if report.OrganizationID == orgID {
			reports = append(reports, report)
		}
	}
	return reports, nil
}

func (s *MemoryComplianceStore) SetOrgConfig(orgID string, configType string, config interface{}) error {
	if s.orgConfigs[orgID] == nil {
		s.orgConfigs[orgID] = make(map[string]json.RawMessage)
	}
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	s.orgConfigs[orgID][configType] = data
	return nil
}

func NewComplianceHandler(tenantStore tenant.TenantStore) *ComplianceHandler {
	store := NewMemoryComplianceStore()
	registry := compliance.NewCheckerRegistry(store)
	reportGen := compliance.NewReportGenerator(registry, store)
	remediation := compliance.NewRemediationService(store)

	return &ComplianceHandler{
		registry:    registry,
		reportGen:   reportGen,
		remediation: remediation,
		store:       store,
		tenantStore: tenantStore,
	}
}

func (h *Handler) getComplianceHandler() *ComplianceHandler {
	if h.complianceHandler == nil {
		h.complianceHandler = NewComplianceHandler(h.tenantStore)
	}
	return h.complianceHandler
}

type ComplianceCheckResponse struct {
	OrganizationID string                              `json:"organization_id"`
	CheckedAt      string                              `json:"checked_at"`
	Results        []*compliance.ComplianceCheckResult `json:"results"`
	Summary        ComplianceCheckSummary              `json:"summary"`
}

type ComplianceCheckSummary struct {
	Total    int `json:"total"`
	Passed   int `json:"passed"`
	Failed   int `json:"failed"`
	Warnings int `json:"warnings"`
}

type ComplianceReportResponse struct {
	ID               string                           `json:"id"`
	OrganizationID   string                           `json:"organization_id"`
	OrganizationName string                           `json:"organization_name"`
	GeneratedAt      string                           `json:"generated_at"`
	Summary          compliance.ReportSummary         `json:"summary"`
	Frameworks       []compliance.ComplianceFramework `json:"frameworks"`
	DownloadURL      string                           `json:"download_url,omitempty"`
}

type ResolveIssueRequest struct {
	Notes string `json:"notes"`
}

type RemediationStepsResponse struct {
	ResultID string                       `json:"result_id"`
	Steps    []compliance.RemediationStep `json:"steps"`
}

func (h *Handler) registerComplianceRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v2/organizations/{id}/compliance/check", h.handleComplianceCheck)
	mux.HandleFunc("GET /api/v2/organizations/{id}/compliance/report", h.handleGetComplianceReport)
	mux.HandleFunc("GET /api/v2/organizations/{id}/compliance/issues", h.handleGetOpenIssues)
	mux.HandleFunc("POST /api/v2/organizations/{id}/compliance/resolve", h.handleResolveIssue)
	mux.HandleFunc("GET /api/v2/organizations/{id}/compliance/remediation/{resultId}", h.handleGetRemediationSteps)
}

func (h *Handler) handleComplianceCheck(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		writeJSONError(w, "organization id is required", http.StatusBadRequest)
		return
	}

	tc := h.requireOrgAccess(w, r, orgID, tenant.PermOrgRead)
	if tc == nil {
		return
	}

	frameworks := parseFrameworks(r.URL.Query().Get("frameworks"))
	ruleTypes := parseRuleTypes(r.URL.Query().Get("types"))

	ch := h.getComplianceHandler()

	ctx := r.Context()
	rules, err := ch.store.GetRules(ctx, orgID, frameworks)
	if err != nil {
		writeJSONError(w, "failed to get compliance rules", http.StatusInternalServerError)
		return
	}

	if len(ruleTypes) > 0 {
		rules = filterRulesByType(rules, ruleTypes)
	}

	results, err := ch.registry.CheckAll(ctx, orgID, rules)
	if err != nil {
		writeJSONError(w, fmt.Sprintf("compliance check failed: %v", err), http.StatusInternalServerError)
		return
	}

	for _, result := range results {
		ch.store.checkResults[result.ID] = result
	}

	summary := ComplianceCheckSummary{
		Total: len(results),
	}
	for _, result := range results {
		switch result.Status {
		case compliance.CheckStatusPass:
			summary.Passed++
		case compliance.CheckStatusFail:
			summary.Failed++
		case compliance.CheckStatusWarning:
			summary.Warnings++
		}
	}

	response := ComplianceCheckResponse{
		OrganizationID: orgID,
		CheckedAt:      time.Now().Format(time.RFC3339),
		Results:        results,
		Summary:        summary,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) handleGetComplianceReport(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		writeJSONError(w, "organization id is required", http.StatusBadRequest)
		return
	}

	tc := h.requireOrgAccess(w, r, orgID, tenant.PermOrgRead)
	if tc == nil {
		return
	}

	frameworks := parseFrameworks(r.URL.Query().Get("frameworks"))
	ruleTypes := parseRuleTypes(r.URL.Query().Get("types"))
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	ch := h.getComplianceHandler()
	ctx := r.Context()

	report, err := ch.reportGen.GenerateReport(ctx, orgID, frameworks, ruleTypes)
	if err != nil {
		writeJSONError(w, fmt.Sprintf("failed to generate report: %v", err), http.StatusInternalServerError)
		return
	}

	if format == "pdf" {
		pdfData, err := ch.reportGen.ExportToPDF(ctx, report)
		if err != nil {
			writeJSONError(w, "failed to export PDF", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=compliance-report-%s.pdf", report.ID))
		w.Write(pdfData)
		return
	}

	if format == "json" {
		jsonData, err := ch.reportGen.ExportToJSON(ctx, report)
		if err != nil {
			writeJSONError(w, "failed to export JSON", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=compliance-report-%s.json", report.ID))
		w.Write(jsonData)
		return
	}

	response := ComplianceReportResponse{
		ID:               report.ID,
		OrganizationID:   report.OrganizationID,
		OrganizationName: report.OrganizationName,
		GeneratedAt:      report.GeneratedAt.Format(time.RFC3339),
		Summary:          report.Summary,
		Frameworks:       report.Frameworks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) handleGetOpenIssues(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		writeJSONError(w, "organization id is required", http.StatusBadRequest)
		return
	}

	tc := h.requireOrgAccess(w, r, orgID, tenant.PermOrgRead)
	if tc == nil {
		return
	}

	ch := h.getComplianceHandler()
	ctx := r.Context()

	issues, err := ch.remediation.GetOpenIssues(ctx, orgID)
	if err != nil {
		writeJSONError(w, "failed to get open issues", http.StatusInternalServerError)
		return
	}

	severity := r.URL.Query().Get("severity")
	if severity != "" {
		issues = filterIssuesBySeverity(issues, compliance.Severity(severity))
	}

	response := map[string]interface{}{
		"organization_id": orgID,
		"issues":          issues,
		"total":           len(issues),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) handleResolveIssue(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		writeJSONError(w, "organization id is required", http.StatusBadRequest)
		return
	}

	tc := h.requireOrgAccess(w, r, orgID, tenant.PermOrgWrite)
	if tc == nil {
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSONError(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req ResolveIssueRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	resultID := r.URL.Query().Get("result_id")
	if resultID == "" {
		writeJSONError(w, "result_id is required", http.StatusBadRequest)
		return
	}

	ch := h.getComplianceHandler()
	ctx := r.Context()

	if err := ch.remediation.MarkAsResolved(ctx, resultID, req.Notes); err != nil {
		writeJSONError(w, fmt.Sprintf("failed to resolve issue: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"result_id":   resultID,
		"resolved":    true,
		"resolved_at": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) handleGetRemediationSteps(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	resultID := r.PathValue("resultId")
	if orgID == "" || resultID == "" {
		writeJSONError(w, "organization id and result id are required", http.StatusBadRequest)
		return
	}

	tc := h.requireOrgAccess(w, r, orgID, tenant.PermOrgRead)
	if tc == nil {
		return
	}

	ch := h.getComplianceHandler()
	ctx := r.Context()

	result, err := ch.store.GetCheckResult(ctx, resultID)
	if err != nil {
		writeJSONError(w, "result not found", http.StatusNotFound)
		return
	}

	steps, err := ch.remediation.GetRemediationSteps(ctx, result)
	if err != nil {
		writeJSONError(w, fmt.Sprintf("failed to get remediation steps: %v", err), http.StatusInternalServerError)
		return
	}

	response := RemediationStepsResponse{
		ResultID: resultID,
		Steps:    steps,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func parseFrameworks(s string) []compliance.ComplianceFramework {
	if s == "" {
		return nil
	}
	frameworks := []compliance.ComplianceFramework{}
	for _, f := range splitList(s) {
		frameworks = append(frameworks, compliance.ComplianceFramework(f))
	}
	return frameworks
}

func parseRuleTypes(s string) []compliance.RuleType {
	if s == "" {
		return nil
	}
	types := []compliance.RuleType{}
	for _, t := range splitList(s) {
		types = append(types, compliance.RuleType(t))
	}
	return types
}

func splitList(s string) []string {
	var result []string
	for _, item := range []byte(s) {
		if item == ',' {
			continue
		}
		result = append(result, string(item))
	}
	if len(result) == 0 && s != "" {
		return []string{s}
	}
	return result
}

func filterRulesByType(rules []compliance.ComplianceRule, types []compliance.RuleType) []compliance.ComplianceRule {
	typeSet := make(map[compliance.RuleType]bool)
	for _, t := range types {
		typeSet[t] = true
	}
	var filtered []compliance.ComplianceRule
	for _, rule := range rules {
		if typeSet[rule.Type] {
			filtered = append(filtered, rule)
		}
	}
	return filtered
}

func filterIssuesBySeverity(issues []*compliance.ComplianceCheckResult, severity compliance.Severity) []*compliance.ComplianceCheckResult {
	var filtered []*compliance.ComplianceCheckResult
	for _, issue := range issues {
		for _, finding := range issue.Findings {
			if finding.Severity == severity {
				filtered = append(filtered, issue)
				break
			}
		}
	}
	return filtered
}

func init() {
	postInitFuncs = append(postInitFuncs, func(h *Handler) {
		if h.complianceHandler == nil {
			h.complianceHandler = NewComplianceHandler(h.tenantStore)
		}
	})
}

type MockComplianceStore struct {
	configs     map[string]map[string]json.RawMessage
	results     map[string]*compliance.ComplianceCheckResult
	rules       map[string]*compliance.ComplianceRule
	orgNames    map[string]string
	reports     []*compliance.ComplianceReport
	memberships map[string][]map[string]interface{}
	regions     map[string]string
}

func NewMockComplianceStore() *MockComplianceStore {
	return &MockComplianceStore{
		configs:     make(map[string]map[string]json.RawMessage),
		results:     make(map[string]*compliance.ComplianceCheckResult),
		rules:       make(map[string]*compliance.ComplianceRule),
		orgNames:    make(map[string]string),
		memberships: make(map[string][]map[string]interface{}),
		regions:     make(map[string]string),
	}
}

func (m *MockComplianceStore) SetConfig(orgID, configType string, config interface{}) error {
	if m.configs[orgID] == nil {
		m.configs[orgID] = make(map[string]json.RawMessage)
	}
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	m.configs[orgID][configType] = data
	return nil
}

func (m *MockComplianceStore) GetOrgConfig(ctx context.Context, orgID string, configType string) (json.RawMessage, error) {
	if m.configs[orgID] == nil {
		return nil, fmt.Errorf("not found")
	}
	return m.configs[orgID][configType], nil
}

func (m *MockComplianceStore) GetMemberships(ctx context.Context, orgID string) ([]map[string]interface{}, error) {
	return m.memberships[orgID], nil
}

func (m *MockComplianceStore) GetDataRegion(ctx context.Context, orgID string) (string, error) {
	return m.regions[orgID], nil
}

func (m *MockComplianceStore) GetCheckResult(ctx context.Context, resultID string) (*compliance.ComplianceCheckResult, error) {
	return m.results[resultID], nil
}

func (m *MockComplianceStore) UpdateCheckResult(ctx context.Context, result *compliance.ComplianceCheckResult) error {
	m.results[result.ID] = result
	return nil
}

func (m *MockComplianceStore) GetOpenIssues(ctx context.Context, orgID string) ([]*compliance.ComplianceCheckResult, error) {
	var issues []*compliance.ComplianceCheckResult
	for _, r := range m.results {
		if r.OrganizationID == orgID && r.Status != compliance.CheckStatusPass && r.ResolvedAt == nil {
			issues = append(issues, r)
		}
	}
	return issues, nil
}

func (m *MockComplianceStore) GetRule(ctx context.Context, ruleID string) (*compliance.ComplianceRule, error) {
	return m.rules[ruleID], nil
}

func (m *MockComplianceStore) GetOrganization(ctx context.Context, orgID string) (string, error) {
	return m.orgNames[orgID], nil
}

func (m *MockComplianceStore) GetRules(ctx context.Context, orgID string, frameworks []compliance.ComplianceFramework) ([]compliance.ComplianceRule, error) {
	rules := []compliance.ComplianceRule{}
	rules = append(rules, compliance.DefaultSOC2Rules()...)
	rules = append(rules, compliance.DefaultHIPAARules()...)
	rules = append(rules, compliance.DefaultGDPRRules()...)

	if len(frameworks) == 0 {
		return rules, nil
	}

	fwSet := make(map[compliance.ComplianceFramework]bool)
	for _, f := range frameworks {
		fwSet[f] = true
	}

	var filtered []compliance.ComplianceRule
	for _, rule := range rules {
		for _, f := range rule.Frameworks {
			if fwSet[f] {
				filtered = append(filtered, rule)
				break
			}
		}
	}
	return filtered, nil
}

func (m *MockComplianceStore) SaveReport(ctx context.Context, report *compliance.ComplianceReport) error {
	m.reports = append(m.reports, report)
	return nil
}

func (m *MockComplianceStore) GetHistoricalReports(ctx context.Context, orgID string, limit int) ([]*compliance.ComplianceReport, error) {
	return m.reports, nil
}

func (m *MockComplianceStore) AddMembership(orgID, userID, role string) {
	if m.memberships[orgID] == nil {
		m.memberships[orgID] = []map[string]interface{}{}
	}
	m.memberships[orgID] = append(m.memberships[orgID], map[string]interface{}{
		"user_id": userID,
		"role":    role,
	})
}
