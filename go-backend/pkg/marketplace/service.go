package marketplace

import (
	"context"
	"errors"
	"time"
)

type MarketplaceService struct {
	store MarketplaceStore
}

func NewMarketplaceService(store MarketplaceStore) *MarketplaceService {
	return &MarketplaceService{store: store}
}

func (s *MarketplaceService) PublishSkill(ctx context.Context, input PublishSkillInput) (*SkillListing, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	skill := &SkillListing{
		ID:          input.ID,
		Name:        input.Name,
		Description: input.Description,
		Author:      input.Author,
		Version:     input.Version,
		Category:    input.Category,
		Tags:        input.Tags,
		Price:       input.Price,
	}

	if skill.ID == "" {
		skill.ID = generateID()
	}
	if skill.Version == "" {
		skill.Version = "1.0.0"
	}

	return s.store.CreateSkill(ctx, skill)
}

func (s *MarketplaceService) InstallSkill(ctx context.Context, skillID, orgID string) error {
	_, err := s.store.GetSkill(ctx, skillID)
	if err != nil {
		return err
	}

	_, err = s.store.PurchaseSkill(ctx, skillID, orgID)
	if errors.Is(err, ErrAlreadyPurchased) {
		return nil
	}
	return err
}

func (s *MarketplaceService) UninstallSkill(ctx context.Context, skillID, orgID string) error {
	purchases, err := s.store.GetPurchases(ctx, orgID)
	if err != nil {
		return err
	}

	for _, p := range purchases {
		if p.SkillID == skillID {
			return s.deletePurchase(ctx, p.ID, orgID)
		}
	}

	return ErrPurchaseNotFound
}

func (s *MarketplaceService) UpdateSkill(ctx context.Context, skillID, version string) error {
	skill, err := s.store.GetSkill(ctx, skillID)
	if err != nil {
		return err
	}

	versions, err := s.store.GetVersions(ctx, skillID)
	if err != nil {
		return err
	}

	var targetVersion *SkillVersion
	for _, v := range versions {
		if v.Version == version {
			targetVersion = v
			break
		}
	}

	if targetVersion == nil {
		return errors.New("version not found")
	}

	skill.Version = version
	skill.UpdatedAt = time.Now()

	return s.store.UpdateSkill(ctx, skillID, skill)
}

func (s *MarketplaceService) RateSkill(ctx context.Context, skillID, userID string, rating int, comment string) error {
	if rating < 1 || rating > 5 {
		return ErrInvalidRating
	}

	_, err := s.store.GetSkill(ctx, skillID)
	if err != nil {
		return err
	}

	review := &SkillReview{
		ID:      generateID(),
		SkillID: skillID,
		UserID:  userID,
		Rating:  rating,
		Comment: comment,
	}

	_, err = s.store.AddReview(ctx, review)
	return err
}

func (s *MarketplaceService) GetRecommendations(ctx context.Context, orgID string) ([]*SkillListing, error) {
	purchases, err := s.store.GetPurchases(ctx, orgID)
	if err != nil {
		return nil, err
	}

	purchasedIDs := make(map[string]bool)
	for _, p := range purchases {
		purchasedIDs[p.SkillID] = true
	}

	purchasedCategories := make(map[SkillCategory]int)
	for _, p := range purchases {
		skill, err := s.store.GetSkill(ctx, p.SkillID)
		if err == nil {
			purchasedCategories[skill.Category]++
		}
	}

	allSkills, err := s.store.ListSkills(ctx, ListSkillsOptions{
		SortBy:   "rating",
		SortDesc: true,
		Limit:    100,
	})
	if err != nil {
		return nil, err
	}

	var recommendations []*SkillListing
	for _, skill := range allSkills {
		if purchasedIDs[skill.ID] {
			continue
		}
		recommendations = append(recommendations, skill)
	}

	if len(recommendations) > 10 {
		recommendations = recommendations[:10]
	}

	return recommendations, nil
}

func (s *MarketplaceService) GetSkill(ctx context.Context, id string) (*SkillListing, error) {
	return s.store.GetSkill(ctx, id)
}

func (s *MarketplaceService) ListSkills(ctx context.Context, opts ListSkillsOptions) ([]*SkillListing, error) {
	return s.store.ListSkills(ctx, opts)
}

func (s *MarketplaceService) SearchSkills(ctx context.Context, query string) ([]*SkillListing, error) {
	return s.store.SearchSkills(ctx, query)
}

func (s *MarketplaceService) GetReviews(ctx context.Context, skillID string) ([]*SkillReview, error) {
	return s.store.GetReviews(ctx, skillID)
}

func (s *MarketplaceService) GetPurchases(ctx context.Context, orgID string) ([]*SkillPurchase, error) {
	return s.store.GetPurchases(ctx, orgID)
}

func (s *MarketplaceService) AddVersion(ctx context.Context, skillID, version, changelog, downloadURL string) (*SkillVersion, error) {
	_, err := s.store.GetSkill(ctx, skillID)
	if err != nil {
		return nil, err
	}

	v := &SkillVersion{
		SkillID:     skillID,
		Version:     version,
		Changelog:   changelog,
		DownloadURL: downloadURL,
	}

	return s.store.CreateVersion(ctx, v)
}

func (s *MarketplaceService) GetVersions(ctx context.Context, skillID string) ([]*SkillVersion, error) {
	return s.store.GetVersions(ctx, skillID)
}

func (s *MarketplaceService) deletePurchase(ctx context.Context, purchaseID, orgID string) error {
	purchases, err := s.store.GetPurchases(ctx, orgID)
	if err != nil {
		return err
	}

	var filtered []*SkillPurchase
	for _, p := range purchases {
		if p.ID != purchaseID {
			filtered = append(filtered, p)
		}
	}

	if len(filtered) == len(purchases) {
		return ErrPurchaseNotFound
	}

	memStore, ok := s.store.(*MemoryMarketplaceStore)
	if !ok {
		return errors.New("store does not support delete operation")
	}

	memStore.mu.Lock()
	memStore.purchases[orgID] = filtered
	memStore.mu.Unlock()

	return nil
}
