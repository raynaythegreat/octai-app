package marketplace

import (
	"errors"
	"time"
)

type SkillCategory string

const (
	CategoryAutomation    SkillCategory = "automation"
	CategoryCommunication SkillCategory = "communication"
	CategoryData          SkillCategory = "data"
	CategoryIntegration   SkillCategory = "integration"
	CategoryAI            SkillCategory = "ai"
	CategoryUtility       SkillCategory = "utility"
)

func (c SkillCategory) Valid() bool {
	switch c {
	case CategoryAutomation, CategoryCommunication, CategoryData,
		CategoryIntegration, CategoryAI, CategoryUtility:
		return true
	default:
		return false
	}
}

type SkillListing struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Author      string        `json:"author"`
	Version     string        `json:"version"`
	Category    SkillCategory `json:"category"`
	Tags        []string      `json:"tags,omitempty"`
	Price       float64       `json:"price"`
	Rating      float64       `json:"rating"`
	Downloads   int64         `json:"downloads"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

type SkillReview struct {
	ID        string    `json:"id"`
	SkillID   string    `json:"skill_id"`
	UserID    string    `json:"user_id"`
	Rating    int       `json:"rating"`
	Comment   string    `json:"comment,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SkillPurchase struct {
	ID             string    `json:"id"`
	SkillID        string    `json:"skill_id"`
	OrganizationID string    `json:"organization_id"`
	PurchasedAt    time.Time `json:"purchased_at"`
}

type SkillVersion struct {
	ID          string    `json:"id"`
	SkillID     string    `json:"skill_id"`
	Version     string    `json:"version"`
	Changelog   string    `json:"changelog,omitempty"`
	DownloadURL string    `json:"download_url"`
	CreatedAt   time.Time `json:"created_at"`
}

type ListSkillsOptions struct {
	Category  SkillCategory
	Author    string
	Tags      []string
	MinPrice  *float64
	MaxPrice  *float64
	MinRating *float64
	SortBy    string
	SortDesc  bool
	Limit     int
	Offset    int
}

type PublishSkillInput struct {
	ID          string
	Name        string
	Description string
	Author      string
	Version     string
	Category    SkillCategory
	Tags        []string
	Price       float64
}

var (
	ErrSkillNotFound       = errors.New("skill not found")
	ErrReviewNotFound      = errors.New("review not found")
	ErrPurchaseNotFound    = errors.New("purchase not found")
	ErrAlreadyPurchased    = errors.New("skill already purchased")
	ErrInvalidCategory     = errors.New("invalid category")
	ErrInvalidRating       = errors.New("rating must be between 1 and 5")
	ErrInvalidPrice        = errors.New("price cannot be negative")
	ErrSkillNameRequired   = errors.New("skill name is required")
	ErrSkillAuthorRequired = errors.New("skill author is required")
	ErrVersionExists       = errors.New("version already exists")
)

// AgentSpec describes an agent member within a team template.
type AgentSpec struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Role        string `json:"role,omitempty"`
	Description string `json:"description,omitempty"`
}

// WorkflowSpec describes a workflow step within a team template.
type WorkflowSpec struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Steps       []string `json:"steps,omitempty"`
}

// TeamTemplate is a pre-built multi-agent team configuration available in the marketplace.
type TeamTemplate struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Category    SkillCategory `json:"category"`
	Agents      []AgentSpec   `json:"agents"`
	Workflows   []WorkflowSpec `json:"workflows,omitempty"`
	Author      string        `json:"author"`
	Price       float64       `json:"price"`
	Rating      float64       `json:"rating"`
	Downloads   int64         `json:"downloads"`
	Tags        []string      `json:"tags,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// BuiltinTeamTemplates returns a set of pre-built team templates.
func BuiltinTeamTemplates() []TeamTemplate {
	return []TeamTemplate{}
}

func (s *SkillListing) IsFree() bool {
	return s.Price == 0
}

func (r *SkillReview) Validate() error {
	if r.Rating < 1 || r.Rating > 5 {
		return ErrInvalidRating
	}
	return nil
}

func (i *PublishSkillInput) Validate() error {
	if i.Name == "" {
		return ErrSkillNameRequired
	}
	if i.Author == "" {
		return ErrSkillAuthorRequired
	}
	if !i.Category.Valid() {
		return ErrInvalidCategory
	}
	if i.Price < 0 {
		return ErrInvalidPrice
	}
	return nil
}
