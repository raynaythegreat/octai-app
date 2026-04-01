package hubspot

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type ContactRepository interface {
	CreateContact(ctx context.Context, orgID string, contact *InternalContact) error
	UpdateContact(ctx context.Context, orgID string, externalID string, contact *InternalContact) error
	GetContactByExternalID(ctx context.Context, orgID string, externalID string) (*InternalContact, error)
	DeleteContact(ctx context.Context, orgID string, externalID string) error
}

type DealRepository interface {
	CreateDeal(ctx context.Context, orgID string, deal *InternalDeal) error
	UpdateDeal(ctx context.Context, orgID string, externalID string, deal *InternalDeal) error
	GetDealByExternalID(ctx context.Context, orgID string, externalID string) (*InternalDeal, error)
	DeleteDeal(ctx context.Context, orgID string, externalID string) error
}

type InternalContact struct {
	ID         string         `json:"id"`
	OrgID      string         `json:"org_id"`
	ExternalID string         `json:"external_id"`
	Source     string         `json:"source"`
	Email      string         `json:"email"`
	FirstName  string         `json:"first_name"`
	LastName   string         `json:"last_name"`
	FullName   string         `json:"full_name"`
	Company    string         `json:"company"`
	Phone      string         `json:"phone"`
	Properties map[string]any `json:"properties,omitempty"`
	LastSyncAt time.Time      `json:"last_sync_at"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type InternalDeal struct {
	ID          string         `json:"id"`
	OrgID       string         `json:"org_id"`
	ExternalID  string         `json:"external_id"`
	Source      string         `json:"source"`
	Title       string         `json:"title"`
	Amount      float64        `json:"amount"`
	Stage       string         `json:"stage"`
	Probability float64        `json:"probability"`
	CloseDate   time.Time      `json:"close_date,omitempty"`
	Pipeline    string         `json:"pipeline,omitempty"`
	ContactIDs  []string       `json:"contact_ids,omitempty"`
	Properties  map[string]any `json:"properties,omitempty"`
	LastSyncAt  time.Time      `json:"last_sync_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type SyncConfig struct {
	BatchSize    int           `json:"batch_size"`
	SyncInterval time.Duration `json:"sync_interval"`
	FullSync     bool          `json:"full_sync"`
	Properties   []string      `json:"properties,omitempty"`
}

type SyncResult struct {
	ContactsProcessed int           `json:"contacts_processed"`
	ContactsCreated   int           `json:"contacts_created"`
	ContactsUpdated   int           `json:"contacts_updated"`
	ContactsSkipped   int           `json:"contacts_skipped"`
	ContactsDeleted   int           `json:"contacts_deleted"`
	DealsProcessed    int           `json:"deals_processed"`
	DealsCreated      int           `json:"deals_created"`
	DealsUpdated      int           `json:"deals_updated"`
	DealsSkipped      int           `json:"deals_skipped"`
	DealsDeleted      int           `json:"deals_deleted"`
	Errors            []SyncError   `json:"errors,omitempty"`
	Duration          time.Duration `json:"duration"`
}

type SyncError struct {
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Error        string `json:"error"`
}

type SyncService struct {
	client          *Client
	contactRepo     ContactRepository
	dealRepo        DealRepository
	config          SyncConfig
	mu              sync.RWMutex
	lastContactSync time.Time
	lastDealSync    time.Time
	running         bool
}

func NewSyncService(client *Client, contactRepo ContactRepository, dealRepo DealRepository, config SyncConfig) *SyncService {
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.SyncInterval == 0 {
		config.SyncInterval = 15 * time.Minute
	}

	return &SyncService{
		client:      client,
		contactRepo: contactRepo,
		dealRepo:    dealRepo,
		config:      config,
	}
}

func (s *SyncService) SyncContacts(ctx context.Context, orgID string) error {
	result, err := s.SyncContactsFull(ctx, orgID)
	if err != nil {
		return err
	}

	if len(result.Errors) > 0 {
		return fmt.Errorf("sync completed with %d errors", len(result.Errors))
	}

	return nil
}

func (s *SyncService) SyncContactsFull(ctx context.Context, orgID string) (*SyncResult, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil, fmt.Errorf("sync already in progress")
	}
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	start := time.Now()
	result := &SyncResult{}

	opts := &ListContactsOptions{
		Limit:      s.config.BatchSize,
		Properties: s.config.Properties,
	}

	if !s.config.FullSync && !s.lastContactSync.IsZero() {
		opts.Properties = append(opts.Properties, "hs_lastmodifieddate")
	}

	after := ""
	for {
		opts.After = after
		contacts, nextAfter, err := s.client.GetContactsPaginated(ctx, opts)
		if err != nil {
			result.Errors = append(result.Errors, SyncError{
				ResourceType: "contact",
				Error:        err.Error(),
			})
			break
		}

		for _, contact := range contacts {
			result.ContactsProcessed++

			if !s.config.FullSync && !s.lastContactSync.IsZero() {
				if contact.UpdatedAt.Before(s.lastContactSync) {
					result.ContactsSkipped++
					continue
				}
			}

			internal := s.MapContact(contact)
			internal.OrgID = orgID

			existing, err := s.contactRepo.GetContactByExternalID(ctx, orgID, contact.ID)
			if err != nil {
				if createErr := s.contactRepo.CreateContact(ctx, orgID, internal); createErr != nil {
					result.Errors = append(result.Errors, SyncError{
						ResourceType: "contact",
						ResourceID:   contact.ID,
						Error:        createErr.Error(),
					})
					continue
				}
				result.ContactsCreated++
			} else {
				internal.ID = existing.ID
				internal.CreatedAt = existing.CreatedAt
				if updateErr := s.contactRepo.UpdateContact(ctx, orgID, contact.ID, internal); updateErr != nil {
					result.Errors = append(result.Errors, SyncError{
						ResourceType: "contact",
						ResourceID:   contact.ID,
						Error:        updateErr.Error(),
					})
					continue
				}
				result.ContactsUpdated++
			}
		}

		if nextAfter == "" {
			break
		}
		after = nextAfter

		select {
		case <-ctx.Done():
			result.Duration = time.Since(start)
			return result, ctx.Err()
		default:
		}
	}

	s.mu.Lock()
	s.lastContactSync = time.Now()
	s.mu.Unlock()

	result.Duration = time.Since(start)
	return result, nil
}

func (s *SyncService) SyncDeals(ctx context.Context, orgID string) error {
	result, err := s.SyncDealsFull(ctx, orgID)
	if err != nil {
		return err
	}

	if len(result.Errors) > 0 {
		return fmt.Errorf("sync completed with %d errors", len(result.Errors))
	}

	return nil
}

func (s *SyncService) SyncDealsFull(ctx context.Context, orgID string) (*SyncResult, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil, fmt.Errorf("sync already in progress")
	}
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	start := time.Now()
	result := &SyncResult{}

	opts := &ListDealsOptions{
		Limit:      s.config.BatchSize,
		Properties: s.config.Properties,
	}

	after := ""
	for {
		opts.After = after
		deals, nextAfter, err := s.client.GetDealsPaginated(ctx, opts)
		if err != nil {
			result.Errors = append(result.Errors, SyncError{
				ResourceType: "deal",
				Error:        err.Error(),
			})
			break
		}

		for _, deal := range deals {
			result.DealsProcessed++

			if !s.config.FullSync && !s.lastDealSync.IsZero() {
				if deal.UpdatedAt.Before(s.lastDealSync) {
					result.DealsSkipped++
					continue
				}
			}

			internal := s.MapDeal(deal)
			internal.OrgID = orgID

			existing, err := s.dealRepo.GetDealByExternalID(ctx, orgID, deal.ID)
			if err != nil {
				if createErr := s.dealRepo.CreateDeal(ctx, orgID, internal); createErr != nil {
					result.Errors = append(result.Errors, SyncError{
						ResourceType: "deal",
						ResourceID:   deal.ID,
						Error:        createErr.Error(),
					})
					continue
				}
				result.DealsCreated++
			} else {
				internal.ID = existing.ID
				internal.CreatedAt = existing.CreatedAt
				if updateErr := s.dealRepo.UpdateDeal(ctx, orgID, deal.ID, internal); updateErr != nil {
					result.Errors = append(result.Errors, SyncError{
						ResourceType: "deal",
						ResourceID:   deal.ID,
						Error:        updateErr.Error(),
					})
					continue
				}
				result.DealsUpdated++
			}
		}

		if nextAfter == "" {
			break
		}
		after = nextAfter

		select {
		case <-ctx.Done():
			result.Duration = time.Since(start)
			return result, ctx.Err()
		default:
		}
	}

	s.mu.Lock()
	s.lastDealSync = time.Now()
	s.mu.Unlock()

	result.Duration = time.Since(start)
	return result, nil
}

func (s *SyncService) MapContact(contact HubSpotContact) *InternalContact {
	return &InternalContact{
		ExternalID: contact.ID,
		Source:     "hubspot",
		Email:      contact.Email,
		FirstName:  contact.FirstName,
		LastName:   contact.LastName,
		FullName:   fmt.Sprintf("%s %s", contact.FirstName, contact.LastName),
		Company:    contact.Company,
		Phone:      contact.Phone,
		Properties: contact.Properties,
		LastSyncAt: time.Now(),
		CreatedAt:  contact.CreatedAt,
		UpdatedAt:  contact.UpdatedAt,
	}
}

func (s *SyncService) MapDeal(deal HubSpotDeal) *InternalDeal {
	return &InternalDeal{
		ExternalID:  deal.ID,
		Source:      "hubspot",
		Title:       deal.Title,
		Amount:      deal.Amount,
		Stage:       deal.Stage,
		Probability: deal.Probability,
		CloseDate:   deal.CloseDate,
		Pipeline:    deal.Pipeline,
		ContactIDs:  deal.ContactIDs,
		Properties:  deal.Properties,
		LastSyncAt:  time.Now(),
		CreatedAt:   deal.CreatedAt,
		UpdatedAt:   deal.UpdatedAt,
	}
}

func (s *SyncService) SyncContactByID(ctx context.Context, orgID, contactID string) error {
	contact, err := s.client.GetContact(ctx, contactID)
	if err != nil {
		return fmt.Errorf("failed to fetch contact from HubSpot: %w", err)
	}

	internal := s.MapContact(*contact)
	internal.OrgID = orgID

	existing, err := s.contactRepo.GetContactByExternalID(ctx, orgID, contactID)
	if err != nil {
		return s.contactRepo.CreateContact(ctx, orgID, internal)
	}

	internal.ID = existing.ID
	internal.CreatedAt = existing.CreatedAt
	return s.contactRepo.UpdateContact(ctx, orgID, contactID, internal)
}

func (s *SyncService) SyncDealByID(ctx context.Context, orgID, dealID string) error {
	deal, err := s.client.GetDeal(ctx, dealID)
	if err != nil {
		return fmt.Errorf("failed to fetch deal from HubSpot: %w", err)
	}

	internal := s.MapDeal(*deal)
	internal.OrgID = orgID

	existing, err := s.dealRepo.GetDealByExternalID(ctx, orgID, dealID)
	if err != nil {
		return s.dealRepo.CreateDeal(ctx, orgID, internal)
	}

	internal.ID = existing.ID
	internal.CreatedAt = existing.CreatedAt
	return s.dealRepo.UpdateDeal(ctx, orgID, dealID, internal)
}

func (s *SyncService) GetLastContactSync() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastContactSync
}

func (s *SyncService) GetLastDealSync() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastDealSync
}

func (s *SyncService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *SyncService) StartPeriodicSync(ctx context.Context, orgID string) chan struct{} {
	stopCh := make(chan struct{})

	go func() {
		ticker := time.NewTicker(s.config.SyncInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := s.SyncContacts(ctx, orgID); err != nil {
					fmt.Printf("contact sync error: %v\n", err)
				}
				if err := s.SyncDeals(ctx, orgID); err != nil {
					fmt.Printf("deal sync error: %v\n", err)
				}
			case <-stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return stopCh
}
