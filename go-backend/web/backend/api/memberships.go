package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/logger"
	"github.com/raynaythegreat/octai-app/pkg/tenant"
)

type MemberResponse struct {
	ID         string  `json:"id"`
	UserID     string  `json:"user_id"`
	Role       string  `json:"role"`
	InvitedBy  *string `json:"invited_by,omitempty"`
	InvitedAt  *string `json:"invited_at,omitempty"`
	AcceptedAt *string `json:"accepted_at,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

type InviteMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role"`
}

type ListMembersResponse struct {
	Members []MemberResponse `json:"members"`
	Total   int              `json:"total"`
}

func (h *Handler) registerMembershipRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v2/organizations/{id}/members", h.handleListMembers)
	mux.HandleFunc("POST /api/v2/organizations/{id}/members", h.handleInviteMember)
	mux.HandleFunc("PUT /api/v2/organizations/{id}/members/{userId}", h.handleUpdateMemberRole)
	mux.HandleFunc("DELETE /api/v2/organizations/{id}/members/{userId}", h.handleRemoveMember)
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.Format(time.RFC3339)
	return &formatted
}

func (h *Handler) handleListMembers(w http.ResponseWriter, r *http.Request) {
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
	members, err := svc.ListMemberships(r.Context(), orgID)
	if err != nil {
		logger.ErrorC("api", fmt.Sprintf("failed to list members: %v", err))
		writeJSONError(w, "failed to list members", http.StatusInternalServerError)
		return
	}

	response := ListMembersResponse{
		Members: make([]MemberResponse, 0, len(members)),
		Total:   len(members),
	}

	for _, m := range members {
		response.Members = append(response.Members, MemberResponse{
			ID:         m.ID,
			UserID:     m.UserID,
			Role:       string(m.Role),
			InvitedBy:  m.InvitedBy,
			InvitedAt:  formatTimePtr(m.InvitedAt),
			AcceptedAt: formatTimePtr(m.AcceptedAt),
			CreatedAt:  m.CreatedAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) handleInviteMember(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		writeJSONError(w, "organization id is required", http.StatusBadRequest)
		return
	}

	tc := h.requireOrgAccess(w, r, orgID, tenant.PermMembersManage)
	if tc == nil {
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSONError(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req InviteMemberRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		writeJSONError(w, "user_id is required", http.StatusBadRequest)
		return
	}

	role := tenant.Role(req.Role)
	if role == "" {
		role = tenant.RoleMember
	}
	if !role.Valid() {
		writeJSONError(w, "invalid role", http.StatusBadRequest)
		return
	}

	svc := h.getTenantService()
	membership, err := svc.AddMember(r.Context(), tenant.AddMemberInput{
		OrganizationID: orgID,
		UserID:         req.UserID,
		Role:           role,
		InvitedBy:      tc.UserID,
	})
	if err != nil {
		if errors.Is(err, tenant.ErrAlreadyMember) {
			writeJSONError(w, "user is already a member", http.StatusConflict)
			return
		}
		if errors.Is(err, tenant.ErrLimitExceeded) {
			writeJSONError(w, "member limit exceeded for current plan", http.StatusPaymentRequired)
			return
		}
		logger.ErrorC("api", fmt.Sprintf("failed to invite member: %v", err))
		writeJSONError(w, "failed to invite member", http.StatusInternalServerError)
		return
	}

	logger.Infof("member invited: %s to organization %s", req.UserID, orgID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(MemberResponse{
		ID:         membership.ID,
		UserID:     membership.UserID,
		Role:       string(membership.Role),
		InvitedBy:  membership.InvitedBy,
		InvitedAt:  formatTimePtr(membership.InvitedAt),
		AcceptedAt: formatTimePtr(membership.AcceptedAt),
		CreatedAt:  membership.CreatedAt.Format(time.RFC3339),
	})
}

func (h *Handler) handleUpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	targetUserID := r.PathValue("userId")
	if orgID == "" || targetUserID == "" {
		writeJSONError(w, "organization id and user id are required", http.StatusBadRequest)
		return
	}

	tc := h.requireOrgAccess(w, r, orgID, tenant.PermMembersManage)
	if tc == nil {
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSONError(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req UpdateMemberRoleRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	newRole := tenant.Role(req.Role)
	if !newRole.Valid() {
		writeJSONError(w, "invalid role", http.StatusBadRequest)
		return
	}

	svc := h.getTenantService()
	members, err := svc.ListMemberships(r.Context(), orgID)
	if err != nil {
		writeJSONError(w, "failed to list members", http.StatusInternalServerError)
		return
	}

	var membershipID string
	for _, m := range members {
		if m.UserID == targetUserID {
			membershipID = m.ID
			break
		}
	}

	if membershipID == "" {
		writeJSONError(w, "member not found", http.StatusNotFound)
		return
	}

	if err := svc.UpdateRole(r.Context(), orgID, membershipID, newRole); err != nil {
		if errors.Is(err, tenant.ErrMembershipNotFound) {
			writeJSONError(w, "member not found", http.StatusNotFound)
			return
		}
		if err.Error() == "cannot demote the last owner" {
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		logger.ErrorC("api", fmt.Sprintf("failed to update member role: %v", err))
		writeJSONError(w, "failed to update member role", http.StatusInternalServerError)
		return
	}

	logger.Infof("member role updated: %s -> %s in organization %s", targetUserID, newRole, orgID)

	updatedMembers, _ := svc.ListMemberships(r.Context(), orgID)
	var updatedMember *tenant.Membership
	for _, m := range updatedMembers {
		if m.UserID == targetUserID {
			updatedMember = m
			break
		}
	}

	if updatedMember == nil {
		writeJSONError(w, "failed to retrieve updated member", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MemberResponse{
		ID:        updatedMember.ID,
		UserID:    updatedMember.UserID,
		Role:      string(updatedMember.Role),
		CreatedAt: updatedMember.CreatedAt.Format(time.RFC3339),
	})
}

func (h *Handler) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	targetUserID := r.PathValue("userId")
	if orgID == "" || targetUserID == "" {
		writeJSONError(w, "organization id and user id are required", http.StatusBadRequest)
		return
	}

	tc := h.requireOrgAccess(w, r, orgID, tenant.PermMembersManage)
	if tc == nil {
		return
	}

	svc := h.getTenantService()
	members, err := svc.ListMemberships(r.Context(), orgID)
	if err != nil {
		writeJSONError(w, "failed to list members", http.StatusInternalServerError)
		return
	}

	var membershipID string
	for _, m := range members {
		if m.UserID == targetUserID {
			membershipID = m.ID
			break
		}
	}

	if membershipID == "" {
		writeJSONError(w, "member not found", http.StatusNotFound)
		return
	}

	if err := svc.RemoveMember(r.Context(), orgID, membershipID); err != nil {
		if errors.Is(err, tenant.ErrMembershipNotFound) {
			writeJSONError(w, "member not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, tenant.ErrCannotRemoveOwner) {
			writeJSONError(w, "cannot remove the organization owner", http.StatusBadRequest)
			return
		}
		logger.ErrorC("api", fmt.Sprintf("failed to remove member: %v", err))
		writeJSONError(w, "failed to remove member", http.StatusInternalServerError)
		return
	}

	logger.Infof("member removed: %s from organization %s", targetUserID, orgID)

	w.WriteHeader(http.StatusNoContent)
}

func init() {
	postInitFuncs = append(postInitFuncs, func(h *Handler) {
		h.membershipIDs = make(map[string]string)
	})
}

