package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/logger"
)

type OIDCConfig struct {
	Issuer       string   `json:"issuer"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURI  string   `json:"redirect_uri"`
	Scopes       []string `json:"scopes"`
}

type User struct {
	ID            string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	FirstName     string `json:"given_name"`
	LastName      string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
	Provider      string `json:"provider"`
}

type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	IDToken      string    `json:"id_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scope        string    `json:"scope"`
}

type OIDCDiscovery struct {
	Issuer                string   `json:"issuer"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	UserInfoEndpoint      string   `json:"userinfo_endpoint"`
	JWKSEndpoint          string   `json:"jwks_uri"`
	EndSessionEndpoint    string   `json:"end_session_endpoint"`
	ScopesSupported       []string `json:"scopes_supported"`
}

type OIDCService struct {
	config     *OIDCConfig
	discovery  *OIDCDiscovery
	httpClient *http.Client
	provider   string
}

func NewOIDCService(config *OIDCConfig, provider string) *OIDCService {
	return &OIDCService{
		config:   config,
		provider: provider,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *OIDCService) Init(ctx context.Context) error {
	if s.config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if s.config.Issuer == "" {
		return fmt.Errorf("issuer is required")
	}

	if s.config.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}

	if s.config.RedirectURI == "" {
		return fmt.Errorf("redirect_uri is required")
	}

	discovery, err := s.fetchDiscovery(ctx)
	if err != nil {
		return fmt.Errorf("fetching OIDC discovery: %w", err)
	}

	s.discovery = discovery

	if len(s.config.Scopes) == 0 {
		s.config.Scopes = []string{"openid", "email", "profile"}
	}

	logger.InfoC("oidc", fmt.Sprintf("OIDC service initialized for provider: %s", s.provider))

	return nil
}

func (s *OIDCService) fetchDiscovery(ctx context.Context) (*OIDCDiscovery, error) {
	discoveryURL := strings.TrimSuffix(s.config.Issuer, "/") + "/.well-known/openid-configuration"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating discovery request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discovery request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var discovery OIDCDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return nil, fmt.Errorf("decoding discovery document: %w", err)
	}

	return &discovery, nil
}

func (s *OIDCService) GetAuthURL(ctx context.Context, state string) (string, error) {
	if s.discovery == nil {
		return "", fmt.Errorf("service not initialized")
	}

	if state == "" {
		return "", fmt.Errorf("state is required")
	}

	params := url.Values{
		"response_type": {"code"},
		"client_id":     {s.config.ClientID},
		"redirect_uri":  {s.config.RedirectURI},
		"scope":         {strings.Join(s.config.Scopes, " ")},
		"state":         {state},
	}

	if s.provider == "google" {
		params.Set("access_type", "offline")
		params.Set("prompt", "consent")
	}

	authURL := s.discovery.AuthorizationEndpoint + "?" + params.Encode()

	logger.DebugCF("oidc", "Generated auth URL", map[string]any{
		"provider": s.provider,
		"state":    state,
	})

	return authURL, nil
}

func (s *OIDCService) HandleCallback(ctx context.Context, code, state string) (*User, *Token, error) {
	if code == "" {
		return nil, nil, fmt.Errorf("authorization code is required")
	}

	token, err := s.exchangeCode(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("exchanging code: %w", err)
	}

	user, err := s.fetchUserInfo(ctx, token.AccessToken)
	if err != nil {
		return nil, token, fmt.Errorf("fetching user info: %w", err)
	}

	user.Provider = s.provider

	logger.InfoCF("oidc", "OIDC login successful", map[string]any{
		"email":    user.Email,
		"provider": user.Provider,
	})

	return user, token, nil
}

func (s *OIDCService) exchangeCode(ctx context.Context, code string) (*Token, error) {
	if s.discovery == nil {
		return nil, fmt.Errorf("service not initialized")
	}

	data := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {s.config.RedirectURI},
		"client_id":    {s.config.ClientID},
	}

	if s.config.ClientSecret != "" {
		data.Set("client_secret", s.config.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.discovery.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}

	token := &Token{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		IDToken:      tokenResp.IDToken,
		TokenType:    tokenResp.TokenType,
		ExpiresIn:    tokenResp.ExpiresIn,
		Scope:        tokenResp.Scope,
	}

	if tokenResp.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	return token, nil
}

func (s *OIDCService) fetchUserInfo(ctx context.Context, accessToken string) (*User, error) {
	if s.discovery == nil {
		return nil, fmt.Errorf("service not initialized")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.discovery.UserInfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating userinfo request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading userinfo response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("parsing userinfo response: %w", err)
	}

	return &user, nil
}

func (s *OIDCService) RefreshToken(ctx context.Context, refreshToken string) (*Token, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh token is required")
	}

	if s.discovery == nil {
		return nil, fmt.Errorf("service not initialized")
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {s.config.ClientID},
	}

	if s.config.ClientSecret != "" {
		data.Set("client_secret", s.config.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.discovery.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parsing refresh response: %w", err)
	}

	token := &Token{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		IDToken:      tokenResp.IDToken,
		TokenType:    tokenResp.TokenType,
		ExpiresIn:    tokenResp.ExpiresIn,
		Scope:        tokenResp.Scope,
	}

	if tokenResp.RefreshToken == "" {
		token.RefreshToken = refreshToken
	}

	if tokenResp.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	logger.DebugCF("oidc", "Token refreshed successfully", map[string]any{
		"provider": s.provider,
	})

	return token, nil
}

func (s *OIDCService) GetLogoutURL(postLogoutRedirectURI string) (string, error) {
	if s.discovery == nil || s.discovery.EndSessionEndpoint == "" {
		return "", fmt.Errorf("end session endpoint not available")
	}

	logoutURL := s.discovery.EndSessionEndpoint

	if postLogoutRedirectURI != "" {
		params := url.Values{
			"post_logout_redirect_uri": {postLogoutRedirectURI},
		}
		logoutURL = logoutURL + "?" + params.Encode()
	}

	return logoutURL, nil
}

func (s *OIDCService) GetDiscovery() *OIDCDiscovery {
	return s.discovery
}

func GenerateState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func GenerateNonce() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func GoogleOIDCConfig(clientID, clientSecret, redirectURI string) *OIDCConfig {
	return &OIDCConfig{
		Issuer:       "https://accounts.google.com",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
		Scopes:       []string{"openid", "email", "profile"},
	}
}

func AzureADOIDCConfig(tenantID, clientID, clientSecret, redirectURI string) *OIDCConfig {
	return &OIDCConfig{
		Issuer:       fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", tenantID),
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
		Scopes:       []string{"openid", "email", "profile"},
	}
}

func OktaOIDCConfig(domain, clientID, clientSecret, redirectURI string) *OIDCConfig {
	return &OIDCConfig{
		Issuer:       fmt.Sprintf("https://%s", domain),
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
		Scopes:       []string{"openid", "email", "profile", "groups"},
	}
}
