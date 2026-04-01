package saml

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type SAMLConfig struct {
	IDPSSOURL        string        `json:"idp_sso_url"`
	IDPSLOURL        string        `json:"idp_slo_url"`
	SPSLOURL         string        `json:"sp_slo_url"`
	EntityID         string        `json:"entity_id"`
	Certificate      string        `json:"certificate"`
	PrivateKey       string        `json:"private_key"`
	IDPCertificate   string        `json:"idp_certificate"`
	ACSURL           string        `json:"acs_url"`
	AttributeMap     AttributeMap  `json:"attribute_map"`
	AllowedClockSkew time.Duration `json:"allowed_clock_skew"`
	Provider         string        `json:"provider"`
}

type AttributeMap struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	NameID    string `json:"name_id"`
	Groups    string `json:"groups"`
	UID       string `json:"uid"`
}

func DefaultAttributeMap() AttributeMap {
	return AttributeMap{
		Email:     "email",
		FirstName: "firstName",
		LastName:  "lastName",
		NameID:    "name_id",
		Groups:    "groups",
		UID:       "uid",
	}
}

func LoadFromEnvironment(prefix string) (*SAMLConfig, error) {
	if prefix == "" {
		prefix = "SAML"
	}

	cfg := &SAMLConfig{
		IDPSSOURL:        os.Getenv(fmt.Sprintf("%s_IDP_SSO_URL", prefix)),
		IDPSLOURL:        os.Getenv(fmt.Sprintf("%s_IDP_SLO_URL", prefix)),
		SPSLOURL:         os.Getenv(fmt.Sprintf("%s_SP_SLO_URL", prefix)),
		EntityID:         os.Getenv(fmt.Sprintf("%s_ENTITY_ID", prefix)),
		Certificate:      os.Getenv(fmt.Sprintf("%s_CERTIFICATE", prefix)),
		PrivateKey:       os.Getenv(fmt.Sprintf("%s_PRIVATE_KEY", prefix)),
		IDPCertificate:   os.Getenv(fmt.Sprintf("%s_IDP_CERTIFICATE", prefix)),
		ACSURL:           os.Getenv(fmt.Sprintf("%s_ACS_URL", prefix)),
		Provider:         os.Getenv(fmt.Sprintf("%s_PROVIDER", prefix)),
		AllowedClockSkew: 5 * time.Minute,
		AttributeMap:     DefaultAttributeMap(),
	}

	if emailAttr := os.Getenv(fmt.Sprintf("%s_ATTR_EMAIL", prefix)); emailAttr != "" {
		cfg.AttributeMap.Email = emailAttr
	}
	if firstNameAttr := os.Getenv(fmt.Sprintf("%s_ATTR_FIRST_NAME", prefix)); firstNameAttr != "" {
		cfg.AttributeMap.FirstName = firstNameAttr
	}
	if lastNameAttr := os.Getenv(fmt.Sprintf("%s_ATTR_LAST_NAME", prefix)); lastNameAttr != "" {
		cfg.AttributeMap.LastName = lastNameAttr
	}
	if groupsAttr := os.Getenv(fmt.Sprintf("%s_ATTR_GROUPS", prefix)); groupsAttr != "" {
		cfg.AttributeMap.Groups = groupsAttr
	}
	if uidAttr := os.Getenv(fmt.Sprintf("%s_ATTR_UID", prefix)); uidAttr != "" {
		cfg.AttributeMap.UID = uidAttr
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *SAMLConfig) Validate() error {
	var errs []string

	if c.IDPSSOURL == "" {
		errs = append(errs, "IDPSSOURL is required")
	}
	if c.EntityID == "" {
		errs = append(errs, "EntityID is required")
	}
	if c.IDPCertificate == "" {
		errs = append(errs, "IDPCertificate is required")
	}
	if c.ACSURL == "" {
		errs = append(errs, "ACSURL is required")
	}

	if c.AllowedClockSkew < 0 {
		errs = append(errs, "AllowedClockSkew must be non-negative")
	}

	if len(errs) > 0 {
		return fmt.Errorf("SAML config validation failed: %s", strings.Join(errs, "; "))
	}

	return nil
}

func (c *SAMLConfig) GetProviderType() string {
	if c.Provider != "" {
		return strings.ToLower(c.Provider)
	}

	if strings.Contains(strings.ToLower(c.IDPSSOURL), "okta") {
		return "okta"
	}
	if strings.Contains(strings.ToLower(c.IDPSSOURL), "microsoft") ||
		strings.Contains(strings.ToLower(c.IDPSSOURL), "azure") {
		return "azure"
	}
	if strings.Contains(strings.ToLower(c.IDPSSOURL), "onelogin") {
		return "onelogin"
	}

	return "generic"
}
