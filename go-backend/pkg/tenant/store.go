package tenant

import (
	"context"
	"sync"
	"time"
)

type TenantStore interface {
	GetOrganization(ctx context.Context, id string) (*Organization, error)
	GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error)
	CreateOrganization(ctx context.Context, org *Organization) error
	UpdateOrganization(ctx context.Context, org *Organization) error
	DeleteOrganization(ctx context.Context, id string) error

	GetMembership(ctx context.Context, orgID, userID string) (*Membership, error)
	CreateMembership(ctx context.Context, membership *Membership) error
	UpdateMembership(ctx context.Context, membership *Membership) error
	DeleteMembership(ctx context.Context, id string) error
	ListMemberships(ctx context.Context, orgID string) ([]*Membership, error)

	GetSubscription(ctx context.Context, orgID string) (*Subscription, error)
	CreateSubscription(ctx context.Context, sub *Subscription) error
	UpdateSubscription(ctx context.Context, sub *Subscription) error

	CreateUsageRecord(ctx context.Context, record *UsageRecord) error
	GetMonthlyUsage(ctx context.Context, orgID string, eventType string) (int64, error)
}

type MemoryTenantStore struct {
	mu            sync.RWMutex
	organizations map[string]*Organization
	orgsBySlug    map[string]*Organization
	memberships   map[string]*Membership
	membersByOrg  map[string][]string
	membersByUser map[string][]string
	subscriptions map[string]*Subscription
	subsByOrg     map[string]string
	usageRecords  []*UsageRecord
}

func NewMemoryTenantStore() *MemoryTenantStore {
	return &MemoryTenantStore{
		organizations: make(map[string]*Organization),
		orgsBySlug:    make(map[string]*Organization),
		memberships:   make(map[string]*Membership),
		membersByOrg:  make(map[string][]string),
		membersByUser: make(map[string][]string),
		subscriptions: make(map[string]*Subscription),
		subsByOrg:     make(map[string]string),
		usageRecords:  make([]*UsageRecord, 0),
	}
}

func (s *MemoryTenantStore) GetOrganization(ctx context.Context, id string) (*Organization, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	org, ok := s.organizations[id]
	if !ok {
		return nil, ErrOrgNotFound
	}
	return org, nil
}

func (s *MemoryTenantStore) GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	org, ok := s.orgsBySlug[slug]
	if !ok {
		return nil, ErrOrgNotFound
	}
	return org, nil
}

func (s *MemoryTenantStore) CreateOrganization(ctx context.Context, org *Organization) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.organizations[org.ID]; exists {
		return ErrOrgNotFound
	}
	if _, exists := s.orgsBySlug[org.Slug]; exists {
		return ErrOrgNotFound
	}

	now := time.Now()
	org.CreatedAt = now
	org.UpdatedAt = now

	s.organizations[org.ID] = org
	s.orgsBySlug[org.Slug] = org
	return nil
}

func (s *MemoryTenantStore) UpdateOrganization(ctx context.Context, org *Organization) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.organizations[org.ID]
	if !ok {
		return ErrOrgNotFound
	}

	if org.Slug != existing.Slug {
		delete(s.orgsBySlug, existing.Slug)
		s.orgsBySlug[org.Slug] = org
	}

	org.UpdatedAt = time.Now()
	org.CreatedAt = existing.CreatedAt
	s.organizations[org.ID] = org
	return nil
}

func (s *MemoryTenantStore) DeleteOrganization(ctx context.Context, id string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	org, ok := s.organizations[id]
	if !ok {
		return ErrOrgNotFound
	}

	now := time.Now()
	org.DeletedAt = &now
	org.UpdatedAt = now
	return nil
}

func (s *MemoryTenantStore) GetMembership(ctx context.Context, orgID, userID string) (*Membership, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, id := range s.membersByOrg[orgID] {
		m, ok := s.memberships[id]
		if ok && m.UserID == userID {
			return m, nil
		}
	}
	return nil, ErrMembershipNotFound
}

func (s *MemoryTenantStore) CreateMembership(ctx context.Context, membership *Membership) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.memberships[membership.ID]; exists {
		return ErrAlreadyMember
	}

	now := time.Now()
	membership.CreatedAt = now
	membership.UpdatedAt = now

	s.memberships[membership.ID] = membership
	s.membersByOrg[membership.OrganizationID] = append(s.membersByOrg[membership.OrganizationID], membership.ID)
	s.membersByUser[membership.UserID] = append(s.membersByUser[membership.UserID], membership.ID)
	return nil
}

func (s *MemoryTenantStore) UpdateMembership(ctx context.Context, membership *Membership) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.memberships[membership.ID]
	if !ok {
		return ErrMembershipNotFound
	}

	membership.CreatedAt = existing.CreatedAt
	membership.UpdatedAt = time.Now()
	membership.OrganizationID = existing.OrganizationID
	membership.UserID = existing.UserID
	s.memberships[membership.ID] = membership
	return nil
}

func (s *MemoryTenantStore) DeleteMembership(ctx context.Context, id string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.memberships[id]
	if !ok {
		return ErrMembershipNotFound
	}

	delete(s.memberships, id)

	orgMembers := s.membersByOrg[m.OrganizationID]
	for i, mid := range orgMembers {
		if mid == id {
			s.membersByOrg[m.OrganizationID] = append(orgMembers[:i], orgMembers[i+1:]...)
			break
		}
	}

	userMembers := s.membersByUser[m.UserID]
	for i, mid := range userMembers {
		if mid == id {
			s.membersByUser[m.UserID] = append(userMembers[:i], userMembers[i+1:]...)
			break
		}
	}
	return nil
}

func (s *MemoryTenantStore) ListMemberships(ctx context.Context, orgID string) ([]*Membership, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.membersByOrg[orgID]
	result := make([]*Membership, 0, len(ids))
	for _, id := range ids {
		if m, ok := s.memberships[id]; ok {
			result = append(result, m)
		}
	}
	return result, nil
}

func (s *MemoryTenantStore) GetSubscription(ctx context.Context, orgID string) (*Subscription, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	subID, ok := s.subsByOrg[orgID]
	if !ok {
		return nil, ErrSubscriptionNotFound
	}
	sub, ok := s.subscriptions[subID]
	if !ok {
		return nil, ErrSubscriptionNotFound
	}
	return sub, nil
}

func (s *MemoryTenantStore) CreateSubscription(ctx context.Context, sub *Subscription) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	sub.CreatedAt = now
	sub.UpdatedAt = now

	s.subscriptions[sub.ID] = sub
	s.subsByOrg[sub.OrganizationID] = sub.ID
	return nil
}

func (s *MemoryTenantStore) UpdateSubscription(ctx context.Context, sub *Subscription) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.subscriptions[sub.ID]
	if !ok {
		return ErrSubscriptionNotFound
	}

	sub.CreatedAt = existing.CreatedAt
	sub.UpdatedAt = time.Now()
	sub.OrganizationID = existing.OrganizationID
	s.subscriptions[sub.ID] = sub
	return nil
}

func (s *MemoryTenantStore) CreateUsageRecord(ctx context.Context, record *UsageRecord) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record.CreatedAt = time.Now()
	s.usageRecords = append(s.usageRecords, record)
	return nil
}

func (s *MemoryTenantStore) GetMonthlyUsage(ctx context.Context, orgID string, eventType string) (int64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	var total int64
	for _, r := range s.usageRecords {
		if r.OrganizationID == orgID && r.EventType == eventType && r.CreatedAt.After(startOfMonth) {
			total += r.Quantity
		}
	}
	return total, nil
}

func (s *MemoryTenantStore) ListOrganizationsForUser(ctx context.Context, userID string) ([]*Organization, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	memberIDs := s.membersByUser[userID]
	orgs := make([]*Organization, 0, len(memberIDs))
	seen := make(map[string]struct{})

	for _, mid := range memberIDs {
		m, ok := s.memberships[mid]
		if !ok {
			continue
		}
		if _, exists := seen[m.OrganizationID]; exists {
			continue
		}
		seen[m.OrganizationID] = struct{}{}

		org, ok := s.organizations[m.OrganizationID]
		if ok && org.DeletedAt == nil {
			orgs = append(orgs, org)
		}
	}

	return orgs, nil
}
