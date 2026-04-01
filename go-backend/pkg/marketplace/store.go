package marketplace

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type MarketplaceStore interface {
	ListSkills(ctx context.Context, opts ListSkillsOptions) ([]*SkillListing, error)
	GetSkill(ctx context.Context, id string) (*SkillListing, error)
	SearchSkills(ctx context.Context, query string) ([]*SkillListing, error)
	GetSkillByAuthor(ctx context.Context, author string) ([]*SkillListing, error)
	CreateSkill(ctx context.Context, skill *SkillListing) (*SkillListing, error)
	UpdateSkill(ctx context.Context, id string, skill *SkillListing) error
	DeleteSkill(ctx context.Context, id string) error
	AddReview(ctx context.Context, review *SkillReview) (*SkillReview, error)
	GetReviews(ctx context.Context, skillID string) ([]*SkillReview, error)
	PurchaseSkill(ctx context.Context, skillID, orgID string) (*SkillPurchase, error)
	GetPurchases(ctx context.Context, orgID string) ([]*SkillPurchase, error)
	CreateVersion(ctx context.Context, version *SkillVersion) (*SkillVersion, error)
	GetVersions(ctx context.Context, skillID string) ([]*SkillVersion, error)
}

type MemoryMarketplaceStore struct {
	mu             sync.RWMutex
	skills         map[string]*SkillListing
	reviews        map[string][]*SkillReview
	purchases      map[string][]*SkillPurchase
	versions       map[string][]*SkillVersion
	skillsByAuthor map[string][]string
}

func NewMemoryMarketplaceStore() *MemoryMarketplaceStore {
	return &MemoryMarketplaceStore{
		skills:         make(map[string]*SkillListing),
		reviews:        make(map[string][]*SkillReview),
		purchases:      make(map[string][]*SkillPurchase),
		versions:       make(map[string][]*SkillVersion),
		skillsByAuthor: make(map[string][]string),
	}
}

func (s *MemoryMarketplaceStore) ListSkills(ctx context.Context, opts ListSkillsOptions) ([]*SkillListing, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*SkillListing
	for _, skill := range s.skills {
		if opts.Category != "" && skill.Category != opts.Category {
			continue
		}
		if opts.Author != "" && skill.Author != opts.Author {
			continue
		}
		if opts.MinPrice != nil && skill.Price < *opts.MinPrice {
			continue
		}
		if opts.MaxPrice != nil && skill.Price > *opts.MaxPrice {
			continue
		}
		if opts.MinRating != nil && skill.Rating < *opts.MinRating {
			continue
		}
		if len(opts.Tags) > 0 {
			if !containsAnyTag(skill.Tags, opts.Tags) {
				continue
			}
		}
		result = append(result, skill)
	}

	sortSkills(result, opts.SortBy, opts.SortDesc)

	if opts.Limit > 0 && len(result) > opts.Limit {
		result = result[:opts.Limit]
	}

	return result, nil
}

func (s *MemoryMarketplaceStore) GetSkill(ctx context.Context, id string) (*SkillListing, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	skill, ok := s.skills[id]
	if !ok {
		return nil, ErrSkillNotFound
	}
	return skill, nil
}

func (s *MemoryMarketplaceStore) SearchSkills(ctx context.Context, query string) ([]*SkillListing, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(query)
	var result []*SkillListing

	for _, skill := range s.skills {
		if strings.Contains(strings.ToLower(skill.Name), query) ||
			strings.Contains(strings.ToLower(skill.Description), query) ||
			strings.Contains(strings.ToLower(skill.Author), query) ||
			containsTag(skill.Tags, query) {
			result = append(result, skill)
		}
	}

	sortSkills(result, "rating", true)
	return result, nil
}

func (s *MemoryMarketplaceStore) GetSkillByAuthor(ctx context.Context, author string) ([]*SkillListing, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*SkillListing
	for _, id := range s.skillsByAuthor[author] {
		if skill, ok := s.skills[id]; ok {
			result = append(result, skill)
		}
	}
	return result, nil
}

func (s *MemoryMarketplaceStore) CreateSkill(ctx context.Context, skill *SkillListing) (*SkillListing, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	skill.CreatedAt = now
	skill.UpdatedAt = now

	s.skills[skill.ID] = skill
	s.skillsByAuthor[skill.Author] = append(s.skillsByAuthor[skill.Author], skill.ID)

	return skill, nil
}

func (s *MemoryMarketplaceStore) UpdateSkill(ctx context.Context, id string, skill *SkillListing) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.skills[id]
	if !ok {
		return ErrSkillNotFound
	}

	if skill.Author != existing.Author {
		oldAuthor := existing.Author
		newAuthor := skill.Author

		var filtered []string
		for _, sid := range s.skillsByAuthor[oldAuthor] {
			if sid != id {
				filtered = append(filtered, sid)
			}
		}
		s.skillsByAuthor[oldAuthor] = filtered
		s.skillsByAuthor[newAuthor] = append(s.skillsByAuthor[newAuthor], id)
	}

	skill.ID = id
	skill.CreatedAt = existing.CreatedAt
	skill.UpdatedAt = time.Now()
	s.skills[id] = skill

	return nil
}

func (s *MemoryMarketplaceStore) DeleteSkill(ctx context.Context, id string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	skill, ok := s.skills[id]
	if !ok {
		return ErrSkillNotFound
	}

	delete(s.skills, id)

	var filtered []string
	for _, sid := range s.skillsByAuthor[skill.Author] {
		if sid != id {
			filtered = append(filtered, sid)
		}
	}
	s.skillsByAuthor[skill.Author] = filtered

	delete(s.reviews, id)
	delete(s.versions, id)

	return nil
}

func (s *MemoryMarketplaceStore) AddReview(ctx context.Context, review *SkillReview) (*SkillReview, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.skills[review.SkillID]; !ok {
		return nil, ErrSkillNotFound
	}

	now := time.Now()
	review.CreatedAt = now
	review.UpdatedAt = now

	s.reviews[review.SkillID] = append(s.reviews[review.SkillID], review)

	s.updateSkillRating(review.SkillID)

	return review, nil
}

func (s *MemoryMarketplaceStore) GetReviews(ctx context.Context, skillID string) ([]*SkillReview, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.skills[skillID]; !ok {
		return nil, ErrSkillNotFound
	}

	reviews := s.reviews[skillID]
	result := make([]*SkillReview, len(reviews))
	copy(result, reviews)
	return result, nil
}

func (s *MemoryMarketplaceStore) PurchaseSkill(ctx context.Context, skillID, orgID string) (*SkillPurchase, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.skills[skillID]; !ok {
		return nil, ErrSkillNotFound
	}

	for _, p := range s.purchases[orgID] {
		if p.SkillID == skillID {
			return nil, ErrAlreadyPurchased
		}
	}

	purchase := &SkillPurchase{
		ID:             generateID(),
		SkillID:        skillID,
		OrganizationID: orgID,
		PurchasedAt:    time.Now(),
	}

	s.purchases[orgID] = append(s.purchases[orgID], purchase)

	if skill, ok := s.skills[skillID]; ok {
		skill.Downloads++
	}

	return purchase, nil
}

func (s *MemoryMarketplaceStore) GetPurchases(ctx context.Context, orgID string) ([]*SkillPurchase, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	purchases := s.purchases[orgID]
	result := make([]*SkillPurchase, len(purchases))
	copy(result, purchases)
	return result, nil
}

func (s *MemoryMarketplaceStore) CreateVersion(ctx context.Context, version *SkillVersion) (*SkillVersion, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.skills[version.SkillID]; !ok {
		return nil, ErrSkillNotFound
	}

	for _, v := range s.versions[version.SkillID] {
		if v.Version == version.Version {
			return nil, ErrVersionExists
		}
	}

	version.ID = generateID()
	version.CreatedAt = time.Now()

	s.versions[version.SkillID] = append(s.versions[version.SkillID], version)

	if skill, ok := s.skills[version.SkillID]; ok {
		skill.Version = version.Version
		skill.UpdatedAt = time.Now()
	}

	return version, nil
}

func (s *MemoryMarketplaceStore) GetVersions(ctx context.Context, skillID string) ([]*SkillVersion, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.skills[skillID]; !ok {
		return nil, ErrSkillNotFound
	}

	versions := s.versions[skillID]
	result := make([]*SkillVersion, len(versions))
	copy(result, versions)
	return result, nil
}

func (s *MemoryMarketplaceStore) updateSkillRating(skillID string) {
	reviews := s.reviews[skillID]
	if len(reviews) == 0 {
		return
	}

	var total int
	for _, r := range reviews {
		total += r.Rating
	}

	if skill, ok := s.skills[skillID]; ok {
		skill.Rating = float64(total) / float64(len(reviews))
	}
}

func sortSkills(skills []*SkillListing, sortBy string, desc bool) {
	if sortBy == "" {
		sortBy = "created_at"
	}

	sort.Slice(skills, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "name":
			less = skills[i].Name < skills[j].Name
		case "rating":
			less = skills[i].Rating < skills[j].Rating
		case "downloads":
			less = skills[i].Downloads < skills[j].Downloads
		case "price":
			less = skills[i].Price < skills[j].Price
		case "created_at":
			less = skills[i].CreatedAt.Before(skills[j].CreatedAt)
		default:
			less = skills[i].CreatedAt.Before(skills[j].CreatedAt)
		}
		if desc {
			return !less
		}
		return less
	})
}

func containsAnyTag(skillTags, searchTags []string) bool {
	for _, st := range searchTags {
		for _, t := range skillTags {
			if strings.EqualFold(t, st) {
				return true
			}
		}
	}
	return false
}

func containsTag(tags []string, query string) bool {
	for _, t := range tags {
		if strings.Contains(strings.ToLower(t), query) {
			return true
		}
	}
	return false
}

func generateID() string {
	return time.Now().Format("20060102150405") + randomSuffix()
}

func randomSuffix() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars[time.Now().Nanosecond()%len(chars)]
	}
	return string(b)
}
