package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/raynaythegreat/octai-app/pkg/logger"
	"github.com/raynaythegreat/octai-app/pkg/marketplace"
)

type SkillListingResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Version     string   `json:"version"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags,omitempty"`
	Price       float64  `json:"price"`
	Rating      float64  `json:"rating"`
	Downloads   int64    `json:"downloads"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type SkillReviewResponse struct {
	ID        string `json:"id"`
	SkillID   string `json:"skill_id"`
	UserID    string `json:"user_id"`
	Rating    int    `json:"rating"`
	Comment   string `json:"comment,omitempty"`
	CreatedAt string `json:"created_at"`
}

type SkillPurchaseResponse struct {
	ID             string `json:"id"`
	SkillID        string `json:"skill_id"`
	OrganizationID string `json:"organization_id"`
	PurchasedAt    string `json:"purchased_at"`
}

type ListSkillsResponse struct {
	Skills []SkillListingResponse `json:"skills"`
	Total  int                    `json:"total"`
}

type PublishSkillRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Version     string   `json:"version,omitempty"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags,omitempty"`
	Price       float64  `json:"price"`
}

type AddReviewRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment,omitempty"`
}

type PurchaseResponse struct {
	ID          string `json:"id"`
	SkillID     string `json:"skill_id"`
	PurchasedAt string `json:"purchased_at"`
}

func (h *Handler) getMarketplaceService() *marketplace.MarketplaceService {
	if h.marketplaceStore == nil {
		h.marketplaceStore = marketplace.NewMemoryMarketplaceStore()
	}
	return marketplace.NewMarketplaceService(h.marketplaceStore)
}

func (h *Handler) registerMarketplaceRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v2/marketplace/skills", h.handleListMarketplaceSkills)
	mux.HandleFunc("GET /api/v2/marketplace/skills/{id}", h.handleGetMarketplaceSkill)
	mux.HandleFunc("POST /api/v2/marketplace/skills", h.handlePublishSkill)
	mux.HandleFunc("POST /api/v2/marketplace/skills/{id}/purchase", h.handlePurchaseSkill)
	mux.HandleFunc("GET /api/v2/marketplace/purchases", h.handleGetPurchases)
	mux.HandleFunc("POST /api/v2/marketplace/skills/{id}/reviews", h.handleAddReview)
	mux.HandleFunc("GET /api/v2/marketplace/skills/{id}/reviews", h.handleGetReviews)
	mux.HandleFunc("GET /api/v2/marketplace/recommendations", h.handleGetRecommendations)
}

func (h *Handler) handleListMarketplaceSkills(w http.ResponseWriter, r *http.Request) {
	opts := marketplace.ListSkillsOptions{}

	if category := r.URL.Query().Get("category"); category != "" {
		opts.Category = marketplace.SkillCategory(category)
	}
	if author := r.URL.Query().Get("author"); author != "" {
		opts.Author = author
	}
	if minPrice := r.URL.Query().Get("min_price"); minPrice != "" {
		if val, err := strconv.ParseFloat(minPrice, 64); err == nil {
			opts.MinPrice = &val
		}
	}
	if maxPrice := r.URL.Query().Get("max_price"); maxPrice != "" {
		if val, err := strconv.ParseFloat(maxPrice, 64); err == nil {
			opts.MaxPrice = &val
		}
	}
	if minRating := r.URL.Query().Get("min_rating"); minRating != "" {
		if val, err := strconv.ParseFloat(minRating, 64); err == nil {
			opts.MinRating = &val
		}
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil && val > 0 {
			opts.Limit = val
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if val, err := strconv.Atoi(offset); err == nil && val >= 0 {
			opts.Offset = val
		}
	}
	if sortBy := r.URL.Query().Get("sort_by"); sortBy != "" {
		opts.SortBy = sortBy
	}
	if sortDir := r.URL.Query().Get("sort_dir"); sortDir == "desc" {
		opts.SortDesc = true
	}

	svc := h.getMarketplaceService()
	skills, err := svc.ListSkills(r.Context(), opts)
	if err != nil {
		logger.ErrorC("api", fmt.Sprintf("failed to list skills: %v", err))
		writeJSONError(w, "failed to list skills", http.StatusInternalServerError)
		return
	}

	response := ListSkillsResponse{
		Skills: make([]SkillListingResponse, 0, len(skills)),
		Total:  len(skills),
	}

	for _, skill := range skills {
		response.Skills = append(response.Skills, SkillListingResponse{
			ID:          skill.ID,
			Name:        skill.Name,
			Description: skill.Description,
			Author:      skill.Author,
			Version:     skill.Version,
			Category:    string(skill.Category),
			Tags:        skill.Tags,
			Price:       skill.Price,
			Rating:      skill.Rating,
			Downloads:   skill.Downloads,
			CreatedAt:   skill.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   skill.UpdatedAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) handleGetMarketplaceSkill(w http.ResponseWriter, r *http.Request) {
	skillID := r.PathValue("id")
	if skillID == "" {
		writeJSONError(w, "skill id is required", http.StatusBadRequest)
		return
	}

	svc := h.getMarketplaceService()
	skill, err := svc.GetSkill(r.Context(), skillID)
	if err != nil {
		if errors.Is(err, marketplace.ErrSkillNotFound) {
			writeJSONError(w, "skill not found", http.StatusNotFound)
			return
		}
		writeJSONError(w, "failed to get skill", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SkillListingResponse{
		ID:          skill.ID,
		Name:        skill.Name,
		Description: skill.Description,
		Author:      skill.Author,
		Version:     skill.Version,
		Category:    string(skill.Category),
		Tags:        skill.Tags,
		Price:       skill.Price,
		Rating:      skill.Rating,
		Downloads:   skill.Downloads,
		CreatedAt:   skill.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   skill.UpdatedAt.Format(time.RFC3339),
	})
}

func (h *Handler) handlePublishSkill(w http.ResponseWriter, r *http.Request) {
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

	var req PublishSkillRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		writeJSONError(w, "name is required", http.StatusBadRequest)
		return
	}

	input := marketplace.PublishSkillInput{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Author:      req.Author,
		Version:     req.Version,
		Category:    marketplace.SkillCategory(req.Category),
		Tags:        req.Tags,
		Price:       req.Price,
	}

	if input.Author == "" {
		input.Author = userID
	}

	svc := h.getMarketplaceService()
	skill, err := svc.PublishSkill(r.Context(), input)
	if err != nil {
		if errors.Is(err, marketplace.ErrInvalidCategory) {
			writeJSONError(w, "invalid category", http.StatusBadRequest)
			return
		}
		logger.ErrorC("api", fmt.Sprintf("failed to publish skill: %v", err))
		writeJSONError(w, "failed to publish skill", http.StatusInternalServerError)
		return
	}

	logger.Infof("skill published: %s", skill.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(SkillListingResponse{
		ID:          skill.ID,
		Name:        skill.Name,
		Description: skill.Description,
		Author:      skill.Author,
		Version:     skill.Version,
		Category:    string(skill.Category),
		Tags:        skill.Tags,
		Price:       skill.Price,
		Rating:      skill.Rating,
		Downloads:   skill.Downloads,
		CreatedAt:   skill.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   skill.UpdatedAt.Format(time.RFC3339),
	})
}

func (h *Handler) handlePurchaseSkill(w http.ResponseWriter, r *http.Request) {
	userID := h.requireAuth(w, r)
	if userID == "" {
		return
	}

	skillID := r.PathValue("id")
	if skillID == "" {
		writeJSONError(w, "skill id is required", http.StatusBadRequest)
		return
	}

	orgID := r.URL.Query().Get("org_id")
	if orgID == "" {
		orgID = r.Header.Get("X-Organization-ID")
	}
	if orgID == "" {
		writeJSONError(w, "organization id is required", http.StatusBadRequest)
		return
	}

	svc := h.getMarketplaceService()
	if err := svc.InstallSkill(r.Context(), skillID, orgID); err != nil {
		if errors.Is(err, marketplace.ErrSkillNotFound) {
			writeJSONError(w, "skill not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, marketplace.ErrAlreadyPurchased) {
			writeJSONError(w, "skill already purchased", http.StatusConflict)
			return
		}
		logger.ErrorC("api", fmt.Sprintf("failed to purchase skill: %v", err))
		writeJSONError(w, "failed to purchase skill", http.StatusInternalServerError)
		return
	}

	logger.Infof("skill purchased: %s for org: %s", skillID, orgID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(PurchaseResponse{
		ID:          uuid.New().String(),
		SkillID:     skillID,
		PurchasedAt: time.Now().Format(time.RFC3339),
	})
}

func (h *Handler) handleGetPurchases(w http.ResponseWriter, r *http.Request) {
	userID := h.requireAuth(w, r)
	if userID == "" {
		return
	}

	orgID := r.URL.Query().Get("org_id")
	if orgID == "" {
		orgID = r.Header.Get("X-Organization-ID")
	}
	if orgID == "" {
		writeJSONError(w, "organization id is required", http.StatusBadRequest)
		return
	}

	svc := h.getMarketplaceService()
	purchases, err := svc.GetPurchases(r.Context(), orgID)
	if err != nil {
		logger.ErrorC("api", fmt.Sprintf("failed to get purchases: %v", err))
		writeJSONError(w, "failed to get purchases", http.StatusInternalServerError)
		return
	}

	response := make([]SkillPurchaseResponse, 0, len(purchases))
	for _, p := range purchases {
		response = append(response, SkillPurchaseResponse{
			ID:             p.ID,
			SkillID:        p.SkillID,
			OrganizationID: p.OrganizationID,
			PurchasedAt:    p.PurchasedAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"purchases": response,
		"total":     len(response),
	})
}

func (h *Handler) handleAddReview(w http.ResponseWriter, r *http.Request) {
	userID := h.requireAuth(w, r)
	if userID == "" {
		return
	}

	skillID := r.PathValue("id")
	if skillID == "" {
		writeJSONError(w, "skill id is required", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSONError(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req AddReviewRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if req.Rating < 1 || req.Rating > 5 {
		writeJSONError(w, "rating must be between 1 and 5", http.StatusBadRequest)
		return
	}

	svc := h.getMarketplaceService()
	if err := svc.RateSkill(r.Context(), skillID, userID, req.Rating, req.Comment); err != nil {
		if errors.Is(err, marketplace.ErrSkillNotFound) {
			writeJSONError(w, "skill not found", http.StatusNotFound)
			return
		}
		logger.ErrorC("api", fmt.Sprintf("failed to add review: %v", err))
		writeJSONError(w, "failed to add review", http.StatusInternalServerError)
		return
	}

	logger.Infof("review added for skill: %s", skillID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) handleGetReviews(w http.ResponseWriter, r *http.Request) {
	skillID := r.PathValue("id")
	if skillID == "" {
		writeJSONError(w, "skill id is required", http.StatusBadRequest)
		return
	}

	svc := h.getMarketplaceService()
	reviews, err := svc.GetReviews(r.Context(), skillID)
	if err != nil {
		if errors.Is(err, marketplace.ErrSkillNotFound) {
			writeJSONError(w, "skill not found", http.StatusNotFound)
			return
		}
		logger.ErrorC("api", fmt.Sprintf("failed to get reviews: %v", err))
		writeJSONError(w, "failed to get reviews", http.StatusInternalServerError)
		return
	}

	response := make([]SkillReviewResponse, 0, len(reviews))
	for _, r := range reviews {
		response = append(response, SkillReviewResponse{
			ID:        r.ID,
			SkillID:   r.SkillID,
			UserID:    r.UserID,
			Rating:    r.Rating,
			Comment:   r.Comment,
			CreatedAt: r.CreatedAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"reviews": response,
		"total":   len(response),
	})
}

func (h *Handler) handleGetRecommendations(w http.ResponseWriter, r *http.Request) {
	userID := h.requireAuth(w, r)
	if userID == "" {
		return
	}

	orgID := r.URL.Query().Get("org_id")
	if orgID == "" {
		orgID = r.Header.Get("X-Organization-ID")
	}

	svc := h.getMarketplaceService()
	recommendations, err := svc.GetRecommendations(r.Context(), orgID)
	if err != nil {
		logger.ErrorC("api", fmt.Sprintf("failed to get recommendations: %v", err))
		writeJSONError(w, "failed to get recommendations", http.StatusInternalServerError)
		return
	}

	response := make([]SkillListingResponse, 0, len(recommendations))
	for _, skill := range recommendations {
		response = append(response, SkillListingResponse{
			ID:          skill.ID,
			Name:        skill.Name,
			Description: skill.Description,
			Author:      skill.Author,
			Version:     skill.Version,
			Category:    string(skill.Category),
			Tags:        skill.Tags,
			Price:       skill.Price,
			Rating:      skill.Rating,
			Downloads:   skill.Downloads,
			CreatedAt:   skill.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   skill.UpdatedAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"recommendations": response,
		"total":           len(response),
	})
}
