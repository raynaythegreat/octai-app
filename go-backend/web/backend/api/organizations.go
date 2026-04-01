package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/raynaythegreat/octai-app/pkg/logger"
	"github.com/raynaythegreat/octai-app/pkg/tenant"
)

type OrganizationResponse struct {
	ID        string  `json:"id"`
	Slug      string  `json:"slug"`
	Name      string  `json:"name"`
	LogoURL   *string `json:"logo_url,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type CreateOrganizationRequest struct {
	Slug    string  `json:"slug"`
	Name    string  `json:"name"`
	LogoURL *string `json:"logo_url,omitempty"`
}

type UpdateOrganizationRequest struct {
	Slug    string  `json:"slug,omitempty"`
	Name    string  `json:"name,omitempty"`
	LogoURL *string `json:"logo_url,omitempty"`
}

type ListOrganizationsResponse struct {
	Organizations []OrganizationResponse `json:"organizations"`
	Total         int                    `json:"total"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func writeJSONError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

func (h *Handler) getTenantService() *tenant.TenantService {
	if h.tenantStore == nil {
		h.tenantStore = tenant.NewMemoryTenantStore()
	}
	return tenant.NewTenantService(h.tenantStore)
}

func (h *Handler) getUserIDFromContext(r *http.Request) string {
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		return userID
	}
	return "user-default"
}

func (h *Handler) requireAuth(w http.ResponseWriter, r *http.Request) string {
	userID := h.getUserIDFromContext(r)
	if userID == "" {
		writeJSONError(w, "unauthorized", http.StatusUnauthorized)
		return ""
	}
	return userID
}

func (h *Handler) requireOrgAccess(w http.ResponseWriter, r *http.Request, orgID string, perm tenant.Permission) *tenant.TenantContext {
	userID := h.getUserIDFromContext(r)
	if userID == "" {
		writeJSONError(w, "unauthorized", http.StatusUnauthorized)
		return nil
	}

	svc := h.getTenantService()
	tc, err := svc.GetTenantContext(r.Context(), orgID, userID)
	if err != nil {
		if errors.Is(err, tenant.ErrOrgNotFound) || errors.Is(err, tenant.ErrMembershipNotFound) {
			writeJSONError(w, "organization not found or access denied", http.StatusNotFound)
		} else {
			writeJSONError(w, "failed to verify access", http.StatusInternalServerError)
		}
		return nil
	}

	if !tc.HasPermission(perm) {
		writeJSONError(w, "insufficient permissions", http.StatusForbidden)
		return nil
	}

	return tc
}

func (h *Handler) registerOrganizationRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v2/organizations", h.handleListOrganizations)
	mux.HandleFunc("POST /api/v2/organizations", h.handleCreateOrganization)
	mux.HandleFunc("GET /api/v2/organizations/{id}", h.handleGetOrganization)
	mux.HandleFunc("PUT /api/v2/organizations/{id}", h.handleUpdateOrganization)
	mux.HandleFunc("DELETE /api/v2/organizations/{id}", h.handleDeleteOrganization)
}

func (h *Handler) handleListOrganizations(w http.ResponseWriter, r *http.Request) {
	userID := h.requireAuth(w, r)
	if userID == "" {
		return
	}

	store := h.tenantStore

	memStore, ok := store.(*tenant.MemoryTenantStore)
	if !ok {
		writeJSONError(w, "store not available", http.StatusInternalServerError)
		return
	}

	orgs, err := memStore.ListOrganizationsForUser(r.Context(), userID)
	if err != nil {
		logger.ErrorC("api", fmt.Sprintf("failed to list organizations: %v", err))
		writeJSONError(w, "failed to list organizations", http.StatusInternalServerError)
		return
	}

	response := ListOrganizationsResponse{
		Organizations: make([]OrganizationResponse, 0, len(orgs)),
		Total:         len(orgs),
	}

	for _, org := range orgs {
		response.Organizations = append(response.Organizations, OrganizationResponse{
			ID:        org.ID,
			Slug:      org.Slug,
			Name:      org.Name,
			LogoURL:   org.LogoURL,
			CreatedAt: org.CreatedAt.Format(time.RFC3339),
			UpdatedAt: org.UpdatedAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) handleCreateOrganization(w http.ResponseWriter, r *http.Request) {
	userID := h.requireAuth(w, r)
	if userID == "" {
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSONError(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req CreateOrganizationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		writeJSONError(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.Slug == "" {
		writeJSONError(w, "slug is required", http.StatusBadRequest)
		return
	}

	svc := h.getTenantService()
	org, err := svc.CreateOrganization(r.Context(), tenant.CreateOrganizationInput{
		ID:      uuid.New().String(),
		Slug:    req.Slug,
		Name:    req.Name,
		LogoURL: req.LogoURL,
		UserID:  userID,
	})
	if err != nil {
		if errors.Is(err, tenant.ErrInvalidSlug) {
			writeJSONError(w, "invalid slug format", http.StatusBadRequest)
			return
		}
		logger.ErrorC("api", fmt.Sprintf("failed to create organization: %v", err))
		writeJSONError(w, "failed to create organization", http.StatusInternalServerError)
		return
	}

	logger.Infof("organization created: %s", org.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(OrganizationResponse{
		ID:        org.ID,
		Slug:      org.Slug,
		Name:      org.Name,
		LogoURL:   org.LogoURL,
		CreatedAt: org.CreatedAt.Format(time.RFC3339),
		UpdatedAt: org.UpdatedAt.Format(time.RFC3339),
	})
}

func (h *Handler) handleGetOrganization(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		writeJSONError(w, "organization id is required", http.StatusBadRequest)
		return
	}

	tc := h.requireOrgAccess(w, r, orgID, tenant.PermOrgRead)
	if tc == nil {
		return
	}

	svc := h.getTenantService()
	org, err := svc.GetOrganization(r.Context(), orgID)
	if err != nil {
		if errors.Is(err, tenant.ErrOrgNotFound) {
			writeJSONError(w, "organization not found", http.StatusNotFound)
			return
		}
		writeJSONError(w, "failed to get organization", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(OrganizationResponse{
		ID:        org.ID,
		Slug:      org.Slug,
		Name:      org.Name,
		LogoURL:   org.LogoURL,
		CreatedAt: org.CreatedAt.Format(time.RFC3339),
		UpdatedAt: org.UpdatedAt.Format(time.RFC3339),
	})
}

func (h *Handler) handleUpdateOrganization(w http.ResponseWriter, r *http.Request) {
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

	var req UpdateOrganizationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	svc := h.getTenantService()
	org, err := svc.GetOrganization(r.Context(), orgID)
	if err != nil {
		writeJSONError(w, "organization not found", http.StatusNotFound)
		return
	}

	if req.Name != "" {
		org.Name = req.Name
	}
	if req.Slug != "" {
		org.Slug = req.Slug
	}
	if req.LogoURL != nil {
		org.LogoURL = req.LogoURL
	}

	if err := svc.UpdateOrganization(r.Context(), org); err != nil {
		if errors.Is(err, tenant.ErrInvalidSlug) {
			writeJSONError(w, "invalid slug format", http.StatusBadRequest)
			return
		}
		writeJSONError(w, "failed to update organization", http.StatusInternalServerError)
		return
	}

	logger.Infof("organization updated: %s", org.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(OrganizationResponse{
		ID:        org.ID,
		Slug:      org.Slug,
		Name:      org.Name,
		LogoURL:   org.LogoURL,
		CreatedAt: org.CreatedAt.Format(time.RFC3339),
		UpdatedAt: org.UpdatedAt.Format(time.RFC3339),
	})
}

func (h *Handler) handleDeleteOrganization(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		writeJSONError(w, "organization id is required", http.StatusBadRequest)
		return
	}

	tc := h.requireOrgAccess(w, r, orgID, tenant.PermOrgDelete)
	if tc == nil {
		return
	}

	if err := h.tenantStore.DeleteOrganization(r.Context(), orgID); err != nil {
		if errors.Is(err, tenant.ErrOrgNotFound) {
			writeJSONError(w, "organization not found", http.StatusNotFound)
			return
		}
		writeJSONError(w, "failed to delete organization", http.StatusInternalServerError)
		return
	}

	logger.Infof("organization deleted: %s", orgID)

	w.WriteHeader(http.StatusNoContent)
}
