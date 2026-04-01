package hubspot

import "time"

type HubSpotConfig struct {
	APIKey      string
	PortalID    string
	AccessToken string
}

type HubSpotContact struct {
	ID         string         `json:"id"`
	Email      string         `json:"email"`
	FirstName  string         `json:"firstname"`
	LastName   string         `json:"lastname"`
	Company    string         `json:"company"`
	Phone      string         `json:"phone"`
	Properties map[string]any `json:"properties,omitempty"`
	CreatedAt  time.Time      `json:"created_at,omitempty"`
	UpdatedAt  time.Time      `json:"updated_at,omitempty"`
}

type HubSpotDeal struct {
	ID          string         `json:"id"`
	Title       string         `json:"dealname"`
	Amount      float64        `json:"amount"`
	Stage       string         `json:"dealstage"`
	Probability float64        `json:"probability,omitempty"`
	CloseDate   time.Time      `json:"closedate,omitempty"`
	ContactIDs  []string       `json:"contact_ids,omitempty"`
	Pipeline    string         `json:"pipeline,omitempty"`
	Properties  map[string]any `json:"properties,omitempty"`
	CreatedAt   time.Time      `json:"created_at,omitempty"`
	UpdatedAt   time.Time      `json:"updated_at,omitempty"`
}

type HubSpotCompany struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Domain     string         `json:"domain"`
	Industry   string         `json:"industry"`
	Size       string         `json:"size,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
	CreatedAt  time.Time      `json:"created_at,omitempty"`
	UpdatedAt  time.Time      `json:"updated_at,omitempty"`
}

type HubSpotNote struct {
	ID        string    `json:"id"`
	Body      string    `json:"body"`
	ContactID string    `json:"contact_id,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	OwnerID   string    `json:"owner_id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type HubSpotOwner struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	UserID    string `json:"userId,omitempty"`
}

type HubSpotPipeline struct {
	ID           string                 `json:"id"`
	Label        string                 `json:"label"`
	DisplayOrder int                    `json:"displayOrder"`
	Stages       []HubSpotPipelineStage `json:"stages"`
}

type HubSpotPipelineStage struct {
	ID           string  `json:"id"`
	Label        string  `json:"label"`
	DisplayOrder int     `json:"displayOrder"`
	Probability  float64 `json:"probability,omitempty"`
}

type ContactListResponse struct {
	Results []HubSpotContact `json:"results"`
	Paging  *Paging          `json:"paging,omitempty"`
}

type DealListResponse struct {
	Results []HubSpotDeal `json:"results"`
	Paging  *Paging       `json:"paging,omitempty"`
}

type CompanyListResponse struct {
	Results []HubSpotCompany `json:"results"`
	Paging  *Paging          `json:"paging,omitempty"`
}

type NoteListResponse struct {
	Results []HubSpotNote `json:"results"`
	Paging  *Paging       `json:"paging,omitempty"`
}

type Paging struct {
	Next *NextPage `json:"next,omitempty"`
}

type NextPage struct {
	After string `json:"after"`
	Link  string `json:"link,omitempty"`
}

type ContactCreateRequest struct {
	Properties map[string]string `json:"properties"`
}

type ContactUpdateRequest struct {
	Properties map[string]string `json:"properties"`
}

type DealCreateRequest struct {
	Properties map[string]string `json:"properties"`
}

type NoteCreateRequest struct {
	Properties map[string]string `json:"properties"`
}

type CompanyCreateRequest struct {
	Properties map[string]string `json:"properties"`
}

type ErrorResponse struct {
	Status        string        `json:"status"`
	Message       string        `json:"message"`
	CorrelationID string        `json:"correlationId,omitempty"`
	Category      string        `json:"category,omitempty"`
	Errors        []ErrorDetail `json:"errors,omitempty"`
}

type ErrorDetail struct {
	Message string         `json:"message"`
	Code    string         `json:"code,omitempty"`
	In      string         `json:"in,omitempty"`
	Context map[string]any `json:"context,omitempty"`
}

type WebhookEvent struct {
	EventID          string `json:"eventId"`
	SubscriptionID   string `json:"subscriptionId"`
	PortalID         int64  `json:"portalId"`
	AppID            int64  `json:"appId"`
	OccurredAt       int64  `json:"occurredAt"`
	SubscriptionType string `json:"subscriptionType"`
	AttemptNumber    int    `json:"attemptNumber"`
	ObjectID         string `json:"objectId"`
	ChangeSource     string `json:"changeSource"`
	PropertyName     string `json:"propertyName,omitempty"`
	PropertyValue    string `json:"propertyValue,omitempty"`
	OldValue         string `json:"oldValue,omitempty"`
	NewValue         string `json:"newValue,omitempty"`
	Object           any    `json:"object,omitempty"`
}

type ContactCreatedEvent struct {
	WebhookEvent
	Contact HubSpotContact `json:"object"`
}

type ContactUpdatedEvent struct {
	WebhookEvent
	Contact    HubSpotContact   `json:"object"`
	Properties []PropertyChange `json:"properties,omitempty"`
}

type PropertyChange struct {
	Property  string `json:"property"`
	OldValue  string `json:"oldValue,omitempty"`
	NewValue  string `json:"newValue,omitempty"`
	Source    string `json:"source,omitempty"`
	SourceID  string `json:"sourceId,omitempty"`
	UpdatedBy string `json:"updatedBy,omitempty"`
}

type DealStageChangedEvent struct {
	WebhookEvent
	Deal     HubSpotDeal `json:"object"`
	OldStage string      `json:"oldValue,omitempty"`
	NewStage string      `json:"newValue,omitempty"`
}

type ListContactsOptions struct {
	After      string   `json:"after,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	Properties []string `json:"properties,omitempty"`
}

type ListDealsOptions struct {
	After      string   `json:"after,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	Properties []string `json:"properties,omitempty"`
}

type BatchReadRequest struct {
	Inputs     []BatchReadInput `json:"inputs"`
	Properties []string         `json:"properties,omitempty"`
	IDProperty string           `json:"idProperty,omitempty"`
}

type BatchReadInput struct {
	ID string `json:"id"`
}

type BatchReadResponse struct {
	Results   []HubSpotContact `json:"results"`
	NumErrors int              `json:"numErrors,omitempty"`
}

type Association struct {
	ToObjectType    string `json:"toObjectType"`
	ToObjectID      string `json:"toObjectId"`
	AssociationType string `json:"associationType"`
}

type AssociationResponse struct {
	Results []AssociationResult `json:"results"`
}

type AssociationResult struct {
	ToObjectType string            `json:"toObjectType"`
	ToObjectID   string            `json:"toObjectId"`
	Types        []AssociationType `json:"types"`
}

type AssociationType struct {
	Category string `json:"category"`
	TypeID   int    `json:"typeId"`
}
