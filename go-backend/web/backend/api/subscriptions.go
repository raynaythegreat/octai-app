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

type SubscriptionResponse struct {
	ID                 string  `json:"id"`
	OrganizationID     string  `json:"organization_id"`
	Tier               string  `json:"tier"`
	Status             string  `json:"status"`
	StripeCustomerID   *string `json:"stripe_customer_id,omitempty"`
	StripePriceID      *string `json:"stripe_price_id,omitempty"`
	CurrentPeriodStart *string `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   *string `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd  bool    `json:"cancel_at_period_end"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
	CanceledAt         *string `json:"canceled_at,omitempty"`
}

type CreateSubscriptionRequest struct {
	Tier        string  `json:"tier"`
	PriceID     *string `json:"price_id,omitempty"`
	StripeToken *string `json:"stripe_token,omitempty"`
}

type PlanResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Tier        string     `json:"tier"`
	Price       int        `json:"price"`
	PriceYearly int        `json:"price_yearly"`
	Features    []string   `json:"features"`
	Limits      PlanLimits `json:"limits"`
}

type PlanLimits struct {
	MaxUsers    int `json:"max_users"`
	MaxAgents   int `json:"max_agents"`
	MaxMessages int `json:"max_messages"`
	MaxChannels int `json:"max_channels"`
}

type ListPlansResponse struct {
	Plans []PlanResponse `json:"plans"`
}

var availablePlans = []PlanResponse{
	{
		ID:          "free",
		Name:        "Free",
		Tier:        "free",
		Price:       0,
		PriceYearly: 0,
		Features:    tenant.TierConfig[tenant.TierFree].Features,
		Limits: PlanLimits{
			MaxUsers:    tenant.TierConfig[tenant.TierFree].MaxUsers,
			MaxAgents:   tenant.TierConfig[tenant.TierFree].MaxAgents,
			MaxMessages: tenant.TierConfig[tenant.TierFree].MaxMessages,
			MaxChannels: tenant.TierConfig[tenant.TierFree].MaxChannels,
		},
	},
	{
		ID:          "pro",
		Name:        "Pro",
		Tier:        "pro",
		Price:       29,
		PriceYearly: 290,
		Features:    tenant.TierConfig[tenant.TierPro].Features,
		Limits: PlanLimits{
			MaxUsers:    tenant.TierConfig[tenant.TierPro].MaxUsers,
			MaxAgents:   tenant.TierConfig[tenant.TierPro].MaxAgents,
			MaxMessages: tenant.TierConfig[tenant.TierPro].MaxMessages,
			MaxChannels: tenant.TierConfig[tenant.TierPro].MaxChannels,
		},
	},
	{
		ID:          "business",
		Name:        "Business",
		Tier:        "business",
		Price:       99,
		PriceYearly: 990,
		Features:    tenant.TierConfig[tenant.TierBusiness].Features,
		Limits: PlanLimits{
			MaxUsers:    tenant.TierConfig[tenant.TierBusiness].MaxUsers,
			MaxAgents:   tenant.TierConfig[tenant.TierBusiness].MaxAgents,
			MaxMessages: tenant.TierConfig[tenant.TierBusiness].MaxMessages,
			MaxChannels: tenant.TierConfig[tenant.TierBusiness].MaxChannels,
		},
	},
	{
		ID:          "enterprise",
		Name:        "Enterprise",
		Tier:        "enterprise",
		Price:       0,
		PriceYearly: 0,
		Features:    tenant.TierConfig[tenant.TierEnterprise].Features,
		Limits: PlanLimits{
			MaxUsers:    tenant.TierConfig[tenant.TierEnterprise].MaxUsers,
			MaxAgents:   tenant.TierConfig[tenant.TierEnterprise].MaxAgents,
			MaxMessages: tenant.TierConfig[tenant.TierEnterprise].MaxMessages,
			MaxChannels: tenant.TierConfig[tenant.TierEnterprise].MaxChannels,
		},
	},
}

func (h *Handler) registerSubscriptionRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v2/organizations/{id}/subscription", h.handleGetSubscription)
	mux.HandleFunc("POST /api/v2/organizations/{id}/subscription", h.handleCreateSubscription)
	mux.HandleFunc("DELETE /api/v2/organizations/{id}/subscription", h.handleCancelSubscription)
	mux.HandleFunc("GET /api/v2/plans", h.handleListPlans)
}

func subscriptionToResponse(sub *tenant.Subscription, org *tenant.Organization) SubscriptionResponse {
	return SubscriptionResponse{
		ID:                 sub.ID,
		OrganizationID:     sub.OrganizationID,
		Tier:               string(sub.Tier),
		Status:             string(sub.Status),
		StripeCustomerID:   org.StripeCustomerID,
		StripePriceID:      sub.StripePriceID,
		CurrentPeriodStart: formatTimePtr(sub.CurrentPeriodStart),
		CurrentPeriodEnd:   formatTimePtr(sub.CurrentPeriodEnd),
		CancelAtPeriodEnd:  sub.CancelAtPeriodEnd,
		CreatedAt:          sub.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          sub.UpdatedAt.Format(time.RFC3339),
		CanceledAt:         formatTimePtr(sub.CanceledAt),
	}
}

func (h *Handler) handleGetSubscription(w http.ResponseWriter, r *http.Request) {
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
	sub, err := svc.GetSubscription(r.Context(), orgID)
	if err != nil {
		if errors.Is(err, tenant.ErrSubscriptionNotFound) {
			writeJSONError(w, "subscription not found", http.StatusNotFound)
			return
		}
		logger.ErrorC("api", fmt.Sprintf("failed to get subscription: %v", err))
		writeJSONError(w, "failed to get subscription", http.StatusInternalServerError)
		return
	}

	org, err := svc.GetOrganization(r.Context(), orgID)
	if err != nil {
		writeJSONError(w, "failed to get organization", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscriptionToResponse(sub, org))
}

func (h *Handler) handleCreateSubscription(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		writeJSONError(w, "organization id is required", http.StatusBadRequest)
		return
	}

	tc := h.requireOrgAccess(w, r, orgID, tenant.PermBillingManage)
	if tc == nil {
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSONError(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req CreateSubscriptionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	tier := tenant.Tier(req.Tier)
	if !tier.Valid() {
		writeJSONError(w, "invalid tier", http.StatusBadRequest)
		return
	}

	svc := h.getTenantService()
	existingSub, err := svc.GetSubscription(r.Context(), orgID)
	if err != nil && !errors.Is(err, tenant.ErrSubscriptionNotFound) {
		logger.ErrorC("api", fmt.Sprintf("failed to check existing subscription: %v", err))
		writeJSONError(w, "failed to check subscription", http.StatusInternalServerError)
		return
	}

	if existingSub != nil {
		existingSub.Tier = tier
		existingSub.Status = tenant.SubStatusActive
		existingSub.CancelAtPeriodEnd = false
		existingSub.CanceledAt = nil
		if req.PriceID != nil {
			existingSub.StripePriceID = req.PriceID
		}

		now := time.Now()
		periodEnd := now.AddDate(0, 1, 0)
		existingSub.CurrentPeriodStart = &now
		existingSub.CurrentPeriodEnd = &periodEnd

		if err := h.tenantStore.UpdateSubscription(r.Context(), existingSub); err != nil {
			logger.ErrorC("api", fmt.Sprintf("failed to upgrade subscription: %v", err))
			writeJSONError(w, "failed to upgrade subscription", http.StatusInternalServerError)
			return
		}

		org, _ := svc.GetOrganization(r.Context(), orgID)
		logger.Infof("subscription upgraded: %s -> %s for organization %s", existingSub.Tier, tier, orgID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(subscriptionToResponse(existingSub, org))
		return
	}

	now := time.Now()
	periodEnd := now.AddDate(0, 1, 0)
	newSub := &tenant.Subscription{
		OrganizationID:     orgID,
		Status:             tenant.SubStatusActive,
		Tier:               tier,
		StripePriceID:      req.PriceID,
		CurrentPeriodStart: &now,
		CurrentPeriodEnd:   &periodEnd,
		CancelAtPeriodEnd:  false,
	}

	if err := h.tenantStore.CreateSubscription(r.Context(), newSub); err != nil {
		logger.ErrorC("api", fmt.Sprintf("failed to create subscription: %v", err))
		writeJSONError(w, "failed to create subscription", http.StatusInternalServerError)
		return
	}

	org, _ := svc.GetOrganization(r.Context(), orgID)
	logger.Infof("subscription created: %s for organization %s", tier, orgID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(subscriptionToResponse(newSub, org))
}

func (h *Handler) handleCancelSubscription(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		writeJSONError(w, "organization id is required", http.StatusBadRequest)
		return
	}

	tc := h.requireOrgAccess(w, r, orgID, tenant.PermBillingManage)
	if tc == nil {
		return
	}

	svc := h.getTenantService()
	sub, err := svc.GetSubscription(r.Context(), orgID)
	if err != nil {
		if errors.Is(err, tenant.ErrSubscriptionNotFound) {
			writeJSONError(w, "subscription not found", http.StatusNotFound)
			return
		}
		logger.ErrorC("api", fmt.Sprintf("failed to get subscription: %v", err))
		writeJSONError(w, "failed to get subscription", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	sub.CancelAtPeriodEnd = true
	sub.CanceledAt = &now
	sub.Status = tenant.SubStatusCanceled

	if err := h.tenantStore.UpdateSubscription(r.Context(), sub); err != nil {
		logger.ErrorC("api", fmt.Sprintf("failed to cancel subscription: %v", err))
		writeJSONError(w, "failed to cancel subscription", http.StatusInternalServerError)
		return
	}

	org, _ := svc.GetOrganization(r.Context(), orgID)
	logger.Infof("subscription canceled for organization %s", orgID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscriptionToResponse(sub, org))
}

func (h *Handler) handleListPlans(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ListPlansResponse{
		Plans: availablePlans,
	})
}
