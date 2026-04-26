package db

import (
	"fmt"
	"strings"
)

// ValidateStaticCredentialKeys checks submitted credential keys against the
// declared schema for api_key, custom, or basic auth. Returns a non-nil error
// with a user-facing message when validation fails.
func ValidateStaticCredentialKeys(rc RequiredCredential, submitted map[string]any) error {
	switch rc.AuthType {
	case "oauth2":
		return fmt.Errorf("service %q uses OAuth — use the OAuth flow instead of storing static credentials", rc.Service)
	case "basic":
		return validateBasicCredentialKeys(submitted)
	case "api_key", "custom":
		return validateManifestCredentialKeys(rc, submitted)
	default:
		return fmt.Errorf("unsupported auth_type %q for service %q", rc.AuthType, rc.Service)
	}
}

func validateBasicCredentialKeys(submitted map[string]any) error {
	if _, ok := submitted["username"]; !ok {
		return fmt.Errorf("missing required credential key: username")
	}
	if _, ok := submitted["password"]; !ok {
		return fmt.Errorf("missing required credential key: password")
	}
	for k := range submitted {
		if k != "username" && k != "password" {
			return fmt.Errorf("unexpected credential key %q (only username and password are allowed)", k)
		}
	}
	u, okU := stringFromAny(submitted["username"])
	p, okP := stringFromAny(submitted["password"])
	if !okU || !okP {
		return fmt.Errorf("username and password must be strings")
	}
	if strings.TrimSpace(u) == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if strings.TrimSpace(p) == "" {
		return fmt.Errorf("password cannot be empty")
	}
	return nil
}

func validateManifestCredentialKeys(rc RequiredCredential, submitted map[string]any) error {
	fields := rc.CredentialFields
	if len(fields) == 0 {
		if rc.AuthType == "api_key" {
			if _, ok := submitted["api_key"]; !ok {
				return fmt.Errorf("missing required credential key: api_key")
			}
			for k := range submitted {
				if k != "api_key" {
					return fmt.Errorf("unexpected credential key %q (only api_key is allowed for this connector)", k)
				}
			}
			v, ok := stringFromAny(submitted["api_key"])
			if !ok {
				return fmt.Errorf("api_key must be a string")
			}
			if strings.TrimSpace(v) == "" {
				return fmt.Errorf("api_key cannot be empty")
			}
			return nil
		}
		// custom with no manifest fields — cannot validate server-side.
		return nil
	}

	allowed := make(map[string]CredentialFieldSpec, len(fields))
	for _, f := range fields {
		allowed[f.Key] = f
	}
	for k := range submitted {
		if _, ok := allowed[k]; !ok {
			return fmt.Errorf("unexpected credential key %q", k)
		}
	}
	for _, f := range fields {
		if !f.Required {
			continue
		}
		raw, ok := submitted[f.Key]
		if !ok {
			return fmt.Errorf("missing required credential key: %s", f.Key)
		}
		v, ok := stringFromAny(raw)
		if !ok {
			return fmt.Errorf("credential key %q must be a string", f.Key)
		}
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("%s cannot be empty", f.Key)
		}
	}
	// Optional fields: if present, must be non-empty strings (reject garbage types).
	for k, raw := range submitted {
		spec := allowed[k]
		if spec.Required {
			continue
		}
		v, ok := stringFromAny(raw)
		if !ok {
			return fmt.Errorf("credential key %q must be a string", k)
		}
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("%s cannot be empty when provided", k)
		}
	}
	return nil
}

func stringFromAny(v any) (string, bool) {
	s, ok := v.(string)
	return s, ok
}

// CredentialFieldSpecsMatch returns true if two non-empty field slices are
// identical (same keys, labels, secret, required, placeholder, help_text order).
func CredentialFieldSpecsMatch(a, b []CredentialFieldSpec) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Key != b[i].Key || a[i].Label != b[i].Label ||
			a[i].Secret != b[i].Secret || a[i].Required != b[i].Required ||
			a[i].Placeholder != b[i].Placeholder || a[i].HelpText != b[i].HelpText {
			return false
		}
	}
	return true
}
