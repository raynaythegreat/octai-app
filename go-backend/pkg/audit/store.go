package audit

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrLogNotFound      = errors.New("audit log not found")
	ErrInvalidDateRange = errors.New("invalid date range")
	ErrOrgRequired      = errors.New("organization ID required")
)

type AuditStore interface {
	Store(ctx context.Context, logs []*AuditLog) error
	Query(ctx context.Context, orgID string, filters QueryFilters) (*QueryResult, error)
	GetByUser(ctx context.Context, orgID, userID string, limit int) ([]*AuditLog, error)
	GetByResource(ctx context.Context, orgID string, resourceType ResourceType, resourceID string) ([]*AuditLog, error)
	GetByID(ctx context.Context, orgID, logID string) (*AuditLog, error)
	DeleteOlderThan(ctx context.Context, olderThan time.Time) (int64, error)
	DeleteByOrganization(ctx context.Context, orgID string, olderThan time.Time) (int64, error)
	GetRetentionSettings(ctx context.Context, orgID string) (int, error)
	SetRetentionSettings(ctx context.Context, orgID string, days int) error
}

type MemoryAuditStore struct {
	mu             sync.RWMutex
	logs           []*AuditLog
	logsByOrg      map[string][]int
	logsByUser     map[string][]int
	logsByResource map[string][]int
	retentionDays  map[string]int
	maxLogs        int
}

func NewMemoryAuditStore(maxLogs int) *MemoryAuditStore {
	if maxLogs <= 0 {
		maxLogs = 100000
	}
	return &MemoryAuditStore{
		logs:           make([]*AuditLog, 0, 1000),
		logsByOrg:      make(map[string][]int),
		logsByUser:     make(map[string][]int),
		logsByResource: make(map[string][]int),
		retentionDays:  make(map[string]int),
		maxLogs:        maxLogs,
	}
}

func (s *MemoryAuditStore) Store(ctx context.Context, logs []*AuditLog) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	available := s.maxLogs - len(s.logs)
	if available < len(logs) {
		overflow := len(logs) - available
		if overflow < len(s.logs) {
			s.evictOldest(overflow)
		} else {
			s.logs = s.logs[:0]
			s.logsByOrg = make(map[string][]int)
			s.logsByUser = make(map[string][]int)
			s.logsByResource = make(map[string][]int)
		}
	}

	startIdx := len(s.logs)
	for i, log := range logs {
		idx := startIdx + i
		s.logs = append(s.logs, log)

		orgKey := log.OrganizationID
		s.logsByOrg[orgKey] = append(s.logsByOrg[orgKey], idx)

		userKey := log.OrganizationID + ":" + log.UserID
		s.logsByUser[userKey] = append(s.logsByUser[userKey], idx)

		resourceKey := log.OrganizationID + ":" + string(log.ResourceType) + ":" + log.ResourceID
		s.logsByResource[resourceKey] = append(s.logsByResource[resourceKey], idx)
	}

	return nil
}

func (s *MemoryAuditStore) evictOldest(count int) {
	if count >= len(s.logs) {
		s.logs = s.logs[:0]
		return
	}

	evicted := s.logs[:count]
	s.logs = s.logs[count:]

	for _, log := range evicted {
		s.removeFromIndexes(log)
	}
}

func (s *MemoryAuditStore) removeFromIndexes(log *AuditLog) {
	orgKey := log.OrganizationID
	orgSlice := s.logsByOrg[orgKey]
	s.removeIndexFromSlice(&orgSlice, log.ID)
	s.logsByOrg[orgKey] = orgSlice

	userKey := log.OrganizationID + ":" + log.UserID
	userSlice := s.logsByUser[userKey]
	s.removeIndexFromSlice(&userSlice, log.ID)
	s.logsByUser[userKey] = userSlice

	resourceKey := log.OrganizationID + ":" + string(log.ResourceType) + ":" + log.ResourceID
	resourceSlice := s.logsByResource[resourceKey]
	s.removeIndexFromSlice(&resourceSlice, log.ID)
	s.logsByResource[resourceKey] = resourceSlice
}

func (s *MemoryAuditStore) removeIndexFromSlice(slice *[]int, logID string) {
	for i, idx := range *slice {
		if idx < len(s.logs) && s.logs[idx].ID == logID {
			*slice = append((*slice)[:i], (*slice)[i+1:]...)
			return
		}
	}
}

func (s *MemoryAuditStore) Query(ctx context.Context, orgID string, filters QueryFilters) (*QueryResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if orgID == "" {
		return nil, ErrOrgRequired
	}

	if !filters.StartDate.IsZero() && !filters.EndDate.IsZero() && filters.StartDate.After(filters.EndDate) {
		return nil, ErrInvalidDateRange
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	indices := s.logsByOrg[orgID]
	var result []*AuditLog

	for _, idx := range indices {
		if idx >= len(s.logs) {
			continue
		}

		log := s.logs[idx]
		if !s.matchesFilters(log, filters) {
			continue
		}
		result = append(result, log)
	}

	total := int64(len(result))

	if filters.SortOrder == "" || filters.SortOrder == "desc" {
		for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
			result[i], result[j] = result[j], result[i]
		}
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filters.Offset
	if offset < 0 {
		offset = 0
	}

	hasMore := false
	if offset < len(result) {
		end := offset + limit
		if end > len(result) {
			end = len(result)
		} else {
			hasMore = true
		}
		result = result[offset:end]
	} else {
		result = []*AuditLog{}
	}

	return &QueryResult{
		Logs:    result,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: hasMore,
	}, nil
}

func (s *MemoryAuditStore) matchesFilters(log *AuditLog, filters QueryFilters) bool {
	if filters.UserID != "" && log.UserID != filters.UserID {
		return false
	}
	if filters.Action != "" && log.Action != filters.Action {
		return false
	}
	if filters.ResourceType != "" && log.ResourceType != filters.ResourceType {
		return false
	}
	if filters.ResourceID != "" && log.ResourceID != filters.ResourceID {
		return false
	}
	if filters.Status != "" && log.Status != filters.Status {
		return false
	}
	if filters.IPAddress != "" && log.IPAddress != filters.IPAddress {
		return false
	}
	if !filters.StartDate.IsZero() && log.Timestamp.Before(filters.StartDate) {
		return false
	}
	if !filters.EndDate.IsZero() && log.Timestamp.After(filters.EndDate) {
		return false
	}
	return true
}

func (s *MemoryAuditStore) GetByUser(ctx context.Context, orgID, userID string, limit int) ([]*AuditLog, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	userKey := orgID + ":" + userID
	indices := s.logsByUser[userKey]

	var result []*AuditLog
	for i := len(indices) - 1; i >= 0 && (limit <= 0 || len(result) < limit); i-- {
		idx := indices[i]
		if idx < len(s.logs) {
			result = append(result, s.logs[idx])
		}
	}

	return result, nil
}

func (s *MemoryAuditStore) GetByResource(ctx context.Context, orgID string, resourceType ResourceType, resourceID string) ([]*AuditLog, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	resourceKey := orgID + ":" + string(resourceType) + ":" + resourceID
	indices := s.logsByResource[resourceKey]

	result := make([]*AuditLog, 0, len(indices))
	for _, idx := range indices {
		if idx < len(s.logs) {
			result = append(result, s.logs[idx])
		}
	}

	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result, nil
}

func (s *MemoryAuditStore) GetByID(ctx context.Context, orgID, logID string) (*AuditLog, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, log := range s.logs {
		if log.ID == logID && log.OrganizationID == orgID {
			return log, nil
		}
	}
	return nil, ErrLogNotFound
}

func (s *MemoryAuditStore) DeleteOlderThan(ctx context.Context, olderThan time.Time) (int64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var newLogs []*AuditLog
	var deleted int64

	for _, log := range s.logs {
		if log.Timestamp.Before(olderThan) {
			deleted++
		} else {
			newLogs = append(newLogs, log)
		}
	}

	s.logs = newLogs
	s.rebuildIndexes()

	return deleted, nil
}

func (s *MemoryAuditStore) DeleteByOrganization(ctx context.Context, orgID string, olderThan time.Time) (int64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var newLogs []*AuditLog
	var deleted int64

	for _, log := range s.logs {
		if log.OrganizationID == orgID && (olderThan.IsZero() || log.Timestamp.Before(olderThan)) {
			deleted++
		} else {
			newLogs = append(newLogs, log)
		}
	}

	s.logs = newLogs
	s.rebuildIndexes()

	return deleted, nil
}

func (s *MemoryAuditStore) rebuildIndexes() {
	s.logsByOrg = make(map[string][]int)
	s.logsByUser = make(map[string][]int)
	s.logsByResource = make(map[string][]int)

	for i, log := range s.logs {
		orgKey := log.OrganizationID
		s.logsByOrg[orgKey] = append(s.logsByOrg[orgKey], i)

		userKey := log.OrganizationID + ":" + log.UserID
		s.logsByUser[userKey] = append(s.logsByUser[userKey], i)

		resourceKey := log.OrganizationID + ":" + string(log.ResourceType) + ":" + log.ResourceID
		s.logsByResource[resourceKey] = append(s.logsByResource[resourceKey], i)
	}
}

func (s *MemoryAuditStore) GetRetentionSettings(ctx context.Context, orgID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if days, ok := s.retentionDays[orgID]; ok {
		return days, nil
	}
	return DefaultRetentionDays, nil
}

func (s *MemoryAuditStore) SetRetentionSettings(ctx context.Context, orgID string, days int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if days < 1 {
		days = DefaultRetentionDays
	}
	s.retentionDays[orgID] = days
	return nil
}

func (s *MemoryAuditStore) GetLogCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.logs)
}

type NoopAuditStore struct{}

func NewNoopAuditStore() *NoopAuditStore {
	return &NoopAuditStore{}
}

func (s *NoopAuditStore) Store(ctx context.Context, logs []*AuditLog) error {
	return nil
}

func (s *NoopAuditStore) Query(ctx context.Context, orgID string, filters QueryFilters) (*QueryResult, error) {
	return &QueryResult{Logs: []*AuditLog{}}, nil
}

func (s *NoopAuditStore) GetByUser(ctx context.Context, orgID, userID string, limit int) ([]*AuditLog, error) {
	return nil, nil
}

func (s *NoopAuditStore) GetByResource(ctx context.Context, orgID string, resourceType ResourceType, resourceID string) ([]*AuditLog, error) {
	return nil, nil
}

func (s *NoopAuditStore) GetByID(ctx context.Context, orgID, logID string) (*AuditLog, error) {
	return nil, ErrLogNotFound
}

func (s *NoopAuditStore) DeleteOlderThan(ctx context.Context, olderThan time.Time) (int64, error) {
	return 0, nil
}

func (s *NoopAuditStore) DeleteByOrganization(ctx context.Context, orgID string, olderThan time.Time) (int64, error) {
	return 0, nil
}

func (s *NoopAuditStore) GetRetentionSettings(ctx context.Context, orgID string) (int, error) {
	return DefaultRetentionDays, nil
}

func (s *NoopAuditStore) SetRetentionSettings(ctx context.Context, orgID string, days int) error {
	return nil
}

type CleanupService struct {
	store AuditStore
}

func NewCleanupService(store AuditStore) *CleanupService {
	return &CleanupService{store: store}
}

func (s *CleanupService) Run(ctx context.Context) (int64, error) {
	return s.store.DeleteOlderThan(ctx, time.Now().AddDate(0, 0, -DefaultRetentionDays))
}

func (s *CleanupService) RunForOrg(ctx context.Context, orgID string) (int64, error) {
	retentionDays, err := s.store.GetRetentionSettings(ctx, orgID)
	if err != nil {
		retentionDays = DefaultRetentionDays
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	return s.store.DeleteByOrganization(ctx, orgID, cutoff)
}
