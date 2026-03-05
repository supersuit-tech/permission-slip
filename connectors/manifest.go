package connectors

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// ConnectorManifest represents the connector.json manifest file that external
// connector repos must provide. It describes the connector's identity, actions,
// required credentials, and optional configuration templates so the server can
// register and seed DB rows automatically on startup.
type ConnectorManifest struct {
	ID                  string                  `json:"id"`
	Name                string                  `json:"name"`
	Description         string                  `json:"description"`
	Actions             []ManifestAction        `json:"actions"`
	RequiredCredentials []ManifestCredential    `json:"required_credentials"`
	Templates           []ManifestTemplate      `json:"templates,omitempty"`
	OAuthProviders      []ManifestOAuthProvider `json:"oauth_providers,omitempty"`
}

// ManifestAction describes a single action exposed by an external connector.
type ManifestAction struct {
	ActionType       string          `json:"action_type"`
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	RiskLevel        string          `json:"risk_level"`
	ParametersSchema json.RawMessage `json:"parameters_schema,omitempty"`
}

// ManifestCredential describes a credential requirement for an external connector.
type ManifestCredential struct {
	Service         string   `json:"service"`
	AuthType        string   `json:"auth_type"`
	InstructionsURL string   `json:"instructions_url,omitempty"`
	OAuthProvider   string   `json:"oauth_provider,omitempty"`
	OAuthScopes     []string `json:"oauth_scopes,omitempty"`
}

// ManifestOAuthProvider describes an OAuth 2.0 provider declared by an external
// connector. This allows external connectors to register providers that the
// platform doesn't have built-in support for (e.g. Salesforce, HubSpot).
type ManifestOAuthProvider struct {
	ID           string   `json:"id"`
	AuthorizeURL string   `json:"authorize_url"`
	TokenURL     string   `json:"token_url"`
	Scopes       []string `json:"scopes,omitempty"`
}

// ManifestTemplate describes a predefined configuration preset for an action.
// Templates pre-fill the name, description, and parameter constraints when a
// user creates a new action configuration. ID must be globally unique.
type ManifestTemplate struct {
	ID          string          `json:"id"`
	ActionType  string          `json:"action_type"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
}

// maxInstructionsURLLen is the maximum length for an instructions URL,
// matching the CHECK constraint on the DB column.
const maxInstructionsURLLen = 2048

// idPattern matches valid connector IDs: lowercase alphanumeric with hyphens/underscores.
var idPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,62}$`)

// validRiskLevels are the allowed values for action risk levels.
var validRiskLevels = map[string]bool{
	"low":    true,
	"medium": true,
	"high":   true,
}

// validAuthTypes are the allowed values for credential auth types.
// Must match the CHECK constraint on connector_required_credentials.auth_type.
var validAuthTypes = map[string]bool{
	"api_key": true,
	"basic":   true,
	"custom":  true,
	"oauth2":  true,
}

// LoadManifest reads and validates a connector.json manifest from the given path.
func LoadManifest(path string) (*ConnectorManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest %s: %w", path, err)
	}
	return ParseManifest(data)
}

// ParseManifest parses and validates a connector.json manifest from raw JSON bytes.
func ParseManifest(data []byte) (*ConnectorManifest, error) {
	var m ConnectorManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest JSON: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// Validate checks that all required fields are present and well-formed.
func (m *ConnectorManifest) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("manifest validation: id is required")
	}
	if !idPattern.MatchString(m.ID) {
		return fmt.Errorf("manifest validation: id %q must match %s", m.ID, idPattern.String())
	}
	if m.Name == "" {
		return fmt.Errorf("manifest validation: name is required")
	}
	if len(m.Actions) == 0 {
		return fmt.Errorf("manifest validation: at least one action is required")
	}

	actionTypes := make(map[string]bool, len(m.Actions))
	for i, a := range m.Actions {
		if a.ActionType == "" {
			return fmt.Errorf("manifest validation: actions[%d].action_type is required", i)
		}
		// Action types must be prefixed with the connector ID.
		if !strings.HasPrefix(a.ActionType, m.ID+".") {
			return fmt.Errorf("manifest validation: actions[%d].action_type %q must start with %q", i, a.ActionType, m.ID+".")
		}
		if actionTypes[a.ActionType] {
			return fmt.Errorf("manifest validation: duplicate action_type %q", a.ActionType)
		}
		actionTypes[a.ActionType] = true
		if a.Name == "" {
			return fmt.Errorf("manifest validation: actions[%d].name is required", i)
		}
		if a.RiskLevel != "" && !validRiskLevels[a.RiskLevel] {
			return fmt.Errorf("manifest validation: actions[%d].risk_level %q must be low, medium, or high", i, a.RiskLevel)
		}
	}

	// Validate templates (optional).
	// Template IDs must contain the connector ID to prevent cross-connector
	// collisions, since they are stored with a global primary key.
	templateIDs := make(map[string]bool, len(m.Templates))
	for i, tpl := range m.Templates {
		if tpl.ID == "" {
			return fmt.Errorf("manifest validation: templates[%d].id is required", i)
		}
		if !strings.Contains(tpl.ID, m.ID) {
			return fmt.Errorf("manifest validation: templates[%d].id %q must contain the connector id %q to avoid cross-connector collisions", i, tpl.ID, m.ID)
		}
		if templateIDs[tpl.ID] {
			return fmt.Errorf("manifest validation: duplicate template id %q", tpl.ID)
		}
		templateIDs[tpl.ID] = true
		if tpl.ActionType == "" {
			return fmt.Errorf("manifest validation: templates[%d].action_type is required", i)
		}
		if !actionTypes[tpl.ActionType] {
			return fmt.Errorf("manifest validation: templates[%d].action_type %q does not match any declared action", i, tpl.ActionType)
		}
		if tpl.Name == "" {
			return fmt.Errorf("manifest validation: templates[%d].name is required", i)
		}
		if len(tpl.Parameters) == 0 {
			return fmt.Errorf("manifest validation: templates[%d].parameters is required", i)
		}
		// Verify parameters is a valid JSON object (not an array, string, etc.).
		var params map[string]json.RawMessage
		if err := json.Unmarshal(tpl.Parameters, &params); err != nil {
			return fmt.Errorf("manifest validation: templates[%d].parameters must be a JSON object: %w", i, err)
		}
	}

	services := make(map[string]bool, len(m.RequiredCredentials))
	for i, c := range m.RequiredCredentials {
		if c.Service == "" {
			return fmt.Errorf("manifest validation: required_credentials[%d].service is required", i)
		}
		if services[c.Service] {
			return fmt.Errorf("manifest validation: duplicate credential service %q", c.Service)
		}
		services[c.Service] = true
		if c.AuthType == "" {
			return fmt.Errorf("manifest validation: required_credentials[%d].auth_type is required", i)
		}
		if !validAuthTypes[c.AuthType] {
			return fmt.Errorf("manifest validation: required_credentials[%d].auth_type %q must be api_key, basic, custom, or oauth2", i, c.AuthType)
		}
		// OAuth2-specific validation.
		if c.AuthType == "oauth2" {
			if c.OAuthProvider == "" {
				return fmt.Errorf("manifest validation: required_credentials[%d].oauth_provider is required when auth_type is oauth2", i)
			}
		} else {
			if c.OAuthProvider != "" {
				return fmt.Errorf("manifest validation: required_credentials[%d].oauth_provider must be empty when auth_type is %q", i, c.AuthType)
			}
			if len(c.OAuthScopes) > 0 {
				return fmt.Errorf("manifest validation: required_credentials[%d].oauth_scopes must be empty when auth_type is %q", i, c.AuthType)
			}
		}
		if c.InstructionsURL != "" {
			if len(c.InstructionsURL) > maxInstructionsURLLen {
				return fmt.Errorf("manifest validation: required_credentials[%d].instructions_url exceeds %d characters", i, maxInstructionsURLLen)
			}
			u, err := url.Parse(c.InstructionsURL)
			if err != nil {
				return fmt.Errorf("manifest validation: required_credentials[%d].instructions_url is not a valid URL: %w", i, err)
			}
			if u.Scheme != "http" && u.Scheme != "https" {
				return fmt.Errorf("manifest validation: required_credentials[%d].instructions_url must use http or https scheme", i)
			}
			if u.Host == "" {
				return fmt.Errorf("manifest validation: required_credentials[%d].instructions_url must include a host", i)
			}
		}
	}

	// Collect all declared OAuth provider IDs (from OAuthProviders section)
	// plus well-known built-in providers that don't need to be declared.
	declaredProviders := map[string]bool{
		"google":    true,
		"microsoft": true,
	}

	// Validate OAuth providers (optional, used by external connectors).
	providerIDs := make(map[string]bool, len(m.OAuthProviders))
	for i, p := range m.OAuthProviders {
		if p.ID == "" {
			return fmt.Errorf("manifest validation: oauth_providers[%d].id is required", i)
		}
		if providerIDs[p.ID] {
			return fmt.Errorf("manifest validation: duplicate oauth_provider id %q", p.ID)
		}
		providerIDs[p.ID] = true

		if p.AuthorizeURL == "" {
			return fmt.Errorf("manifest validation: oauth_providers[%d].authorize_url is required", i)
		}
		au, err := url.Parse(p.AuthorizeURL)
		if err != nil {
			return fmt.Errorf("manifest validation: oauth_providers[%d].authorize_url is not a valid URL: %w", i, err)
		}
		if au.Scheme != "https" {
			return fmt.Errorf("manifest validation: oauth_providers[%d].authorize_url must use https scheme", i)
		}

		if p.TokenURL == "" {
			return fmt.Errorf("manifest validation: oauth_providers[%d].token_url is required", i)
		}
		tu, err := url.Parse(p.TokenURL)
		if err != nil {
			return fmt.Errorf("manifest validation: oauth_providers[%d].token_url is not a valid URL: %w", i, err)
		}
		if tu.Scheme != "https" {
			return fmt.Errorf("manifest validation: oauth_providers[%d].token_url must use https scheme", i)
		}

		declaredProviders[p.ID] = true
	}

	// Cross-reference: verify oauth_provider in credentials references a known provider.
	for i, c := range m.RequiredCredentials {
		if c.AuthType == "oauth2" && !declaredProviders[c.OAuthProvider] {
			return fmt.Errorf("manifest validation: required_credentials[%d].oauth_provider %q is not a built-in provider and not declared in oauth_providers", i, c.OAuthProvider)
		}
	}

	return nil
}

// ToDBManifest converts a ConnectorManifest into the db.ExternalConnectorManifest
// struct the DB upsert layer expects. This centralises the conversion so callers
// (main.go, cmd/seed, etc.) don't duplicate the field-mapping logic.
func (m *ConnectorManifest) ToDBManifest() db.ExternalConnectorManifest {
	out := db.ExternalConnectorManifest{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
	}
	for _, a := range m.Actions {
		out.Actions = append(out.Actions, db.ExternalConnectorAction{
			ActionType:       a.ActionType,
			Name:             a.Name,
			Description:      a.Description,
			RiskLevel:        a.RiskLevel,
			ParametersSchema: a.ParametersSchema,
		})
	}
	for _, c := range m.RequiredCredentials {
		out.Credentials = append(out.Credentials, db.ExternalConnectorCredential{
			Service:         c.Service,
			AuthType:        c.AuthType,
			InstructionsURL: c.InstructionsURL,
			OAuthProvider:   c.OAuthProvider,
			OAuthScopes:     c.OAuthScopes,
		})
	}
	for _, tpl := range m.Templates {
		out.Templates = append(out.Templates, db.ExternalConnectorTemplate{
			ID:          tpl.ID,
			ActionType:  tpl.ActionType,
			Name:        tpl.Name,
			Description: tpl.Description,
			Parameters:  tpl.Parameters,
		})
	}
	return out
}
