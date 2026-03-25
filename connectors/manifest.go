package connectors

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"sort"
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
	Status              string                  `json:"status,omitempty"`
	LogoSVG             string                  `json:"logo_svg,omitempty"`
	Actions             []ManifestAction        `json:"actions"`
	RequiredCredentials []ManifestCredential    `json:"required_credentials"`
	Templates           []ManifestTemplate      `json:"templates,omitempty"`
	OAuthProviders      []ManifestOAuthProvider `json:"oauth_providers,omitempty"`
}

// ManifestAction describes a single action exposed by an external connector.
type ManifestAction struct {
	ActionType            string          `json:"action_type"`
	Name                  string          `json:"name"`
	Description           string          `json:"description"`
	RiskLevel             string          `json:"risk_level"`
	ParametersSchema      json.RawMessage `json:"parameters_schema,omitempty"`
	RequiresPaymentMethod bool            `json:"requires_payment_method,omitempty"`
	DisplayTemplate       string          `json:"display_template,omitempty"`
	Preview               *ActionPreview  `json:"preview,omitempty"`
}

// ActionPreview defines a structured layout for rich rendering of an action's
// parameters in the approval UI. Each layout type expects specific field roles
// that map to parameter names from the action's schema.
//
// Supported layouts:
//   - "event":   fields "title", "start", "end"
//   - "message": fields "to", "subject", "body"
//   - "record":  fields "title", "subtitle" (plus any extras)
type ActionPreview struct {
	Layout string            `json:"layout"`
	Fields map[string]string `json:"fields"`
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
	ID              string            `json:"id"`
	AuthorizeURL    string            `json:"authorize_url"`
	TokenURL        string            `json:"token_url"`
	Scopes          []string          `json:"scopes,omitempty"`
	AuthorizeParams map[string]string `json:"authorize_params,omitempty"`
	PKCE            bool              `json:"pkce,omitempty"`
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

// validStatuses are the allowed values for connector status.
var validStatuses = map[string]bool{
	"tested":        true,
	"early_preview": true,
	"untested":      true,
}

// validRiskLevels are the allowed values for action risk levels.
var validRiskLevels = map[string]bool{
	"low":    true,
	"medium": true,
	"high":   true,
}

// validPreviewLayouts are the allowed values for ActionPreview.Layout.
var validPreviewLayouts = map[string]bool{
	"event":   true,
	"message": true,
	"record":  true,
}

// validWidgets are the allowed values for x-ui.widget on schema properties.
var validWidgets = map[string]bool{
	"text":                true,
	"select":              true,
	"multi-select":        true,
	"remote-select":       true,
	"remote-multi-select": true,
	"textarea":            true,
	"toggle":              true,
	"number":              true,
	"date":                true,
	"datetime":            true,
	"list":                true,
}

// validAuthTypes are the allowed values for credential auth types.
// Must match the CHECK constraint on connector_required_credentials.auth_type.
var validAuthTypes = map[string]bool{
	"api_key": true,
	"basic":   true,
	"custom":  true,
	"oauth2":  true,
}

// ReservedAuthorizeParams lists OAuth 2.0 parameters that must not appear in
// a manifest's authorize_params or be passed through to the authorization URL.
// Allowing these to be set by connectors would let a malicious or misconfigured
// manifest override security-critical values (redirect_uri, state, client_id)
// that the platform manages.
var ReservedAuthorizeParams = map[string]bool{
	"redirect_uri":          true,
	"state":                 true,
	"client_id":             true,
	"client_secret":         true,
	"response_type":         true,
	"code":                  true,
	"grant_type":            true,
	"code_challenge":        true,
	"code_challenge_method": true,
}

// ReservedTokenParams lists OAuth 2.0 parameters that belong to the token
// exchange request and must not be set by connectors. code_verifier is a
// token exchange parameter (RFC 7636 §4.5), not an authorization parameter.
var ReservedTokenParams = map[string]bool{
	"code_verifier": true,
}

// validateURL parses a URL and checks scheme and host. allowedSchemes specifies
// which schemes are accepted. Returns a descriptive error if invalid.
func validateURL(raw, fieldName string, allowedSchemes ...string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%s is not a valid URL: %w", fieldName, err)
	}
	schemeOK := false
	for _, s := range allowedSchemes {
		if u.Scheme == s {
			schemeOK = true
			break
		}
	}
	if !schemeOK {
		return fmt.Errorf("%s must use %s scheme", fieldName, strings.Join(allowedSchemes, " or "))
	}
	if u.Host == "" {
		return fmt.Errorf("%s must include a host", fieldName)
	}
	return nil
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
	if m.Status != "" && !validStatuses[m.Status] {
		return fmt.Errorf("manifest validation: status %q must be tested, early_preview, or untested", m.Status)
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
		if a.Preview != nil {
			if a.Preview.Layout == "" {
				return fmt.Errorf("manifest validation: actions[%d].preview.layout is required", i)
			}
			if !validPreviewLayouts[a.Preview.Layout] {
				return fmt.Errorf("manifest validation: actions[%d].preview.layout %q must be event, message, or record", i, a.Preview.Layout)
			}
			if len(a.Preview.Fields) == 0 {
				return fmt.Errorf("manifest validation: actions[%d].preview.fields is required", i)
			}
			// Validate that layout-specific required field roles are present.
			requiredFieldsByLayout := map[string][]string{
				"event":   {"title", "start", "end"},
				"message": {"to"},
				"record":  {"title"},
			}
			if required, ok := requiredFieldsByLayout[a.Preview.Layout]; ok {
				for _, role := range required {
					if _, exists := a.Preview.Fields[role]; !exists {
						return fmt.Errorf("manifest validation: actions[%d].preview.fields missing required role %q for layout %q", i, role, a.Preview.Layout)
					}
				}
			}
		}
	}

	// Validate x-ui hints in parameters_schema (optional).
	for i, a := range m.Actions {
		if len(a.ParametersSchema) > 0 {
			if err := validateParametersSchemaUI(a.ParametersSchema, i); err != nil {
				return err
			}
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

	// Track (service, auth_type) pairs to detect duplicates. A connector may
	// declare the same service with different auth types (e.g. oauth2 + api_key).
	type serviceAuth struct{ service, authType string }
	seen := make(map[serviceAuth]bool, len(m.RequiredCredentials))
	for i, c := range m.RequiredCredentials {
		if c.Service == "" {
			return fmt.Errorf("manifest validation: required_credentials[%d].service is required", i)
		}
		if c.AuthType == "" {
			return fmt.Errorf("manifest validation: required_credentials[%d].auth_type is required", i)
		}
		if !validAuthTypes[c.AuthType] {
			return fmt.Errorf("manifest validation: required_credentials[%d].auth_type %q must be api_key, basic, custom, or oauth2", i, c.AuthType)
		}
		key := serviceAuth{c.Service, c.AuthType}
		if seen[key] {
			return fmt.Errorf("manifest validation: duplicate credential (service=%q, auth_type=%q)", c.Service, c.AuthType)
		}
		seen[key] = true
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
			field := fmt.Sprintf("manifest validation: required_credentials[%d].instructions_url", i)
			if err := validateURL(c.InstructionsURL, field, "http", "https"); err != nil {
				return err
			}
		}
	}

	// Collect all known OAuth provider IDs: built-in + declared in this manifest.
	builtInOAuthMu.Lock()
	knownProviders := make(map[string]bool, len(builtInOAuthProviders)+len(m.OAuthProviders))
	for id := range builtInOAuthProviders {
		knownProviders[id] = true
	}
	builtInOAuthMu.Unlock()

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
		if err := validateURL(p.AuthorizeURL, fmt.Sprintf("manifest validation: oauth_providers[%d].authorize_url", i), "https"); err != nil {
			return err
		}

		if p.TokenURL == "" {
			return fmt.Errorf("manifest validation: oauth_providers[%d].token_url is required", i)
		}
		if err := validateURL(p.TokenURL, fmt.Sprintf("manifest validation: oauth_providers[%d].token_url", i), "https"); err != nil {
			return err
		}

		// Reject reserved OAuth 2.0 parameters in authorize_params to prevent
		// security issues (e.g. overriding redirect_uri, state, or client_id).
		for k := range p.AuthorizeParams {
			if ReservedAuthorizeParams[k] {
				return fmt.Errorf("manifest validation: oauth_providers[%d].authorize_params contains reserved OAuth parameter %q", i, k)
			}
		}

		knownProviders[p.ID] = true
	}

	// Cross-reference: verify oauth_provider in credentials references a known provider.
	for i, c := range m.RequiredCredentials {
		if c.AuthType == "oauth2" && !knownProviders[c.OAuthProvider] {
			return fmt.Errorf("manifest validation: required_credentials[%d].oauth_provider %q is not a built-in provider and not declared in oauth_providers", i, c.OAuthProvider)
		}
	}

	return nil
}

// validateParametersSchemaUI validates x-ui rendering hints embedded in a
// parameters_schema. It checks that widget values, group references, field
// ordering, and visible_when references are all consistent and valid.
func validateParametersSchemaUI(schema json.RawMessage, actionIdx int) error {
	// Parse the schema into a generic map.
	var s map[string]json.RawMessage
	if err := json.Unmarshal(schema, &s); err != nil {
		// Not a JSON object — nothing to validate here (other validation handles this).
		return nil
	}

	// Extract property keys.
	var properties map[string]json.RawMessage
	if raw, ok := s["properties"]; ok {
		if err := json.Unmarshal(raw, &properties); err != nil {
			return nil // malformed properties — not our concern here
		}
	}
	propertyKeys := make(map[string]bool, len(properties))
	for k := range properties {
		propertyKeys[k] = true
	}

	// Extract the "required" array so we can check for visible_when + required conflicts.
	requiredFields := make(map[string]bool)
	if raw, ok := s["required"]; ok {
		var required []string
		if err := json.Unmarshal(raw, &required); err == nil {
			for _, f := range required {
				requiredFields[f] = true
			}
		}
	}

	// Parse root-level x-ui.
	var rootUI struct {
		Groups []struct {
			ID string `json:"id"`
		} `json:"groups"`
		Order []string `json:"order"`
	}
	groupIDs := make(map[string]bool)
	if raw, ok := s["x-ui"]; ok {
		if err := json.Unmarshal(raw, &rootUI); err != nil {
			return fmt.Errorf("manifest validation: actions[%d].parameters_schema x-ui is not a valid object: %w", actionIdx, err)
		}
		for _, g := range rootUI.Groups {
			if g.ID == "" {
				return fmt.Errorf("manifest validation: actions[%d].parameters_schema x-ui.groups contains entry with empty id", actionIdx)
			}
			if groupIDs[g.ID] {
				return fmt.Errorf("manifest validation: actions[%d].parameters_schema x-ui.groups contains duplicate id %q", actionIdx, g.ID)
			}
			groupIDs[g.ID] = true
		}
		// Validate x-ui.order references existing property keys with no duplicates.
		orderSeen := make(map[string]bool, len(rootUI.Order))
		for _, field := range rootUI.Order {
			if !propertyKeys[field] {
				return fmt.Errorf("manifest validation: actions[%d].parameters_schema x-ui.order references unknown property %q", actionIdx, field)
			}
			if orderSeen[field] {
				return fmt.Errorf("manifest validation: actions[%d].parameters_schema x-ui.order contains duplicate entry %q", actionIdx, field)
			}
			orderSeen[field] = true
		}
	}

	// Sort property keys for deterministic validation order. Unlike other
	// validators that iterate slices, we iterate a map — sorting ensures
	// error messages are consistent across runs.
	sortedProps := make([]string, 0, len(properties))
	for k := range properties {
		sortedProps = append(sortedProps, k)
	}
	sort.Strings(sortedProps)

	// First pass: collect visible_when targets for cycle detection.
	// Maps each property to the field it depends on for visibility.
	visibleWhenTarget := make(map[string]string) // property → field it depends on
	for _, propName := range sortedProps {
		var prop struct {
			XUI *struct {
				VisibleWhen *struct {
					Field string `json:"field"`
				} `json:"visible_when"`
			} `json:"x-ui"`
		}
		if err := json.Unmarshal(properties[propName], &prop); err != nil || prop.XUI == nil || prop.XUI.VisibleWhen == nil {
			continue
		}
		// Only record valid references — skip empty fields and self-references
		// so the cycle detector operates on semantically valid edges only.
		if vwField := prop.XUI.VisibleWhen.Field; vwField != "" && vwField != propName {
			visibleWhenTarget[propName] = vwField
		}
	}

	// Detect visible_when dependency cycles of any length (A→B→C→A).
	// Walk the chain from each node; if we revisit the start, it's a cycle.
	for _, start := range sortedProps {
		if _, ok := visibleWhenTarget[start]; !ok {
			continue
		}
		visited := map[string]bool{start: true}
		cur := visibleWhenTarget[start]
		for cur != "" {
			if cur == start {
				// Build the cycle path for a readable error message.
				path := start
				c := visibleWhenTarget[start]
				for c != start {
					path += " → " + c
					c = visibleWhenTarget[c]
				}
				path += " → " + start
				return fmt.Errorf("manifest validation: actions[%d].parameters_schema has a visible_when dependency cycle: %s", actionIdx, path)
			}
			if visited[cur] {
				break // dead end or sub-cycle not involving start
			}
			visited[cur] = true
			cur = visibleWhenTarget[cur]
		}
	}

	// Validate property-level x-ui.
	for _, propName := range sortedProps {
		propRaw := properties[propName]
		var prop struct {
			XUI *struct {
				Widget      string `json:"widget"`
				Group       string `json:"group"`
				HelpURL     string `json:"help_url"`
				VisibleWhen *struct {
					Field  string          `json:"field"`
					Equals json.RawMessage `json:"equals"`
				} `json:"visible_when"`
			} `json:"x-ui"`
		}
		if err := json.Unmarshal(propRaw, &prop); err != nil {
			continue // not our concern
		}
		if prop.XUI == nil {
			continue
		}
		if prop.XUI.Widget != "" && !validWidgets[prop.XUI.Widget] {
			return fmt.Errorf("manifest validation: actions[%d].parameters_schema.properties.%s x-ui.widget %q must be one of: text, select, remote-select, textarea, toggle, number, date, datetime, list", actionIdx, propName, prop.XUI.Widget)
		}
		// A "select" widget needs enum values to populate the dropdown.
		if prop.XUI.Widget == "select" {
			var propObj map[string]json.RawMessage
			if err := json.Unmarshal(propRaw, &propObj); err == nil {
				if _, hasEnum := propObj["enum"]; !hasEnum {
					return fmt.Errorf("manifest validation: actions[%d].parameters_schema.properties.%s x-ui.widget \"select\" requires an \"enum\" array on the property", actionIdx, propName)
				}
			}
		}
		if prop.XUI.HelpURL != "" {
			field := fmt.Sprintf("manifest validation: actions[%d].parameters_schema.properties.%s x-ui.help_url", actionIdx, propName)
			if err := validateURL(prop.XUI.HelpURL, field, "http", "https"); err != nil {
				return err
			}
		}
		if prop.XUI.Group != "" && !groupIDs[prop.XUI.Group] {
			return fmt.Errorf("manifest validation: actions[%d].parameters_schema.properties.%s x-ui.group %q does not match any defined group in x-ui.groups", actionIdx, propName, prop.XUI.Group)
		}
		if prop.XUI.VisibleWhen != nil {
			if prop.XUI.VisibleWhen.Field == "" {
				return fmt.Errorf("manifest validation: actions[%d].parameters_schema.properties.%s x-ui.visible_when requires a \"field\" key", actionIdx, propName)
			}
			if prop.XUI.VisibleWhen.Field == propName {
				return fmt.Errorf("manifest validation: actions[%d].parameters_schema.properties.%s x-ui.visible_when.field must not reference the property itself", actionIdx, propName)
			}
			if !propertyKeys[prop.XUI.VisibleWhen.Field] {
				return fmt.Errorf("manifest validation: actions[%d].parameters_schema.properties.%s x-ui.visible_when.field %q references unknown property", actionIdx, propName, prop.XUI.VisibleWhen.Field)
			}
			// len == 0 means the key was absent; null unmarshal produces []byte("null").
			if len(prop.XUI.VisibleWhen.Equals) == 0 {
				return fmt.Errorf("manifest validation: actions[%d].parameters_schema.properties.%s x-ui.visible_when requires an \"equals\" key", actionIdx, propName)
			}
			// A field with visible_when can be hidden, so it should not be in "required".
			// JSON Schema validation would reject the submission when the field is hidden.
			if requiredFields[propName] {
				return fmt.Errorf("manifest validation: actions[%d].parameters_schema.properties.%s has visible_when but is also listed in \"required\" — hidden fields cannot satisfy required validation", actionIdx, propName)
			}
		}
	}

	return nil
}

// ToDBManifest converts a ConnectorManifest into the db.ExternalConnectorManifest
// struct the DB upsert layer expects. This centralises the conversion so callers
// (main.go, cmd/seed, etc.) don't duplicate the field-mapping logic.
func (m *ConnectorManifest) ToDBManifest() db.ExternalConnectorManifest {
	status := m.Status
	if status == "" {
		status = "untested"
	}
	out := db.ExternalConnectorManifest{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		Status:      status,
		LogoSVG:     m.LogoSVG,
	}
	for _, a := range m.Actions {
		var previewJSON []byte
		if a.Preview != nil {
			var err error
			previewJSON, err = json.Marshal(a.Preview)
			if err != nil {
				log.Printf("warning: failed to marshal preview for action %s: %v", a.ActionType, err)
			}
		}
		out.Actions = append(out.Actions, db.ExternalConnectorAction{
			ActionType:            a.ActionType,
			Name:                  a.Name,
			Description:           a.Description,
			RiskLevel:             a.RiskLevel,
			ParametersSchema:      a.ParametersSchema,
			RequiresPaymentMethod: a.RequiresPaymentMethod,
			DisplayTemplate:       a.DisplayTemplate,
			Preview:               previewJSON,
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
