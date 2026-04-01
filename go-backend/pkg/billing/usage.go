package billing

import (
	"context"
	"errors"
	"time"
)

var (
	ErrUsageLimitExceeded = errors.New("usage limit exceeded")
)

type LimitType string

const (
	LimitTypeMessages LimitType = "messages"
	LimitTypeTokens   LimitType = "tokens"
	LimitTypeUsers    LimitType = "users"
	LimitTypeAgents   LimitType = "agents"
	LimitTypeChannels LimitType = "channels"
	LimitTypeStorage  LimitType = "storage"
)

type UsageStore interface {
	CreateUsageEvent(ctx context.Context, event *UsageEvent) error
	GetUsageSummary(ctx context.Context, orgID string, periodStart, periodEnd time.Time) (*UsageSummary, error)
	GetMonthlyUsage(ctx context.Context, orgID string, eventType string) (int64, error)
	ResetMonthlyCounters(ctx context.Context, orgID string) error
}

type UsageTracker interface {
	TrackMessage(ctx context.Context, orgID string) error
	TrackTokens(ctx context.Context, orgID string, inputCount, outputCount int64) error
	TrackStorage(ctx context.Context, orgID string, bytes int64) error
	GetUsage(ctx context.Context, orgID string) (*UsageSummary, error)
	CheckLimit(ctx context.Context, orgID string, limitType LimitType) (bool, error)
	ResetMonthlyCounters(ctx context.Context, orgID string) error
}

type UsageService struct {
	store     UsageStore
	planStore PlanStore
}

type PlanStore interface {
	GetSubscription(ctx context.Context, orgID string) (*Subscription, error)
}

func NewUsageService(store UsageStore, planStore PlanStore) *UsageService {
	return &UsageService{
		store:     store,
		planStore: planStore,
	}
}

func (s *UsageService) TrackMessage(ctx context.Context, orgID string) error {
	event := &UsageEvent{
		ID:        generateUsageID(),
		OrgID:     orgID,
		EventType: "message",
		Quantity:  1,
		CreatedAt: time.Now(),
	}
	return s.store.CreateUsageEvent(ctx, event)
}

func (s *UsageService) TrackTokens(ctx context.Context, orgID string, inputCount, outputCount int64) error {
	now := time.Now()

	inputEvent := &UsageEvent{
		ID:        generateUsageID(),
		OrgID:     orgID,
		EventType: "token_input",
		Quantity:  inputCount,
		CreatedAt: now,
	}
	if err := s.store.CreateUsageEvent(ctx, inputEvent); err != nil {
		return err
	}

	outputEvent := &UsageEvent{
		ID:        generateUsageID(),
		OrgID:     orgID,
		EventType: "token_output",
		Quantity:  outputCount,
		CreatedAt: now,
	}
	return s.store.CreateUsageEvent(ctx, outputEvent)
}

func (s *UsageService) TrackStorage(ctx context.Context, orgID string, bytes int64) error {
	event := &UsageEvent{
		ID:        generateUsageID(),
		OrgID:     orgID,
		EventType: "storage_bytes",
		Quantity:  bytes,
		CreatedAt: time.Now(),
	}
	return s.store.CreateUsageEvent(ctx, event)
}

func (s *UsageService) GetUsage(ctx context.Context, orgID string) (*UsageSummary, error) {
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)

	return s.store.GetUsageSummary(ctx, orgID, periodStart, periodEnd)
}

func (s *UsageService) CheckLimit(ctx context.Context, orgID string, limitType LimitType) (bool, error) {
	sub, err := s.planStore.GetSubscription(ctx, orgID)
	if err != nil {
		return false, err
	}

	limits := GetPlanLimits(sub.PlanID)

	switch limitType {
	case LimitTypeMessages:
		if IsUnlimited(limits.MaxMessages) {
			return true, nil
		}
		usage, err := s.store.GetMonthlyUsage(ctx, orgID, "message")
		if err != nil {
			return false, err
		}
		return usage < int64(limits.MaxMessages), nil

	case LimitTypeUsers:
		return true, nil

	case LimitTypeAgents:
		return true, nil

	case LimitTypeChannels:
		return true, nil

	case LimitTypeStorage:
		if IsStorageUnlimited(limits.MaxStorageBytes) {
			return true, nil
		}
		summary, err := s.GetUsage(ctx, orgID)
		if err != nil {
			return false, err
		}
		return summary.Storage < limits.MaxStorageBytes, nil

	case LimitTypeTokens:
		return true, nil
	}

	return true, nil
}

func (s *UsageService) ResetMonthlyCounters(ctx context.Context, orgID string) error {
	return s.store.ResetMonthlyCounters(ctx, orgID)
}

func (s *UsageService) IncrementUsage(ctx context.Context, orgID string, eventType string, quantity int64, metadata map[string]interface{}) error {
	event := &UsageEvent{
		ID:        generateUsageID(),
		OrgID:     orgID,
		EventType: eventType,
		Quantity:  quantity,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}
	return s.store.CreateUsageEvent(ctx, event)
}

func generateUsageID() string {
	return time.Now().Format("20060102150405.999999999")
}
