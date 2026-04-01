package hubspot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

const (
	BaseURL           = "https://api.hubapi.com"
	DefaultRateLimit  = 100
	DefaultBurstLimit = 10
	MaxRetries        = 3
	RetryBaseDelay    = 500 * time.Millisecond
)

type Client struct {
	httpClient *http.Client
	config     HubSpotConfig
	baseURL    string
	limiter    *rate.Limiter
}

type ClientOption func(*Client)

func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

func WithRateLimit(requestsPerSecond int, burst int) ClientOption {
	return func(c *Client) {
		c.limiter = rate.NewLimiter(rate.Limit(requestsPerSecond), burst)
	}
}

func NewClient(config HubSpotConfig, opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		config:  config,
		baseURL: BaseURL,
		limiter: rate.NewLimiter(rate.Limit(DefaultRateLimit), DefaultBurstLimit),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *Client) GetContacts(ctx context.Context, opts *ListContactsOptions) ([]HubSpotContact, string, error) {
	return c.GetContactsPaginated(ctx, opts)
}

func (c *Client) GetContactsPaginated(ctx context.Context, opts *ListContactsOptions) ([]HubSpotContact, string, error) {
	if opts == nil {
		opts = &ListContactsOptions{Limit: 100}
	}
	if opts.Limit == 0 {
		opts.Limit = 100
	}

	endpoint := "/crm/v3/objects/contacts"
	params := url.Values{}
	params.Set("limit", strconv.Itoa(opts.Limit))

	if opts.After != "" {
		params.Set("after", opts.After)
	}

	if len(opts.Properties) > 0 {
		for _, prop := range opts.Properties {
			params.Add("properties", prop)
		}
	}

	resp, err := c.doRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, "", err
	}

	var result ContactListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", fmt.Errorf("failed to decode contacts: %w", err)
	}

	contacts := make([]HubSpotContact, len(result.Results))
	for i, c := range result.Results {
		contacts[i] = parseContactFromAPI(c)
	}

	var nextAfter string
	if result.Paging != nil && result.Paging.Next != nil {
		nextAfter = result.Paging.Next.After
	}

	return contacts, nextAfter, nil
}

func (c *Client) GetAllContacts(ctx context.Context, opts *ListContactsOptions) ([]HubSpotContact, error) {
	var allContacts []HubSpotContact
	after := ""

	if opts == nil {
		opts = &ListContactsOptions{Limit: 100}
	}

	for {
		opts.After = after
		contacts, nextAfter, err := c.GetContactsPaginated(ctx, opts)
		if err != nil {
			return nil, err
		}

		allContacts = append(allContacts, contacts...)

		if nextAfter == "" {
			break
		}
		after = nextAfter
	}

	return allContacts, nil
}

func (c *Client) GetContact(ctx context.Context, id string) (*HubSpotContact, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/crm/v3/objects/contacts/%s", id), nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var result struct {
		ID         string         `json:"id"`
		Properties map[string]any `json:"properties"`
		CreatedAt  string         `json:"createdAt"`
		UpdatedAt  string         `json:"updatedAt"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode contact: %w", err)
	}

	contact := parseContactFromAPI(HubSpotContact{
		ID:         result.ID,
		Properties: result.Properties,
	})

	return &contact, nil
}

func (c *Client) CreateContact(ctx context.Context, contact *HubSpotContact) (*HubSpotContact, error) {
	props := map[string]string{
		"email":     contact.Email,
		"firstname": contact.FirstName,
		"lastname":  contact.LastName,
		"company":   contact.Company,
		"phone":     contact.Phone,
	}

	for k, v := range contact.Properties {
		if s, ok := v.(string); ok && s != "" {
			props[k] = s
		}
	}

	req := ContactCreateRequest{Properties: props}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doJSONRequest(ctx, "POST", "/crm/v3/objects/contacts", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var result struct {
		ID         string         `json:"id"`
		Properties map[string]any `json:"properties"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.GetContact(ctx, result.ID)
}

func (c *Client) UpdateContact(ctx context.Context, id string, contact *HubSpotContact) error {
	props := map[string]string{}

	if contact.Email != "" {
		props["email"] = contact.Email
	}
	if contact.FirstName != "" {
		props["firstname"] = contact.FirstName
	}
	if contact.LastName != "" {
		props["lastname"] = contact.LastName
	}
	if contact.Company != "" {
		props["company"] = contact.Company
	}
	if contact.Phone != "" {
		props["phone"] = contact.Phone
	}

	for k, v := range contact.Properties {
		if s, ok := v.(string); ok && s != "" {
			props[k] = s
		}
	}

	req := ContactUpdateRequest{Properties: props}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doJSONRequest(ctx, "PATCH", fmt.Sprintf("/crm/v3/objects/contacts/%s", id), bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.checkErrorResponse(resp)
}

func (c *Client) DeleteContact(ctx context.Context, id string) error {
	resp, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("/crm/v3/objects/contacts/%s", id), nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.checkErrorResponse(resp)
}

func (c *Client) GetDeals(ctx context.Context, opts *ListDealsOptions) ([]HubSpotDeal, string, error) {
	return c.GetDealsPaginated(ctx, opts)
}

func (c *Client) GetDealsPaginated(ctx context.Context, opts *ListDealsOptions) ([]HubSpotDeal, string, error) {
	if opts == nil {
		opts = &ListDealsOptions{Limit: 100}
	}
	if opts.Limit == 0 {
		opts.Limit = 100
	}

	endpoint := "/crm/v3/objects/deals"
	params := url.Values{}
	params.Set("limit", strconv.Itoa(opts.Limit))

	if opts.After != "" {
		params.Set("after", opts.After)
	}

	defaultProps := []string{"dealname", "amount", "dealstage", "pipeline", "closedate", "probability", "createdate", "hs_lastmodifieddate"}
	for _, prop := range defaultProps {
		params.Add("properties", prop)
	}

	if len(opts.Properties) > 0 {
		for _, prop := range opts.Properties {
			params.Add("properties", prop)
		}
	}

	resp, err := c.doRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, "", err
	}

	var result DealListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", fmt.Errorf("failed to decode deals: %w", err)
	}

	deals := make([]HubSpotDeal, len(result.Results))
	for i, d := range result.Results {
		deals[i] = parseDealFromAPI(d)
	}

	var nextAfter string
	if result.Paging != nil && result.Paging.Next != nil {
		nextAfter = result.Paging.Next.After
	}

	return deals, nextAfter, nil
}

func (c *Client) GetAllDeals(ctx context.Context, opts *ListDealsOptions) ([]HubSpotDeal, error) {
	var allDeals []HubSpotDeal
	after := ""

	if opts == nil {
		opts = &ListDealsOptions{Limit: 100}
	}

	for {
		opts.After = after
		deals, nextAfter, err := c.GetDealsPaginated(ctx, opts)
		if err != nil {
			return nil, err
		}

		allDeals = append(allDeals, deals...)

		if nextAfter == "" {
			break
		}
		after = nextAfter
	}

	return allDeals, nil
}

func (c *Client) GetDeal(ctx context.Context, id string) (*HubSpotDeal, error) {
	params := url.Values{}
	props := []string{"dealname", "amount", "dealstage", "pipeline", "closedate", "probability", "createdate", "hs_lastmodifieddate"}
	for _, prop := range props {
		params.Add("properties", prop)
	}

	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/crm/v3/objects/deals/%s", id), params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var result struct {
		ID         string         `json:"id"`
		Properties map[string]any `json:"properties"`
		CreatedAt  string         `json:"createdAt"`
		UpdatedAt  string         `json:"updatedAt"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode deal: %w", err)
	}

	deal := parseDealFromAPI(HubSpotDeal{
		ID:         result.ID,
		Properties: result.Properties,
	})

	return &deal, nil
}

func (c *Client) CreateDeal(ctx context.Context, deal *HubSpotDeal) (*HubSpotDeal, error) {
	props := map[string]string{
		"dealname": deal.Title,
	}

	if deal.Amount > 0 {
		props["amount"] = fmt.Sprintf("%.2f", deal.Amount)
	}
	if deal.Stage != "" {
		props["dealstage"] = deal.Stage
	}
	if deal.Pipeline != "" {
		props["pipeline"] = deal.Pipeline
	}
	if !deal.CloseDate.IsZero() {
		props["closedate"] = deal.CloseDate.Format(time.RFC3339)
	}

	req := DealCreateRequest{Properties: props}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doJSONRequest(ctx, "POST", "/crm/v3/objects/deals", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var result struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.GetDeal(ctx, result.ID)
}

func (c *Client) UpdateDeal(ctx context.Context, id string, deal *HubSpotDeal) error {
	props := map[string]string{}

	if deal.Title != "" {
		props["dealname"] = deal.Title
	}
	if deal.Amount > 0 {
		props["amount"] = fmt.Sprintf("%.2f", deal.Amount)
	}
	if deal.Stage != "" {
		props["dealstage"] = deal.Stage
	}
	if deal.Pipeline != "" {
		props["pipeline"] = deal.Pipeline
	}
	if !deal.CloseDate.IsZero() {
		props["closedate"] = deal.CloseDate.Format(time.RFC3339)
	}

	req := DealCreateRequest{Properties: props}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doJSONRequest(ctx, "PATCH", fmt.Sprintf("/crm/v3/objects/deals/%s", id), bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.checkErrorResponse(resp)
}

func (c *Client) GetCompanies(ctx context.Context) ([]HubSpotCompany, error) {
	return c.GetAllCompanies(ctx)
}

func (c *Client) GetAllCompanies(ctx context.Context) ([]HubSpotCompany, error) {
	var allCompanies []HubSpotCompany
	after := ""

	params := url.Values{}
	props := []string{"name", "domain", "industry", "size", "createdate", "hs_lastmodifieddate"}
	for _, prop := range props {
		params.Add("properties", prop)
	}

	for {
		if after != "" {
			params.Set("after", after)
		}
		params.Set("limit", "100")

		resp, err := c.doRequest(ctx, "GET", "/crm/v3/objects/companies", params, nil)
		if err != nil {
			return nil, err
		}

		if err := c.checkErrorResponse(resp); err != nil {
			resp.Body.Close()
			return nil, err
		}

		var result CompanyListResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode companies: %w", err)
		}
		resp.Body.Close()

		for _, c := range result.Results {
			allCompanies = append(allCompanies, parseCompanyFromAPI(c))
		}

		if result.Paging == nil || result.Paging.Next == nil {
			break
		}
		after = result.Paging.Next.After
	}

	return allCompanies, nil
}

func (c *Client) GetCompany(ctx context.Context, id string) (*HubSpotCompany, error) {
	params := url.Values{}
	props := []string{"name", "domain", "industry", "size", "createdate", "hs_lastmodifieddate"}
	for _, prop := range props {
		params.Add("properties", prop)
	}

	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/crm/v3/objects/companies/%s", id), params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var result struct {
		ID         string         `json:"id"`
		Properties map[string]any `json:"properties"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode company: %w", err)
	}

	company := parseCompanyFromAPI(HubSpotCompany{
		ID:         result.ID,
		Properties: result.Properties,
	})

	return &company, nil
}

func (c *Client) CreateCompany(ctx context.Context, company *HubSpotCompany) (*HubSpotCompany, error) {
	props := map[string]string{
		"name": company.Name,
	}

	if company.Domain != "" {
		props["domain"] = company.Domain
	}
	if company.Industry != "" {
		props["industry"] = company.Industry
	}
	if company.Size != "" {
		props["size"] = company.Size
	}

	req := CompanyCreateRequest{Properties: props}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doJSONRequest(ctx, "POST", "/crm/v3/objects/companies", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var result struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.GetCompany(ctx, result.ID)
}

func (c *Client) CreateNote(ctx context.Context, note *HubSpotNote) (*HubSpotNote, error) {
	props := map[string]string{
		"hs_note_body": note.Body,
	}

	if note.ContactID != "" {
		props["hs_timestamp"] = fmt.Sprintf("%d", time.Now().UnixMilli())
	}

	req := NoteCreateRequest{Properties: props}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doJSONRequest(ctx, "POST", "/crm/v3/objects/notes", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var result struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if note.ContactID != "" {
		if err := c.AssociateNoteToContact(ctx, result.ID, note.ContactID); err != nil {
			return nil, fmt.Errorf("failed to associate note to contact: %w", err)
		}
	}

	return &HubSpotNote{
		ID:        result.ID,
		Body:      note.Body,
		ContactID: note.ContactID,
		Timestamp: time.Now(),
	}, nil
}

func (c *Client) AssociateNoteToContact(ctx context.Context, noteID, contactID string) error {
	endpoint := fmt.Sprintf("/crm/v3/objects/notes/%s/associations/contact/%s/contact_to_note", noteID, contactID)

	resp, err := c.doJSONRequest(ctx, "PUT", endpoint, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.checkErrorResponse(resp)
}

func (c *Client) GetPipelines(ctx context.Context) ([]HubSpotPipeline, error) {
	resp, err := c.doRequest(ctx, "GET", "/crm/v3/pipelines/deals", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var result struct {
		Results []HubSpotPipeline `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode pipelines: %w", err)
	}

	return result.Results, nil
}

func (c *Client) GetOwners(ctx context.Context) ([]HubSpotOwner, error) {
	resp, err := c.doRequest(ctx, "GET", "/crm/v3/owners", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var result struct {
		Results []HubSpotOwner `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode owners: %w", err)
	}

	return result.Results, nil
}

func (c *Client) SearchContacts(ctx context.Context, query string) ([]HubSpotContact, error) {
	searchReq := map[string]any{
		"query":      query,
		"limit":      100,
		"properties": []string{"email", "firstname", "lastname", "company", "phone"},
	}

	body, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doJSONRequest(ctx, "POST", "/crm/v3/objects/contacts/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var result ContactListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode contacts: %w", err)
	}

	contacts := make([]HubSpotContact, len(result.Results))
	for i, c := range result.Results {
		contacts[i] = parseContactFromAPI(c)
	}

	return contacts, nil
}

func (c *Client) TestConnection(ctx context.Context) error {
	_, err := c.GetOwners(ctx)
	return err
}

func (c *Client) doRequest(ctx context.Context, method, endpoint string, params url.Values, body io.Reader) (*http.Response, error) {
	return c.doRequestWithRetry(ctx, method, endpoint, params, body, false)
}

func (c *Client) doJSONRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Response, error) {
	return c.doRequestWithRetry(ctx, method, endpoint, nil, body, true)
}

func (c *Client) doRequestWithRetry(ctx context.Context, method, endpoint string, params url.Values, body io.Reader, isJSON bool) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter error: %w", err)
		}

		endpointURL := c.baseURL + endpoint
		if params != nil && len(params) > 0 {
			endpointURL = endpointURL + "?" + params.Encode()
		}

		var bodyReader io.Reader
		if body != nil {
			bodyBytes, err := io.ReadAll(body)
			if err != nil {
				return nil, fmt.Errorf("failed to read body: %w", err)
			}
			bodyReader = bytes.NewReader(bodyBytes)
			body = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, endpointURL, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		c.setAuth(req)
		req.Header.Set("Accept", "application/json")
		if isJSON {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if shouldRetry(err) && attempt < MaxRetries {
				delay := RetryBaseDelay * time.Duration(1<<attempt)
				time.Sleep(delay)
				continue
			}
			return nil, err
		}

		if resp.StatusCode == 429 || (resp.StatusCode >= 500 && resp.StatusCode < 600) {
			resp.Body.Close()
			lastErr = fmt.Errorf("server returned status %d", resp.StatusCode)
			if attempt < MaxRetries {
				delay := RetryBaseDelay * time.Duration(1<<attempt)
				if resp.StatusCode == 429 {
					retryAfter := resp.Header.Get("Retry-After")
					if retryAfter != "" {
						if seconds, err := strconv.Atoi(retryAfter); err == nil {
							delay = time.Duration(seconds) * time.Second
						}
					}
				}
				time.Sleep(delay)
				continue
			}
			return nil, lastErr
		}

		return resp, nil
	}

	return nil, lastErr
}

func shouldRetry(err error) bool {
	if strings.Contains(err.Error(), "connection reset") ||
		strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "temporary") {
		return true
	}
	return false
}

func (c *Client) setAuth(req *http.Request) {
	if c.config.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.AccessToken)
	} else if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}
}

func (c *Client) checkErrorResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("request failed with status %d: failed to read error body", resp.StatusCode)
	}

	var hsErr ErrorResponse
	if err := json.Unmarshal(body, &hsErr); err == nil {
		if hsErr.Message != "" {
			return fmt.Errorf("hubspot error (status %d): %s", resp.StatusCode, hsErr.Message)
		}
		if len(hsErr.Errors) > 0 {
			var errMsgs []string
			for _, e := range hsErr.Errors {
				errMsgs = append(errMsgs, e.Message)
			}
			return fmt.Errorf("hubspot error (status %d): %s", resp.StatusCode, strings.Join(errMsgs, "; "))
		}
	}

	return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
}

func parseContactFromAPI(c HubSpotContact) HubSpotContact {
	props := c.Properties
	if props == nil {
		return c
	}

	contact := HubSpotContact{
		ID:         c.ID,
		Properties: props,
	}

	if v, ok := props["email"]; ok {
		contact.Email = toString(v)
	}
	if v, ok := props["firstname"]; ok {
		contact.FirstName = toString(v)
	}
	if v, ok := props["lastname"]; ok {
		contact.LastName = toString(v)
	}
	if v, ok := props["company"]; ok {
		contact.Company = toString(v)
	}
	if v, ok := props["phone"]; ok {
		contact.Phone = toString(v)
	}
	if v, ok := props["createdate"]; ok {
		contact.CreatedAt = toTime(v)
	}
	if v, ok := props["hs_lastmodifieddate"]; ok {
		contact.UpdatedAt = toTime(v)
	}

	return contact
}

func parseDealFromAPI(d HubSpotDeal) HubSpotDeal {
	props := d.Properties
	if props == nil {
		return d
	}

	deal := HubSpotDeal{
		ID:         d.ID,
		Properties: props,
	}

	if v, ok := props["dealname"]; ok {
		deal.Title = toString(v)
	}
	if v, ok := props["amount"]; ok {
		deal.Amount = toFloat64(v)
	}
	if v, ok := props["dealstage"]; ok {
		deal.Stage = toString(v)
	}
	if v, ok := props["pipeline"]; ok {
		deal.Pipeline = toString(v)
	}
	if v, ok := props["probability"]; ok {
		deal.Probability = toFloat64(v)
	}
	if v, ok := props["closedate"]; ok {
		deal.CloseDate = toTime(v)
	}
	if v, ok := props["createdate"]; ok {
		deal.CreatedAt = toTime(v)
	}
	if v, ok := props["hs_lastmodifieddate"]; ok {
		deal.UpdatedAt = toTime(v)
	}

	return deal
}

func parseCompanyFromAPI(c HubSpotCompany) HubSpotCompany {
	props := c.Properties
	if props == nil {
		return c
	}

	company := HubSpotCompany{
		ID:         c.ID,
		Properties: props,
	}

	if v, ok := props["name"]; ok {
		company.Name = toString(v)
	}
	if v, ok := props["domain"]; ok {
		company.Domain = toString(v)
	}
	if v, ok := props["industry"]; ok {
		company.Industry = toString(v)
	}
	if v, ok := props["size"]; ok {
		company.Size = toString(v)
	}
	if v, ok := props["createdate"]; ok {
		company.CreatedAt = toTime(v)
	}
	if v, ok := props["hs_lastmodifieddate"]; ok {
		company.UpdatedAt = toTime(v)
	}

	return company
}

func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%.0f", val)
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func toFloat64(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case string:
		var f float64
		fmt.Sscanf(val, "%f", &f)
		return f
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}

func toTime(v any) time.Time {
	switch val := v.(type) {
	case string:
		if strings.Contains(val, "T") {
			t, err := time.Parse(time.RFC3339, val)
			if err == nil {
				return t
			}
		}
		if ms, err := strconv.ParseInt(val, 10, 64); err == nil {
			return time.UnixMilli(ms)
		}
	case float64:
		return time.UnixMilli(int64(val))
	case int64:
		return time.UnixMilli(val)
	}
	return time.Time{}
}
