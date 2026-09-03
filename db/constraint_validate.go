package db

import (
	"encoding/json"
	"errors"
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

// IsSemanticWildcard reports whether a JSON-encoded parameter value matches every
// possible string: the bare wildcard "*" or a $pattern glob made entirely of
// "*" characters ("*", "**", "***", …). Patterns with any non-star character
// (e.g. "a*" or "*@example.com") are not semantic wildcards.
func IsSemanticWildcard(raw json.RawMessage) bool {
	if IsWildcard(raw) {
		return true
	}
	pattern, ok := extractPattern(raw)
	if !ok {
		return false
	}
	return isAllStarGlob(pattern)
}

func isAllStarGlob(pattern string) bool {
	if pattern == "" {
		return false
	}
	for _, r := range pattern {
		if r != '*' {
			return false
		}
	}
	return true
}

// PatternKey is the JSON object key that marks a parameter value as a glob
// pattern. Pattern values are stored as {"$pattern": "<glob>"} in the
// parameters JSONB column — never as bare strings containing "*". This avoids
// ambiguity with fixed values that happen to contain the "*" character.
const PatternKey = "$pattern"

// MetaNamespaceKey is the reserved configuration key for constraints on
// server-resolved metadata (e.g. verified email sender). Nested fields use the
// same wildcard/pattern/fixed syntax as regular parameters.
const MetaNamespaceKey = "$meta"

// ErrMetadataUnresolved indicates $meta constraints are present but verified
// metadata was not supplied to the constraint engine.
var ErrMetadataUnresolved = errors.New("constraint metadata unresolved")

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
		if key == MetaNamespaceKey {
			if err := validateMetaNamespaceConfig(raw); err != nil {
				return err
			}
			continue
		}
		if key == DataWindowNamespaceKey {
			if err := ValidateDataWindowConstraintShape(raw); err != nil {
				return err
			}
			continue
		}
		if token, ok := ExtractRelativeDateToken(raw); ok {
			if err := ValidateRelativeDateToken(token); err != nil {
				return err
			}
			continue
		}
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

func validateMetaNamespaceConfig(raw json.RawMessage) error {
	if len(raw) == 0 || string(raw) == "null" {
		return &ConfigValidationError{
			Parameter: MetaNamespaceKey,
			Reason:    "must be a JSON object",
		}
	}
	var meta map[string]json.RawMessage
	if err := json.Unmarshal(raw, &meta); err != nil {
		return &ConfigValidationError{
			Parameter: MetaNamespaceKey,
			Reason:    "must be a JSON object",
		}
	}
	if len(meta) == 0 {
		return &ConfigValidationError{
			Parameter: MetaNamespaceKey,
			Reason:    "must contain at least one metadata constraint",
		}
	}
	for key, value := range meta {
		param := MetaNamespaceKey + "." + key
		if pattern, ok := extractPattern(value); ok {
			if !strings.Contains(pattern, "*") {
				return &ConfigValidationError{
					Parameter: param,
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
// satisfy the action configuration's parameter constraints.
func ValidateParametersAgainstConfig(configParams, execParams, resolvedMeta json.RawMessage) error {
	if IsStructuredConstraintsV2(configParams) {
		sc, err := ParseStructuredConstraints(configParams)
		if err != nil {
			return err
		}
		return ValidateParametersAgainstStructuredConfig(sc, execParams, resolvedMeta)
	}
	return validateFlatParametersAgainstConfig(configParams, execParams, resolvedMeta)
}

func validateFlatParametersAgainstConfig(configParams, execParams, resolvedMeta json.RawMessage) error {
	var config map[string]json.RawMessage
	if len(configParams) == 0 || string(configParams) == "{}" {
		config = map[string]json.RawMessage{}
	} else if err := json.Unmarshal(configParams, &config); err != nil {
		return fmt.Errorf("invalid configuration parameters: %w", err)
	}

	metaConstraints, hasMeta := config[MetaNamespaceKey]
	if hasMeta {
		delete(config, MetaNamespaceKey)
	}
	if _, hasDW := config[DataWindowNamespaceKey]; hasDW {
		delete(config, DataWindowNamespaceKey)
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

		if token, ok := ExtractRelativeDateToken(configValue); ok {
			_ = token
			// Relative date bounds are enforced when params are injected/clamped
			// after a standing approval match (see ApplyRelativeDateConstraintsToParams).
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

			if err := validateExecPatternConstraint(key, pattern, execValue); err != nil {
				return err
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

	if hasMeta {
		if err := validateResolvedMetaConstraints(metaConstraints, resolvedMeta); err != nil {
			return err
		}
	}

	return nil
}

func validateResolvedMetaConstraints(metaConstraints, resolvedMeta json.RawMessage) error {
	var constraints map[string]json.RawMessage
	if err := json.Unmarshal(metaConstraints, &constraints); err != nil {
		return fmt.Errorf("invalid $meta constraints: %w", err)
	}
	if len(constraints) == 0 {
		return nil
	}
	if len(resolvedMeta) == 0 || string(resolvedMeta) == "null" {
		return ErrMetadataUnresolved
	}

	var meta map[string]json.RawMessage
	if err := json.Unmarshal(resolvedMeta, &meta); err != nil {
		return fmt.Errorf("invalid resolved metadata: %w", err)
	}

	if messagesRaw, ok := meta["messages"]; ok && len(messagesRaw) > 0 && string(messagesRaw) != "null" {
		var messages []map[string]json.RawMessage
		if err := json.Unmarshal(messagesRaw, &messages); err != nil {
			return fmt.Errorf("invalid resolved messages metadata: %w", err)
		}
		return validatePerMessageMetaConstraints(constraints, messages)
	}

	for key, configValue := range constraints {
		param := MetaNamespaceKey + "." + key
		sourceValue := flatMetaSourceValue(key, meta)
		if err := validateMetaFieldConstraint(param, key, configValue, sourceValue); err != nil {
			return err
		}
	}
	return nil
}

func validatePerMessageMetaConstraints(constraints map[string]json.RawMessage, messages []map[string]json.RawMessage) error {
	if len(messages) == 0 {
		return ErrMetadataUnresolved
	}
	for _, msg := range messages {
		for key, configValue := range constraints {
			param := MetaNamespaceKey + "." + key
			sourceValue := perMessageMetaSourceValue(key, msg)
			if err := validateMetaFieldConstraint(param, key, configValue, sourceValue); err != nil {
				return err
			}
		}
	}
	return nil
}

func flatMetaSourceValue(key string, meta map[string]json.RawMessage) json.RawMessage {
	switch key {
	case "sender":
		if v := meta["sender"]; len(v) > 0 {
			return v
		}
		return meta["senders"]
	case "from":
		if v := meta["from"]; len(v) > 0 {
			return v
		}
		return meta["sender"]
	default:
		return meta[key]
	}
}

func perMessageMetaSourceValue(key string, msg map[string]json.RawMessage) json.RawMessage {
	switch key {
	case "sender", "senders":
		return msg["from"]
	case "from":
		return msg["from"]
	default:
		return msg[key]
	}
}

func isRecipientMetaKey(key string) bool {
	return key == "to" || key == "cc" || key == "bcc"
}

func validateMetaFieldConstraint(param, key string, configValue, sourceValue json.RawMessage) error {
	if isRecipientMetaKey(key) {
		return validateRecipientMetaConstraint(param, configValue, sourceValue)
	}
	return validateConstraintValueAgainstSource(param, configValue, sourceValue)
}

func validateExecPatternConstraint(param, pattern string, execValue json.RawMessage) error {
	var values []string
	if err := json.Unmarshal(execValue, &values); err == nil {
		if len(values) == 0 {
			return &ConfigValidationError{
				Parameter: param,
				Reason:    fmt.Sprintf("value must be a non-empty string or array of strings matching pattern %q", pattern),
			}
		}
		for _, value := range values {
			if err := validateStringPattern(param, pattern, value); err != nil {
				return err
			}
		}
		return nil
	}

	var execStr string
	if err := json.Unmarshal(execValue, &execStr); err != nil {
		return &ConfigValidationError{
			Parameter: param,
			Reason:    fmt.Sprintf("value must be a string or array of strings matching pattern %q", pattern),
		}
	}
	return validateStringPattern(param, pattern, execStr)
}

func validateConstraintValueAgainstSource(param string, configValue, sourceValue json.RawMessage) error {
	if IsWildcard(configValue) {
		return nil
	}

	if pattern, ok := ExtractPattern(configValue); ok {
		return validatePatternConstraint(param, pattern, sourceValue)
	}

	if len(sourceValue) == 0 {
		return &ConfigValidationError{
			Parameter: param,
			Reason:    "required metadata field is missing",
		}
	}
	if !jsonValuesEqual(configValue, sourceValue) {
		return &ConfigValidationError{
			Parameter: param,
			Reason:    "metadata value does not match configured value",
		}
	}
	return nil
}

func validatePatternConstraint(param, pattern string, sourceValue json.RawMessage) error {
	if len(sourceValue) == 0 {
		return &ConfigValidationError{
			Parameter: param,
			Reason:    fmt.Sprintf("required metadata field is missing for pattern %q", pattern),
		}
	}

	var values []string
	if err := json.Unmarshal(sourceValue, &values); err == nil {
		for _, value := range values {
			if err := validateStringPattern(param, pattern, value); err != nil {
				return err
			}
		}
		return nil
	}

	var value string
	if err := json.Unmarshal(sourceValue, &value); err != nil {
		return &ConfigValidationError{
			Parameter: param,
			Reason:    fmt.Sprintf("metadata value must be a string or array of strings matching pattern %q", pattern),
		}
	}
	return validateStringPattern(param, pattern, value)
}

func validateRecipientMetaConstraint(param string, configValue, sourceValue json.RawMessage) error {
	if IsWildcard(configValue) {
		return nil
	}

	if pattern, ok := ExtractPattern(configValue); ok {
		return validateRecipientPatternConstraint(param, pattern, sourceValue)
	}

	if len(sourceValue) == 0 {
		return &ConfigValidationError{
			Parameter: param,
			Reason:    "required metadata field is missing",
		}
	}

	var values []string
	if err := json.Unmarshal(sourceValue, &values); err != nil {
		return &ConfigValidationError{
			Parameter: param,
			Reason:    "metadata value must be an array of recipient addresses",
		}
	}
	if len(values) == 0 {
		return &ConfigValidationError{
			Parameter: param,
			Reason:    "required metadata field is missing",
		}
	}

	var expected string
	if err := json.Unmarshal(configValue, &expected); err != nil {
		return fmt.Errorf("invalid recipient metadata constraint for %s: %w", param, err)
	}
	for _, value := range values {
		if value == expected {
			return nil
		}
	}
	return &ConfigValidationError{
		Parameter: param,
		Reason:    "metadata value does not match configured value",
	}
}

func validateRecipientPatternConstraint(param, pattern string, sourceValue json.RawMessage) error {
	if len(sourceValue) == 0 {
		return &ConfigValidationError{
			Parameter: param,
			Reason:    fmt.Sprintf("required metadata field is missing for pattern %q", pattern),
		}
	}

	var values []string
	if err := json.Unmarshal(sourceValue, &values); err != nil {
		return &ConfigValidationError{
			Parameter: param,
			Reason:    fmt.Sprintf("metadata value must be an array of strings matching pattern %q", pattern),
		}
	}
	if len(values) == 0 {
		return &ConfigValidationError{
			Parameter: param,
			Reason:    fmt.Sprintf("required metadata field is missing for pattern %q", pattern),
		}
	}

	for _, value := range values {
		if err := validateStringPattern(param, pattern, value); err == nil {
			return nil
		}
	}
	return &ConfigValidationError{
		Parameter: param,
		Reason:    fmt.Sprintf("no recipient address matches pattern %q", pattern),
	}
}

func validateStringPattern(param, pattern, value string) error {
	if strings.Contains(pattern, "*") {
		if !MatchPattern(pattern, value) {
			return &ConfigValidationError{
				Parameter: param,
				Reason:    fmt.Sprintf("metadata value %q does not match pattern %q", value, pattern),
			}
		}
		return nil
	}
	if value != pattern {
		return &ConfigValidationError{
			Parameter: param,
			Reason:    fmt.Sprintf("metadata value %q does not match pattern %q", value, pattern),
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
