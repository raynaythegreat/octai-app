package tenant

import (
	"context"
	"errors"
	"time"
)

type TenantService struct {
	store TenantStore
}

func NewTenantService(store TenantStore) *TenantService {
	return &TenantService{store: store}
}

type CreateOrganizationInput struct {
	ID      string
	Slug    string
	Name    string
	LogoURL *string
	UserID  string
}

func (s *TenantService) CreateOrganization(ctx context.Context, input CreateOrganizationInput) (*Organization, error) {
	if err := ValidateSlug(input.Slug); err != nil {
		return nil, err
	}
	if input.Name == "" {
		return nil, errors.New("name is required")
	}
	if input.UserID == "" {
		return nil, errors.New("user_id is required")
	}

	org := &Organization{
		ID:       input.ID,
		Slug:     input.Slug,
		Name:     input.Name,
		LogoURL:  input.LogoURL,
		Settings: make(JSONB),
	}

	if err := s.store.CreateOrganization(ctx, org); err != nil {
		return nil, err
	}

	membership := &Membership{
		ID:             generateID(),
		OrganizationID: org.ID,
		UserID:         input.UserID,
		Role:           RoleOwner,
	}
	now := time.Now()
	membership.AcceptedAt = &now

	if err := s.store.CreateMembership(ctx, membership); err != nil {
		return nil, err
	}

	sub := &Subscription{
		ID:             generateID(),
		OrganizationID: org.ID,
		Status:         SubStatusActive,
		Tier:           TierFree,
	}

	if err := s.store.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	return org, nil
}

type AddMemberInput struct {
	OrganizationID string
	UserID         string
	Role           Role
	InvitedBy      string
}

func (s *TenantService) AddMember(ctx context.Context, input AddMemberInput) (*Membership, error) {
	if !input.Role.Valid() {
		return nil, ErrInvalidRole
	}

	_, err := s.store.GetOrganization(ctx, input.OrganizationID)
	if err != nil {
		return nil, err
	}

	existing, _ := s.store.GetMembership(ctx, input.OrganizationID, input.UserID)
	if existing != nil {
		return nil, ErrAlreadyMember
	}

	if err := s.checkMemberLimit(ctx, input.OrganizationID); err != nil {
		return nil, err
	}

	membership := &Membership{
		ID:             generateID(),
		OrganizationID: input.OrganizationID,
		UserID:         input.UserID,
		Role:           input.Role,
		InvitedBy:      &input.InvitedBy,
	}
	now := time.Now()
	membership.InvitedAt = &now

	if err := s.store.CreateMembership(ctx, membership); err != nil {
		return nil, err
	}

	return membership, nil
}

func (s *TenantService) RemoveMember(ctx context.Context, orgID, membershipID string) error {
	membership, err := s.store.GetMembership(ctx, orgID, "")
	if err != nil {
		return err
	}

	members, err := s.store.ListMemberships(ctx, orgID)
	if err != nil {
		return err
	}

	for _, m := range members {
		if m.ID == membershipID {
			membership = m
			break
		}
	}

	if membership == nil || membership.ID != membershipID {
		return ErrMembershipNotFound
	}

	if membership.Role == RoleOwner {
		ownerCount := 0
		for _, m := range members {
			if m.Role == RoleOwner {
				ownerCount++
			}
		}
		if ownerCount <= 1 {
			return ErrCannotRemoveOwner
		}
	}

	return s.store.DeleteMembership(ctx, membershipID)
}

func (s *TenantService) UpdateRole(ctx context.Context, orgID, membershipID string, newRole Role) error {
	if !newRole.Valid() {
		return ErrInvalidRole
	}

	members, err := s.store.ListMemberships(ctx, orgID)
	if err != nil {
		return err
	}

	var membership *Membership
	for _, m := range members {
		if m.ID == membershipID {
			membership = m
			break
		}
	}

	if membership == nil {
		return ErrMembershipNotFound
	}

	if membership.Role == RoleOwner && newRole != RoleOwner {
		ownerCount := 0
		for _, m := range members {
			if m.Role == RoleOwner {
				ownerCount++
			}
		}
		if ownerCount <= 1 {
			return errors.New("cannot demote the last owner")
		}
	}

	membership.Role = newRole
	return s.store.UpdateMembership(ctx, membership)
}

func (s *TenantService) CheckPermission(ctx context.Context, orgID, userID string, perm Permission) (bool, error) {
	membership, err := s.store.GetMembership(ctx, orgID, userID)
	if err != nil {
		if errors.Is(err, ErrMembershipNotFound) {
			return false, nil
		}
		return false, err
	}

	return HasPermission(membership.Role, perm), nil
}

func (s *TenantService) GetTenantContext(ctx context.Context, orgID, userID string) (*TenantContext, error) {
	org, err := s.store.GetOrganization(ctx, orgID)
	if err != nil {
		return nil, err
	}

	membership, err := s.store.GetMembership(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}

	sub, err := s.store.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, err
	}

	return &TenantContext{
		OrganizationID:   org.ID,
		OrganizationSlug: org.Slug,
		UserID:           userID,
		Role:             membership.Role,
		SubscriptionTier: sub.Tier,
	}, nil
}

func (s *TenantService) GetOrganization(ctx context.Context, id string) (*Organization, error) {
	return s.store.GetOrganization(ctx, id)
}

func (s *TenantService) GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error) {
	return s.store.GetOrganizationBySlug(ctx, slug)
}

func (s *TenantService) UpdateOrganization(ctx context.Context, org *Organization) error {
	if err := ValidateSlug(org.Slug); err != nil {
		return err
	}
	if org.Name == "" {
		return errors.New("name is required")
	}
	return s.store.UpdateOrganization(ctx, org)
}

func (s *TenantService) ListMemberships(ctx context.Context, orgID string) ([]*Membership, error) {
	return s.store.ListMemberships(ctx, orgID)
}

func (s *TenantService) GetSubscription(ctx context.Context, orgID string) (*Subscription, error) {
	return s.store.GetSubscription(ctx, orgID)
}

func (s *TenantService) RecordUsage(ctx context.Context, record *UsageRecord) error {
	if record.ID == "" {
		record.ID = generateID()
	}
	return s.store.CreateUsageRecord(ctx, record)
}

func (s *TenantService) CheckUsageLimit(ctx context.Context, orgID, limitType string) (bool, error) {
	tc, err := s.GetTenantContext(ctx, orgID, "")
	if err != nil {
		return false, err
	}

	limits := tc.GetLimits()

	switch limitType {
	case "users":
		members, err := s.store.ListMemberships(ctx, orgID)
		if err != nil {
			return false, err
		}
		return limits.MaxUsers < 0 || len(members) < limits.MaxUsers, nil
	case "messages":
		usage, err := s.store.GetMonthlyUsage(ctx, orgID, "message")
		if err != nil {
			return false, err
		}
		return limits.MaxMessages < 0 || int(usage) < limits.MaxMessages, nil
	}

	return true, nil
}

func (s *TenantService) checkMemberLimit(ctx context.Context, orgID string) error {
	tc, err := s.GetTenantContext(ctx, orgID, "")
	if err != nil {
		return err
	}

	limits := tc.GetLimits()
	if limits.MaxUsers < 0 {
		return nil
	}

	members, err := s.store.ListMemberships(ctx, orgID)
	if err != nil {
		return err
	}

	if len(members) >= limits.MaxUsers {
		return ErrLimitExceeded
	}
	return nil
}

func HasPermission(role Role, perm Permission) bool {
	perms, ok := RolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
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
