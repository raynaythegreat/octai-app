package saml

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Provider interface {
	GetMetadata(ctx context.Context) ([]byte, error)
	Login(ctx context.Context, relayState string) (string, error)
	GetName() string
	GetSSOURL() string
	GetSLOURL() string
}

type ProviderMetadata struct {
	EntityID         string            `xml:"entityID"`
	IDPSSODescriptor *IDPSSODescriptor `xml:"IDPSSODescriptor"`
}

type IDPSSODescriptor struct {
	WantAuthnRequestsSigned bool                  `xml:"wantAuthnRequestsSigned,attr"`
	KeyDescriptors          []KeyDescriptor       `xml:"KeyDescriptor"`
	NameIDFormats           []string              `xml:"NameIDFormat"`
	SingleSignOnServices    []SingleSignOnService `xml:"SingleSignOnService"`
	SingleLogoutServices    []SingleLogoutService `xml:"SingleLogoutService"`
	Attributes              []IDPAttribute        `xml:"Attribute"`
}

type KeyDescriptor struct {
	Use               string   `xml:"use,attr"`
	KeyInfo           KeyInfo  `xml:"KeyInfo"`
	EncryptionMethods []string `xml:"EncryptionMethod"`
}

type KeyInfo struct {
	X509Data X509Data `xml:"X509Data"`
}

type X509Data struct {
	X509Certificates []string `xml:"X509Certificate"`
}

type SingleSignOnService struct {
	Binding  string `xml:"Binding,attr"`
	Location string `xml:"Location,attr"`
}

type SingleLogoutService struct {
	Binding  string `xml:"Binding,attr"`
	Location string `xml:"Location,attr"`
}

type IDPAttribute struct {
	Name       string   `xml:"Name,attr"`
	NameFormat string   `xml:"NameFormat,attr"`
	Values     []string `xml:"AttributeValue"`
}

type OktaProvider struct {
	config *SAMLConfig
	client *http.Client
}

func NewOktaProvider(config *SAMLConfig) *OktaProvider {
	return &OktaProvider{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *OktaProvider) GetName() string {
	return "okta"
}

func (p *OktaProvider) GetSSOURL() string {
	return p.config.IDPSSOURL
}

func (p *OktaProvider) GetSLOURL() string {
	return p.config.IDPSLOURL
}

func (p *OktaProvider) GetMetadata(ctx context.Context) ([]byte, error) {
	metadataURL := strings.TrimSuffix(p.config.IDPSSOURL, "/sso/saml") + "/sso/saml/metadata"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating metadata request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata request failed with status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading metadata response: %w", err)
	}

	return data, nil
}

func (p *OktaProvider) Login(ctx context.Context, relayState string) (string, error) {
	return buildSAMLRequest(p.config, relayState, "")
}

type AzureADProvider struct {
	config   *SAMLConfig
	client   *http.Client
	tenantID string
	appID    string
}

func NewAzureADProvider(config *SAMLConfig, tenantID, appID string) *AzureADProvider {
	return &AzureADProvider{
		config:   config,
		tenantID: tenantID,
		appID:    appID,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *AzureADProvider) GetName() string {
	return "azure"
}

func (p *AzureADProvider) GetSSOURL() string {
	return p.config.IDPSSOURL
}

func (p *AzureADProvider) GetSLOURL() string {
	return p.config.IDPSLOURL
}

func (p *AzureADProvider) GetMetadata(ctx context.Context) ([]byte, error) {
	metadataURL := fmt.Sprintf(
		"https://login.microsoftonline.com/%s/federationmetadata/2007-06/federationmetadata.xml",
		p.tenantID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating metadata request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata request failed with status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading metadata response: %w", err)
	}

	return data, nil
}

func (p *AzureADProvider) Login(ctx context.Context, relayState string) (string, error) {
	return buildSAMLRequest(p.config, relayState, p.appID)
}

type OneLoginProvider struct {
	config    *SAMLConfig
	client    *http.Client
	appID     string
	subdomain string
}

func NewOneLoginProvider(config *SAMLConfig, subdomain, appID string) *OneLoginProvider {
	return &OneLoginProvider{
		config:    config,
		appID:     appID,
		subdomain: subdomain,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *OneLoginProvider) GetName() string {
	return "onelogin"
}

func (p *OneLoginProvider) GetSSOURL() string {
	return p.config.IDPSSOURL
}

func (p *OneLoginProvider) GetSLOURL() string {
	return p.config.IDPSLOURL
}

func (p *OneLoginProvider) GetMetadata(ctx context.Context) ([]byte, error) {
	metadataURL := fmt.Sprintf(
		"https://%s.onelogin.com/saml/metadata/%s",
		p.subdomain,
		p.appID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating metadata request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata request failed with status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading metadata response: %w", err)
	}

	return data, nil
}

func (p *OneLoginProvider) Login(ctx context.Context, relayState string) (string, error) {
	return buildSAMLRequest(p.config, relayState, p.appID)
}

func buildSAMLRequest(config *SAMLConfig, relayState, appID string) (string, error) {
	now := time.Now().UTC()
	issueInstant := now.Format("2006-01-02T15:04:05Z")
	id := fmt.Sprintf("_%s", generateID())

	authnRequest := fmt.Sprintf(`<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" ID="%s" Version="2.0" IssueInstant="%s" Destination="%s" ProtocolBinding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" AssertionConsumerServiceURL="%s">
	<saml:Issuer xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">%s</saml:Issuer>
	<samlp:NameIDPolicy xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress" AllowCreate="true"/>
</samlp:AuthnRequest>`,
		id,
		issueInstant,
		config.IDPSSOURL,
		config.ACSURL,
		config.EntityID,
	)

	return encodeSAMLRequest(authnRequest, relayState), nil
}

func generateID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 20)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}

func encodeSAMLRequest(request, relayState string) string {
	encoded := base64Encode([]byte(request))
	if relayState != "" {
		return encoded + "&RelayState=" + relayState
	}
	return encoded
}

func base64Encode(data []byte) string {
	return string(data)
}

func ParseMetadata(data []byte) (*ProviderMetadata, error) {
	var metadata ProviderMetadata
	if err := xml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("parsing metadata XML: %w", err)
	}
	return &metadata, nil
}

func ExtractCertificateFromMetadata(metadata *ProviderMetadata) (string, error) {
	if metadata.IDPSSODescriptor == nil {
		return "", fmt.Errorf("IDPSSODescriptor not found in metadata")
	}

	for _, kd := range metadata.IDPSSODescriptor.KeyDescriptors {
		if kd.Use == "signing" || kd.Use == "" {
			for _, cert := range kd.KeyInfo.X509Data.X509Certificates {
				if cert != "" {
					return cert, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no signing certificate found in metadata")
}
