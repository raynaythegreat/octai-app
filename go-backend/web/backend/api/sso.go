package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/auth/oidc"
	"github.com/raynaythegreat/octai-app/pkg/auth/saml"
	"github.com/raynaythegreat/octai-app/pkg/logger"
)

const (
	ssoFlowTTL      = 10 * time.Minute
	ssoFlowGCAge    = 30 * time.Minute
	ssoProviderSAML = "saml"
	ssoProviderOIDC = "oidc"
	ssoFlowPending  = "pending"
	ssoFlowSuccess  = "success"
	ssoFlowError    = "error"
	ssoFlowExpired  = "expired"
)

type ssoFlow struct {
	ID           string
	Provider     string
	ProviderType string
	Status       string
	Error        string
	State        string
	Nonce        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ExpiresAt    time.Time
	User         *ssoUser
}

type ssoUser struct {
	ID         string                 `json:"id"`
	Email      string                 `json:"email"`
	FirstName  string                 `json:"first_name,omitempty"`
	LastName   string                 `json:"last_name,omitempty"`
	Provider   string                 `json:"provider"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

type ssoFlowResponse struct {
	FlowID    string   `json:"flow_id"`
	Provider  string   `json:"provider"`
	Status    string   `json:"status"`
	AuthURL   string   `json:"auth_url,omitempty"`
	User      *ssoUser `json:"user,omitempty"`
	Error     string   `json:"error,omitempty"`
	ExpiresAt string   `json:"expires_at,omitempty"`
}

type samlLoginResponse struct {
	FlowID    string `json:"flow_id"`
	AuthURL   string `json:"auth_url"`
	ExpiresAt string `json:"expires_at"`
}

type oidcLoginResponse struct {
	FlowID    string `json:"flow_id"`
	AuthURL   string `json:"auth_url"`
	ExpiresAt string `json:"expires_at"`
}

type ssoCallbackResponse struct {
	Status string   `json:"status"`
	User   *ssoUser `json:"user,omitempty"`
	Error  string   `json:"error,omitempty"`
}

var (
	ssoSAMLService  *saml.SAMLService
	ssoOIDCServices map[string]*oidc.OIDCService
)

func initSSOServices() error {
	ssoSAMLService = saml.NewSAMLService()

	if config, err := saml.LoadFromEnvironment("SAML"); err == nil {
		if err := ssoSAMLService.Init(config); err != nil {
			logger.WarnCF("sso", "Failed to initialize SAML service", map[string]any{"error": err.Error()})
		}
	}

	ssoOIDCServices = make(map[string]*oidc.OIDCService)

	if clientID := os.Getenv("GOOGLE_OIDC_CLIENT_ID"); clientID != "" {
		clientSecret := os.Getenv("GOOGLE_OIDC_CLIENT_SECRET")
		redirectURI := os.Getenv("GOOGLE_OIDC_REDIRECT_URI")
		if redirectURI == "" {
			redirectURI = "/api/v2/oidc/callback"
		}

		googleService := oidc.NewOIDCService(
			oidc.GoogleOIDCConfig(clientID, clientSecret, redirectURI),
			"google",
		)
		if err := googleService.Init(nil); err != nil {
			logger.WarnCF("sso", "Failed to initialize Google OIDC service", map[string]any{"error": err.Error()})
		} else {
			ssoOIDCServices["google"] = googleService
		}
	}

	return nil
}

func (h *Handler) registerSSORoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v2/saml/login", h.handleSAMLLogin)
	mux.HandleFunc("POST /api/v2/saml/callback", h.handleSAMLCallback)
	mux.HandleFunc("GET /api/v2/saml/metadata", h.handleSAMLMetadata)
	mux.HandleFunc("GET /api/v2/oidc/login", h.handleOIDCLogin)
	mux.HandleFunc("GET /api/v2/oidc/callback", h.handleOIDCCallback)
}

func (h *Handler) handleSAMLLogin(w http.ResponseWriter, r *http.Request) {
	if ssoSAMLService == nil || ssoSAMLService.GetProvider() == nil {
		http.Error(w, "SAML service not configured", http.StatusServiceUnavailable)
		return
	}

	state, err := generateSSOState()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to generate state: %v", err), http.StatusInternalServerError)
		return
	}

	now := time.Now()
	flow := &ssoFlow{
		ID:           newSSOFlowID(),
		Provider:     ssoSAMLService.GetProvider().GetName(),
		ProviderType: ssoProviderSAML,
		Status:       ssoFlowPending,
		State:        state,
		CreatedAt:    now,
		UpdatedAt:    now,
		ExpiresAt:    now.Add(ssoFlowTTL),
	}
	h.storeSSOFlow(flow)

	authURL, err := ssoSAMLService.GenerateAuthnRequest(state)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to generate auth request: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(samlLoginResponse{
		FlowID:    flow.ID,
		AuthURL:   ssoSAMLService.GetProvider().GetSSOURL() + "?SAMLRequest=" + authURL,
		ExpiresAt: flow.ExpiresAt.Format(time.RFC3339),
	})
}

func (h *Handler) handleSAMLCallback(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req struct {
		SAMLResponse string `json:"SAMLResponse"`
		RelayState   string `json:"RelayState"`
	}

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		values, err := parseFormBody(string(body))
		if err != nil {
			http.Error(w, "failed to parse form body", http.StatusBadRequest)
			return
		}
		if v := values["SAMLResponse"]; len(v) > 0 {
			req.SAMLResponse = v[0]
		}
		if v := values["RelayState"]; len(v) > 0 {
			req.RelayState = v[0]
		}
	} else {
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
			return
		}
	}

	if req.SAMLResponse == "" {
		http.Error(w, "missing SAMLResponse", http.StatusBadRequest)
		return
	}

	user, err := ssoSAMLService.HandleLogin(r.Context(), req.SAMLResponse)
	if err != nil {
		logger.ErrorCF("sso", "SAML login failed", map[string]any{"error": err.Error()})
		http.Error(w, fmt.Sprintf("SAML login failed: %v", err), http.StatusUnauthorized)
		return
	}

	ssoUser := &ssoUser{
		ID:         user.ID,
		Email:      user.Email,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		Provider:   user.Provider,
		Attributes: user.Attributes,
	}

	if req.RelayState != "" {
		if flow, ok := h.getSSOFlow(req.RelayState); ok {
			flow.Status = ssoFlowSuccess
			flow.User = ssoUser
			h.storeSSOFlow(flow)
		}
	}

	logger.InfoCF("sso", "SAML authentication successful", map[string]any{
		"email":    user.Email,
		"provider": user.Provider,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ssoCallbackResponse{
		Status: "success",
		User:   ssoUser,
	})
}

func (h *Handler) handleSAMLMetadata(w http.ResponseWriter, r *http.Request) {
	if ssoSAMLService == nil {
		http.Error(w, "SAML service not configured", http.StatusServiceUnavailable)
		return
	}

	metadata, err := ssoSAMLService.GetMetadata(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get metadata: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Write(metadata)
}

func (h *Handler) handleOIDCLogin(w http.ResponseWriter, r *http.Request) {
	provider := strings.TrimSpace(r.URL.Query().Get("provider"))
	if provider == "" {
		provider = "google"
	}

	oidcService, ok := ssoOIDCServices[provider]
	if !ok {
		http.Error(w, fmt.Sprintf("OIDC provider %q not configured", provider), http.StatusBadRequest)
		return
	}

	state, err := oidc.GenerateState()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to generate state: %v", err), http.StatusInternalServerError)
		return
	}

	nonce, err := oidc.GenerateNonce()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to generate nonce: %v", err), http.StatusInternalServerError)
		return
	}

	now := time.Now()
	flow := &ssoFlow{
		ID:           newSSOFlowID(),
		Provider:     provider,
		ProviderType: ssoProviderOIDC,
		Status:       ssoFlowPending,
		State:        state,
		Nonce:        nonce,
		CreatedAt:    now,
		UpdatedAt:    now,
		ExpiresAt:    now.Add(ssoFlowTTL),
	}
	h.storeSSOFlow(flow)

	authURL, err := oidcService.GetAuthURL(r.Context(), state)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to generate auth URL: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(oidcLoginResponse{
		FlowID:    flow.ID,
		AuthURL:   authURL,
		ExpiresAt: flow.ExpiresAt.Format(time.RFC3339),
	})
}

func (h *Handler) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))

	if code == "" {
		renderSSOCallbackPage(w, "", ssoFlowError, "Missing authorization code", "missing_code")
		return
	}

	flow, ok := h.getSSOFlow(state)
	if !ok {
		renderSSOCallbackPage(w, "", ssoFlowError, "Flow not found", "flow_not_found")
		return
	}

	if flow.Status != ssoFlowPending {
		renderSSOCallbackPage(w, flow.ID, flow.Status, "Flow already completed", flow.Error)
		return
	}

	oidcService, ok := ssoOIDCServices[flow.Provider]
	if !ok {
		h.setSSOFlowError(flow.ID, fmt.Sprintf("provider %q not configured", flow.Provider))
		renderSSOCallbackPage(w, flow.ID, ssoFlowError, "Provider not configured", "")
		return
	}

	user, _, err := oidcService.HandleCallback(r.Context(), code, state)
	if err != nil {
		h.setSSOFlowError(flow.ID, err.Error())
		renderSSOCallbackPage(w, flow.ID, ssoFlowError, "Authentication failed", err.Error())
		return
	}

	ssoUser := &ssoUser{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Provider:  user.Provider,
	}

	flow.Status = ssoFlowSuccess
	flow.User = ssoUser
	h.storeSSOFlow(flow)

	logger.InfoCF("sso", "OIDC authentication successful", map[string]any{
		"email":    user.Email,
		"provider": user.Provider,
	})

	renderSSOCallbackPage(w, flow.ID, ssoFlowSuccess, "Authentication successful", "")
}

func (h *Handler) storeSSOFlow(flow *ssoFlow) {
	h.ssoMu.Lock()
	defer h.ssoMu.Unlock()

	h.gcSSOFlowsLocked(time.Now())
	h.ssoFlows[flow.ID] = flow
	if flow.State != "" {
		h.ssoStates[flow.State] = flow.ID
	}
}

func (h *Handler) getSSOFlow(flowID string) (*ssoFlow, bool) {
	h.ssoMu.Lock()
	defer h.ssoMu.Unlock()

	h.gcSSOFlowsLocked(time.Now())
	flow, ok := h.ssoFlows[flowID]
	if !ok {
		return nil, false
	}
	cp := *flow
	return &cp, true
}

func (h *Handler) setSSOFlowError(flowID, errMsg string) {
	h.ssoMu.Lock()
	defer h.ssoMu.Unlock()

	flow, ok := h.ssoFlows[flowID]
	if !ok {
		return
	}
	flow.Status = ssoFlowError
	flow.Error = errMsg
	flow.UpdatedAt = time.Now()
}

func (h *Handler) gcSSOFlowsLocked(now time.Time) {
	for id, flow := range h.ssoFlows {
		if flow.Status == ssoFlowPending && !flow.ExpiresAt.IsZero() && now.After(flow.ExpiresAt) {
			flow.Status = ssoFlowExpired
			flow.Error = "flow expired"
			flow.UpdatedAt = now
		}

		if flow.Status != ssoFlowPending && now.Sub(flow.UpdatedAt) > ssoFlowGCAge {
			if flow.State != "" {
				delete(h.ssoStates, flow.State)
			}
			delete(h.ssoFlows, id)
		}
	}
}

func newSSOFlowID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("sso_%d", time.Now().UnixNano())
	}
	return "sso_" + hex.EncodeToString(buf)
}

func generateSSOState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func renderSSOCallbackPage(w http.ResponseWriter, flowID, status, title, errMsg string) {
	payload := map[string]string{
		"type":   "octai-sso-result",
		"flowId": flowID,
		"status": status,
	}
	if errMsg != "" {
		payload["error"] = errMsg
	}
	payloadJSON, _ := json.Marshal(payload)

	message := title
	if errMsg != "" {
		message = fmt.Sprintf("%s: %s", title, errMsg)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if status == ssoFlowSuccess {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}

	_, _ = fmt.Fprintf(
		w,
		`<!doctype html><html><head><meta charset="utf-8"><title>OctAi SSO</title></head><body><script>(function(){var payload=%s;var hasOpener=false;try{if(window.opener&&!window.opener.closed){window.opener.postMessage(payload,window.location.origin);hasOpener=true}}catch(e){}var target='/sso/callback?flow_id='+encodeURIComponent(payload.flowId||'')+'&status='+encodeURIComponent(payload.status||'');setTimeout(function(){if(hasOpener){window.close();return}window.location.replace(target)},800)})();</script><div style="font-family:Inter,system-ui,sans-serif;padding:24px"><h2>%s</h2><p>%s</p><p>You can close this window.</p></div></body></html>`,
		string(payloadJSON),
		html.EscapeString(title),
		html.EscapeString(message),
	)
}

func parseFormBody(body string) (map[string][]string, error) {
	result := make(map[string][]string)
	pairs := strings.Split(body, "&")
	for _, pair := range pairs {
		if pair == "" {
			continue
		}
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			result[kv[0]] = append(result[kv[0]], kv[1])
		} else {
			result[kv[0]] = append(result[kv[0]], "")
		}
	}
	return result, nil
}
