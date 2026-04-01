package saml

import (
	"bytes"
	"compress/flate"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/logger"
)

type User struct {
	ID         string                 `json:"id"`
	Email      string                 `json:"email"`
	FirstName  string                 `json:"first_name,omitempty"`
	LastName   string                 `json:"last_name,omitempty"`
	NameID     string                 `json:"name_id"`
	Groups     []string               `json:"groups,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	Provider   string                 `json:"provider"`
}

type SAMLService struct {
	config   *SAMLConfig
	provider Provider
	idpCert  *x509.Certificate
	spCert   *x509.Certificate
	spKey    *rsa.PrivateKey
}

func NewSAMLService() *SAMLService {
	return &SAMLService{}
}

func (s *SAMLService) Init(config *SAMLConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	s.config = config

	if config.IDPCertificate != "" {
		cert, err := parseCertificate(config.IDPCertificate)
		if err != nil {
			return fmt.Errorf("parsing IDP certificate: %w", err)
		}
		s.idpCert = cert
	}

	if config.Certificate != "" && config.PrivateKey != "" {
		cert, err := parseCertificate(config.Certificate)
		if err != nil {
			return fmt.Errorf("parsing SP certificate: %w", err)
		}
		s.spCert = cert

		key, err := parsePrivateKey(config.PrivateKey)
		if err != nil {
			return fmt.Errorf("parsing SP private key: %w", err)
		}
		s.spKey = key
	}

	s.provider = s.createProvider(config)

	logger.InfoC("saml", fmt.Sprintf("SAML service initialized with provider: %s", config.GetProviderType()))

	return nil
}

func (s *SAMLService) createProvider(config *SAMLConfig) Provider {
	switch config.GetProviderType() {
	case "okta":
		return NewOktaProvider(config)
	case "azure":
		return NewAzureADProvider(config, "", "")
	case "onelogin":
		return NewOneLoginProvider(config, "", "")
	default:
		return NewOktaProvider(config)
	}
}

func (s *SAMLService) GetProvider() Provider {
	return s.provider
}

func (s *SAMLService) GetMetadata(ctx context.Context) ([]byte, error) {
	if s.provider == nil {
		return nil, fmt.Errorf("provider not initialized")
	}

	return s.provider.GetMetadata(ctx)
}

func (s *SAMLService) HandleLogin(ctx context.Context, samlResponse string) (*User, error) {
	if samlResponse == "" {
		return nil, fmt.Errorf("SAML response cannot be empty")
	}

	decoded, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		return nil, fmt.Errorf("decoding SAML response: %w", err)
	}

	if err := s.ValidateSignature(string(decoded), s.config.IDPCertificate); err != nil {
		return nil, fmt.Errorf("signature validation failed: %w", err)
	}

	attributes, err := s.ExtractAttributes(string(decoded))
	if err != nil {
		return nil, fmt.Errorf("extracting attributes: %w", err)
	}

	user := &User{
		Attributes: attributes,
		Provider:   s.config.GetProviderType(),
	}

	if email, ok := attributes[s.config.AttributeMap.Email].(string); ok {
		user.Email = email
	}
	if firstName, ok := attributes[s.config.AttributeMap.FirstName].(string); ok {
		user.FirstName = firstName
	}
	if lastName, ok := attributes[s.config.AttributeMap.LastName].(string); ok {
		user.LastName = lastName
	}
	if nameID, ok := attributes[s.config.AttributeMap.NameID].(string); ok {
		user.NameID = nameID
		user.ID = nameID
	}
	if uid, ok := attributes[s.config.AttributeMap.UID].(string); ok && uid != "" {
		user.ID = uid
	}
	if groups, ok := attributes[s.config.AttributeMap.Groups].([]string); ok {
		user.Groups = groups
	}

	if user.ID == "" {
		user.ID = user.Email
	}

	if user.Email == "" {
		return nil, fmt.Errorf("no email found in SAML assertion")
	}

	logger.InfoCF("saml", "SAML login successful", map[string]any{
		"email":    user.Email,
		"provider": user.Provider,
	})

	return user, nil
}

func (s *SAMLService) ParseSAMLResponse(encoded string) (map[string]string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed: %w", err)
	}

	decompressed, err := decompressResponse(decoded)
	if err != nil {
		decoded = decompressed
	}

	result := make(map[string]string)

	var response struct {
		Assertion struct {
			AttributeStatement struct {
				Attributes []struct {
					Name   string   `xml:"Name,attr"`
					Values []string `xml:"AttributeValue"`
				} `xml:"Attribute"`
			} `xml:"AttributeStatement"`
			Subject struct {
				NameID struct {
				} `xml:"NameID"`
			} `xml:"Subject"`
		} `xml:"Assertion"`
	}

	if err := xml.Unmarshal(decoded, &response); err != nil {
		return nil, fmt.Errorf("XML unmarshal failed: %w", err)
	}

	for _, attr := range response.Assertion.AttributeStatement.Attributes {
		if len(attr.Values) > 0 {
			result[attr.Name] = attr.Values[0]
		}
	}

	return result, nil
}

func (s *SAMLService) ValidateSignature(xmlData, cert string) error {
	if cert == "" {
		logger.WarnC("saml", "No IDP certificate provided, skipping signature validation")
		return nil
	}

	if s.idpCert == nil {
		return fmt.Errorf("IDP certificate not loaded")
	}

	signature, signedInfo, err := extractSignature(xmlData)
	if err != nil {
		return fmt.Errorf("extracting signature: %w", err)
	}

	if signature == nil {
		return nil
	}

	canonicalized, err := canonicalizeXML(signedInfo)
	if err != nil {
		return fmt.Errorf("canonicalizing signed info: %w", err)
	}

	hashed := sha256.Sum256([]byte(canonicalized))

	if err := rsa.VerifyPKCS1v15(s.idpCert.PublicKey.(*rsa.PublicKey), crypto.SHA256, hashed[:], signature); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

func (s *SAMLService) ExtractAttributes(assertion string) (map[string]interface{}, error) {
	attributes := make(map[string]interface{})

	var samlResponse struct {
		Assertion struct {
			ID           string `xml:"ID,attr"`
			IssueInstant string `xml:"IssueInstant,attr"`
			Issuer       string `xml:"Issuer"`
			Subject      struct {
				NameID struct {
					Format string `xml:"Format,attr"`
					Value  string `xml:",chardata"`
				} `xml:"NameID"`
			} `xml:"Subject"`
			Conditions struct {
				NotBefore    string `xml:"NotBefore,attr"`
				NotOnOrAfter string `xml:"NotOnOrAfter,attr"`
			} `xml:"Conditions"`
			AttributeStatement struct {
				Attributes []struct {
					Name       string   `xml:"Name,attr"`
					NameFormat string   `xml:"NameFormat,attr"`
					Values     []string `xml:"AttributeValue"`
				} `xml:"Attribute"`
			} `xml:"AttributeStatement"`
			AuthnStatement struct {
				AuthnInstant        string `xml:"AuthnInstant,attr"`
				SessionIndex        string `xml:"SessionIndex,attr"`
				SessionNotOnOrAfter string `xml:"SessionNotOnOrAfter,attr"`
				AuthnContext        struct {
					AuthnContextClassRef string `xml:"AuthnContextClassRef"`
				} `xml:"AuthnContext"`
			} `xml:"AuthnStatement"`
		} `xml:"Assertion"`
	}

	if err := xml.Unmarshal([]byte(assertion), &samlResponse); err != nil {
		return nil, fmt.Errorf("parsing assertion XML: %w", err)
	}

	if samlResponse.Assertion.Subject.NameID.Value != "" {
		attributes[s.config.AttributeMap.NameID] = samlResponse.Assertion.Subject.NameID.Value
	}

	for _, attr := range samlResponse.Assertion.AttributeStatement.Attributes {
		if len(attr.Values) == 1 {
			attributes[attr.Name] = attr.Values[0]
		} else if len(attr.Values) > 1 {
			attributes[attr.Name] = attr.Values
		}
	}

	if s.config.AllowedClockSkew > 0 {
		if err := validateConditions(samlResponse.Assertion.Conditions, s.config.AllowedClockSkew); err != nil {
			return nil, fmt.Errorf("conditions validation: %w", err)
		}
	}

	return attributes, nil
}

func (s *SAMLService) GenerateAuthnRequest(relayState string) (string, error) {
	if s.provider == nil {
		return "", fmt.Errorf("provider not initialized")
	}

	return s.provider.Login(context.Background(), relayState)
}

func parseCertificate(certPEM string) (*x509.Certificate, error) {
	block, _ := pemDecode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing certificate: %w", err)
	}

	return cert, nil
}

func parsePrivateKey(keyPEM string) (*rsa.PrivateKey, error) {
	block, _ := pemDecode(keyPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		key2, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parsing private key: %w", err)
		}
		rsaKey, ok := key2.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}
		return rsaKey, nil
	}

	return key, nil
}

func pemDecode(data string) (*pemBlock, error) {
	data = strings.TrimSpace(data)
	if !strings.HasPrefix(data, "-----BEGIN") {
		data = "-----BEGIN CERTIFICATE-----\n" + data + "\n-----END CERTIFICATE-----"
	}

	var block pemBlock
	rest := []byte(data)
	for {
		var b *pemBlock
		b, rest = decodePEM(rest)
		if b == nil {
			break
		}
		block = *b
	}

	if block.Bytes == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	return &block, nil
}

type pemBlock struct {
	Type  string
	Bytes []byte
}

func decodePEM(data []byte) (*pemBlock, []byte) {
	startMarker := []byte("-----BEGIN ")
	endMarker := []byte("-----END ")

	startIdx := bytes.Index(data, startMarker)
	if startIdx == -1 {
		return nil, nil
	}

	data = data[startIdx:]
	endStart := bytes.Index(data, []byte("-----\n"))
	if endStart == -1 {
		endStart = bytes.Index(data, []byte("-----\r\n"))
	}
	if endStart == -1 {
		return nil, nil
	}

	typeEnd := bytes.Index(data, []byte("-----"))
	if typeEnd == -1 {
		return nil, nil
	}
	blockType := string(data[len(startMarker):typeEnd])

	headerEnd := bytes.Index(data, []byte("\n"))
	if headerEnd == -1 {
		return nil, nil
	}

	endIdx := bytes.Index(data, endMarker)
	if endIdx == -1 {
		return nil, nil
	}

	endTypeEnd := bytes.Index(data[endIdx:], []byte("-----"))
	if endTypeEnd == -1 {
		return nil, nil
	}

	footerEnd := bytes.Index(data[endIdx+endTypeEnd:], []byte("-----"))
	if footerEnd == -1 {
		return nil, nil
	}

	base64Data := data[headerEnd+1 : endIdx]
	base64Data = bytes.ReplaceAll(base64Data, []byte("\n"), []byte(""))
	base64Data = bytes.ReplaceAll(base64Data, []byte("\r"), []byte(""))
	base64Data = bytes.ReplaceAll(base64Data, []byte(" "), []byte(""))

	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(base64Data)))
	n, err := base64.StdEncoding.Decode(decoded, base64Data)
	if err != nil {
		return nil, nil
	}

	rest := data[endIdx+endTypeEnd+footerEnd+5:]

	return &pemBlock{
		Type:  blockType,
		Bytes: decoded[:n],
	}, rest
}

func decompressResponse(data []byte) ([]byte, error) {
	reader := flate.NewReader(bytes.NewReader(data))
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		return nil, fmt.Errorf("decompression failed: %w", err)
	}

	return buf.Bytes(), nil
}

func extractSignature(xmlData string) ([]byte, string, error) {
	var doc struct {
		Assertion struct {
			Signature struct {
				SignedInfo     string `xml:",innerxml"`
				SignatureValue string `xml:"SignatureValue"`
			} `xml:"Signature"`
		} `xml:"Assertion"`
		Response struct {
			Signature struct {
				SignedInfo     string `xml:",innerxml"`
				SignatureValue string `xml:"SignatureValue"`
			} `xml:"Signature"`
		} `xml:",chardata"`
	}

	if err := xml.Unmarshal([]byte(xmlData), &doc); err != nil {
		return nil, "", fmt.Errorf("parsing XML for signature: %w", err)
	}

	var sigValue string
	var signedInfo string

	if doc.Assertion.Signature.SignatureValue != "" {
		sigValue = doc.Assertion.Signature.SignatureValue
		signedInfo = doc.Assertion.Signature.SignedInfo
	} else if doc.Response.Signature.SignatureValue != "" {
		sigValue = doc.Response.Signature.SignatureValue
		signedInfo = doc.Response.Signature.SignedInfo
	}

	if sigValue == "" {
		return nil, "", nil
	}

	signature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(sigValue))
	if err != nil {
		return nil, "", fmt.Errorf("decoding signature value: %w", err)
	}

	return signature, signedInfo, nil
}

func canonicalizeXML(xmlData string) (string, error) {
	xmlData = strings.ReplaceAll(xmlData, "\r\n", "\n")
	xmlData = strings.ReplaceAll(xmlData, "\r", "\n")
	xmlData = strings.TrimSpace(xmlData)
	return xmlData, nil
}

func validateConditions(conditions struct {
	NotBefore    string `xml:"NotBefore,attr"`
	NotOnOrAfter string `xml:"NotOnOrAfter,attr"`
}, allowedSkew time.Duration) error {
	now := time.Now().UTC()

	if conditions.NotBefore != "" {
		notBefore, err := time.Parse(time.RFC3339, conditions.NotBefore)
		if err != nil {
			return fmt.Errorf("parsing NotBefore: %w", err)
		}
		if now.Add(allowedSkew).Before(notBefore) {
			return fmt.Errorf("assertion not yet valid (NotBefore: %s)", conditions.NotBefore)
		}
	}

	if conditions.NotOnOrAfter != "" {
		notOnOrAfter, err := time.Parse(time.RFC3339, conditions.NotOnOrAfter)
		if err != nil {
			return fmt.Errorf("parsing NotOnOrAfter: %w", err)
		}
		if now.Add(-allowedSkew).After(notOnOrAfter) {
			return fmt.Errorf("assertion expired (NotOnOrAfter: %s)", conditions.NotOnOrAfter)
		}
	}

	return nil
}
