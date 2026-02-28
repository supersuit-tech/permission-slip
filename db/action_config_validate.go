package db

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ── Wildcard & Pattern Syntax ───────────────────────────────────────────────
//
// Action configuration parameters support a wildcard/pattern syntax that
// controls what values an agent is allowed to supply:
//
//   - WildcardValue ("*"): The agent may supply any value (of any JSON type)
//     for this parameter, or omit it entirely. Stored as the JSON string "*".
//
//   - Pattern value ({"$pattern": "<glob>"}): The agent must supply a string
//     value that matches the glob pattern. The "*" character in the glob
//     matches any sequence of characters (including none).
//     Examples: {"$pattern":"*@mycompany.com"}, {"$pattern":"supersuit-tech/*"}
//     Pattern parameters are required — the agent must provide a matching value.
//
//   - Fixed value (anything without "*"): The agent must supply this exact
//     value. Permission Slip enforces an exact match via semantic JSON comparison.

// WildcardValue is the sentinel string that marks a parameter as fully agent-controlled.
// When a configuration parameter is set to "*", the agent may supply any value
// for that parameter at execution time.
const WildcardValue = "*"

// IsWildcard reports whether a JSON-encoded parameter value is the bare wildcard
// string "*". Non-string JSON values (numbers, objects, arrays, booleans, null)
// are never considered wildcards.
func IsWildcard(raw json.RawMessage) bool {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return false
	}
	return s == WildcardValue
}

// PatternKey is the JSON object key that marks a parameter value as a glob
// pattern. Pattern values are stored as {"$pattern": "<glob>"} in the
// parameters JSONB column — never as bare strings containing "*". This avoids
// ambiguity with fixed values that happen to contain the "*" character.
const PatternKey = "$pattern"

// IsPattern reports whether a JSON-encoded parameter value is a glob pattern
// wrapper: a JSON object of the form {"$pattern": "<glob>"}. The glob string
// must contain at least one "*".
//
// Plain strings containing "*" (e.g. `"*@mycompany.com"`) are NOT treated as
// patterns — they remain fixed values requiring exact match. This preserves
// backward compatibility with configurations created before pattern support.
func IsPattern(raw json.RawMessage) bool {
	p, ok := extractPattern(raw)
	return ok && strings.Contains(p, "*")
}

// ExtractPattern returns the glob string from a pattern wrapper object, or
// ("", false) if raw is not a valid pattern wrapper.
func ExtractPattern(raw json.RawMessage) (string, bool) {
	return extractPattern(raw)
}

func extractPattern(raw json.RawMessage) (string, bool) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", false
	}
	if len(obj) != 1 {
		return "", false
	}
	patternRaw, ok := obj[PatternKey]
	if !ok {
		return "", false
	}
	var s string
	if err := json.Unmarshal(patternRaw, &s); err != nil {
		return "", false
	}
	return s, true
}

// MatchPattern checks whether value matches a glob pattern where "*" matches
// any sequence of characters (including the empty string). The match is
// case-sensitive and anchored (the entire value must match).
//
// The pattern is converted to a regular expression: all regex metacharacters
// are escaped, then each "*" is replaced with ".*".
func MatchPattern(pattern, value string) bool {
	re, err := patternToRegexp(pattern)
	if err != nil {
		// Invalid pattern — treat as non-matching. This shouldn't happen
		// with well-formed patterns but guards against edge cases.
		return false
	}
	return re.MatchString(value)
}

// patternToRegexp converts a glob pattern to a compiled regexp.
// Each "*" becomes ".*", and all other regex metacharacters are escaped.
func patternToRegexp(pattern string) (*regexp.Regexp, error) {
	// Split on "*" to get the literal segments.
	parts := strings.Split(pattern, "*")
	var b strings.Builder
	b.WriteString("^")
	for i, part := range parts {
		b.WriteString(regexp.QuoteMeta(part))
		if i < len(parts)-1 {
			b.WriteString(".*")
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

// ValidateConfigParameters checks that parameter values are well-formed.
// It rejects $pattern wrapper objects that don't contain at least one "*",
// since those would be confusing (use a plain fixed value instead).
func ValidateConfigParameters(params json.RawMessage) error {
	if len(params) == 0 || string(params) == "{}" {
		return nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(params, &m); err != nil {
		return fmt.Errorf("parameters must be a JSON object")
	}
	for key, raw := range m {
		if pattern, ok := extractPattern(raw); ok {
			if !strings.Contains(pattern, "*") {
				return &ConfigValidationError{
					Parameter: key,
					Reason:    fmt.Sprintf("$pattern value %q must contain at least one '*' wildcard; use a plain string for fixed values", pattern),
				}
			}
		}
	}
	return nil
}

// ── Configuration Validation ────────────────────────────────────────────────

// ConfigValidationError describes a parameter that violates the configuration constraints.
type ConfigValidationError struct {
	Parameter string // parameter key that failed validation
	Reason    string // human-readable reason
}

func (e *ConfigValidationError) Error() string {
	return fmt.Sprintf("parameter %q: %s", e.Parameter, e.Reason)
}

// ValidateParametersAgainstConfig checks that the provided execution parameters
// satisfy the action configuration's parameter constraints:
//
//   - Wildcard parameters (value == "*") accept any value and may be omitted.
//   - Pattern parameters ({"$pattern": "<glob>"}) require a string value that
//     matches the glob pattern. Pattern parameters are required.
//   - Fixed parameters (anything else) must match exactly.
//   - Extra parameters not in the configuration are rejected.
//   - Missing parameters that are fixed or pattern in the configuration are rejected.
//   - Missing wildcard parameters are allowed (the agent chose not to provide them).
//
// Both configParams and execParams are raw JSONB (JSON objects). Returns nil if
// validation passes, or a *ConfigValidationError describing the first violation.
func ValidateParametersAgainstConfig(configParams, execParams json.RawMessage) error {
	var config map[string]json.RawMessage
	if len(configParams) == 0 || string(configParams) == "{}" {
		config = map[string]json.RawMessage{}
	} else if err := json.Unmarshal(configParams, &config); err != nil {
		return fmt.Errorf("invalid configuration parameters: %w", err)
	}

	var exec map[string]json.RawMessage
	if len(execParams) == 0 || string(execParams) == "{}" {
		exec = map[string]json.RawMessage{}
	} else if err := json.Unmarshal(execParams, &exec); err != nil {
		return fmt.Errorf("invalid execution parameters: %w", err)
	}

	// Check each configured parameter against the execution parameters.
	for key, configValue := range config {
		if IsWildcard(configValue) {
			// Bare wildcard: any value (or missing) is acceptable.
			continue
		}

		if pattern, ok := ExtractPattern(configValue); ok {
			// Pattern wrapper object: {"$pattern": "<glob>"}.
			execValue, present := exec[key]
			if !present {
				return &ConfigValidationError{
					Parameter: key,
					Reason:    "required pattern parameter is missing",
				}
			}

			// The execution value must be a string.
			var execStr string
			if err := json.Unmarshal(execValue, &execStr); err != nil {
				return &ConfigValidationError{
					Parameter: key,
					Reason:    fmt.Sprintf("value must be a string matching pattern %q", pattern),
				}
			}

			if strings.Contains(pattern, "*") {
				// Glob pattern: match against the pattern.
				if !MatchPattern(pattern, execStr) {
					return &ConfigValidationError{
						Parameter: key,
						Reason:    fmt.Sprintf("value %q does not match pattern %q", execStr, pattern),
					}
				}
			} else {
				// Malformed $pattern without "*" — treat as exact string match
				// so it doesn't silently compare against the raw JSON wrapper.
				if execStr != pattern {
					return &ConfigValidationError{
						Parameter: key,
						Reason:    fmt.Sprintf("value %q does not match pattern %q", execStr, pattern),
					}
				}
			}
			continue
		}

		// Fixed parameter: must be present and match exactly.
		execValue, present := exec[key]
		if !present {
			return &ConfigValidationError{
				Parameter: key,
				Reason:    "required fixed parameter is missing",
			}
		}

		if !jsonValuesEqual(configValue, execValue) {
			return &ConfigValidationError{
				Parameter: key,
				Reason:    "value does not match configured value",
			}
		}
	}

	// Check for extra parameters not in the configuration.
	for key := range exec {
		if _, configured := config[key]; !configured {
			return &ConfigValidationError{
				Parameter: key,
				Reason:    "parameter not defined in configuration",
			}
		}
	}

	return nil
}

// jsonValuesEqual compares two JSON values for semantic equality by
// unmarshalling to interface{} and comparing the canonical JSON encoding.
// This handles differences in whitespace, key ordering, etc.
func jsonValuesEqual(a, b json.RawMessage) bool {
	var va, vb any
	if err := json.Unmarshal(a, &va); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &vb); err != nil {
		return false
	}
	// Re-marshal to canonical form and compare bytes.
	ca, err := json.Marshal(va)
	if err != nil {
		return false
	}
	cb, err := json.Marshal(vb)
	if err != nil {
		return false
	}
	return string(ca) == string(cb)
}
