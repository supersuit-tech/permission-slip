package db

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Structured constraint format (v2) expresses standing approval rules as DNF:
// match any group (OR), each group matches all conditions (AND).
// Per-field allow/deny uses any_of/none_of/matches/does_not_match operators.

const (
	ConstraintVersionKey = "$version"
	ConstraintVersionV2  = 2
)

// Condition operators for structured constraints.
const (
	OpAnyOf        = "any_of"
	OpNoneOf       = "none_of"
	OpMatches      = "matches"
	OpDoesNotMatch = "does_not_match"
	GroupMatchAll  = "all"
	GroupMatchAny  = "any"
)

// StructuredConstraints is the v2 canonical constraint document.
type StructuredConstraints struct {
	Version int               `json:"$version"`
	Match   string            `json:"match"`
	Groups  []ConstraintGroup `json:"groups"`
}

// ConstraintGroup is one AND-scenario within a structured constraint.
type ConstraintGroup struct {
	Match      string                `json:"match"`
	Conditions []ConstraintCondition `json:"conditions"`
}

// ConstraintCondition is a single field predicate within a group.
type ConstraintCondition struct {
	Field  string            `json:"field"`
	Op     string            `json:"op"`
	Values []json.RawMessage `json:"values,omitempty"`
	Value  json.RawMessage   `json:"value,omitempty"`
}

// IsStructuredConstraintsV2 reports whether raw JSON uses the v2 structured form.
func IsStructuredConstraintsV2(raw json.RawMessage) bool {
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "{}" {
		return false
	}
	var probe struct {
		Version int `json:"$version"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return false
	}
	return probe.Version == ConstraintVersionV2
}

// ParseStructuredConstraints parses v2 JSON or adapts a legacy flat map into v2 form.
func ParseStructuredConstraints(raw json.RawMessage) (StructuredConstraints, error) {
	if IsStructuredConstraintsV2(raw) {
		var sc StructuredConstraints
		if err := json.Unmarshal(raw, &sc); err != nil {
			return StructuredConstraints{}, fmt.Errorf("invalid structured constraints: %w", err)
		}
		return sc, nil
	}
	return flatMapToStructured(raw)
}

func flatMapToStructured(raw json.RawMessage) (StructuredConstraints, error) {
	var obj map[string]json.RawMessage
	if len(raw) == 0 || string(raw) == "null" {
		obj = map[string]json.RawMessage{}
	} else if err := json.Unmarshal(raw, &obj); err != nil {
		return StructuredConstraints{}, fmt.Errorf("constraints must be a JSON object: %w", err)
	}

	conditions := make([]ConstraintCondition, 0, len(obj))
	for key, val := range obj {
		if key == MetaNamespaceKey {
			var meta map[string]json.RawMessage
			if err := json.Unmarshal(val, &meta); err != nil {
				return StructuredConstraints{}, fmt.Errorf("$meta constraints must be a JSON object: %w", err)
			}
			for metaKey, metaVal := range meta {
				conditions = append(conditions, ConstraintCondition{
					Field: MetaNamespaceKey + "." + metaKey,
					Op:    OpMatches,
					Value: metaVal,
				})
			}
			continue
		}
		conditions = append(conditions, ConstraintCondition{
			Field: key,
			Op:    OpMatches,
			Value: val,
		})
	}

	return StructuredConstraints{
		Version: ConstraintVersionV2,
		Match:   GroupMatchAny,
		Groups: []ConstraintGroup{{
			Match:      GroupMatchAll,
			Conditions: conditions,
		}},
	}, nil
}

// ValidateStructuredConstraintShape validates v2 constraint document shape and value encodings.
func ValidateStructuredConstraintShape(raw json.RawMessage) error {
	sc, err := ParseStructuredConstraints(raw)
	if err != nil {
		return err
	}
	if !IsStructuredConstraintsV2(raw) {
		// Legacy flat maps are validated by the existing flat-path validators.
		return nil
	}
	return validateStructuredConstraints(sc)
}

func validateStructuredConstraints(sc StructuredConstraints) error {
	if sc.Version != ConstraintVersionV2 {
		return fmt.Errorf("unsupported constraint version %d", sc.Version)
	}
	if sc.Match != "" && sc.Match != GroupMatchAny {
		return fmt.Errorf("top-level match must be %q", GroupMatchAny)
	}
	if len(sc.Groups) == 0 {
		return fmt.Errorf("constraints must contain at least one group")
	}
	for gi, group := range sc.Groups {
		if group.Match != "" && group.Match != GroupMatchAll {
			return fmt.Errorf("group %d match must be %q", gi, GroupMatchAll)
		}
		if len(group.Conditions) == 0 {
			return fmt.Errorf("group %d must contain at least one condition", gi)
		}
		for ci, cond := range group.Conditions {
			if err := validateStructuredCondition(gi, ci, cond); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateStructuredCondition(gi, ci int, cond ConstraintCondition) error {
	param := fmt.Sprintf("groups[%d].conditions[%d]", gi, ci)
	if strings.TrimSpace(cond.Field) == "" {
		return fmt.Errorf("%s: field must not be empty", param)
	}
	switch cond.Op {
	case OpAnyOf, OpNoneOf:
		if len(cond.Values) == 0 {
			return fmt.Errorf("%s: %s requires at least one value", param, cond.Op)
		}
		if len(cond.Value) > 0 {
			return fmt.Errorf("%s: %s must use values, not value", param, cond.Op)
		}
		for vi, val := range cond.Values {
			if err := validateConstraintValueEncoding(cond.Field, val); err != nil {
				return fmt.Errorf("%s.values[%d]: %w", param, vi, err)
			}
		}
	case OpMatches, OpDoesNotMatch:
		if len(cond.Value) == 0 || string(cond.Value) == "null" {
			return fmt.Errorf("%s: %s requires a value", param, cond.Op)
		}
		if len(cond.Values) > 0 {
			return fmt.Errorf("%s: %s must use value, not values", param, cond.Op)
		}
		if err := validateConstraintValueEncoding(cond.Field, cond.Value); err != nil {
			return fmt.Errorf("%s: %w", param, err)
		}
	default:
		return fmt.Errorf("%s: unsupported op %q", param, cond.Op)
	}
	return nil
}

func validateConstraintValueEncoding(field string, val json.RawMessage) error {
	if field == DataWindowNamespaceKey {
		return ValidateDataWindowConstraintShape(val)
	}
	if pattern, ok := extractPattern(val); ok {
		if !strings.Contains(pattern, "*") {
			return fmt.Errorf("$pattern value %q must contain at least one '*' wildcard; use a plain string for fixed values", pattern)
		}
	}
	return nil
}

// StructuredConstraintsHasNonWildcard reports whether any condition is not a bare wildcard.
func StructuredConstraintsHasNonWildcard(sc StructuredConstraints) bool {
	for _, group := range sc.Groups {
		for _, cond := range group.Conditions {
			if cond.Field == DataWindowNamespaceKey {
				return true
			}
			for _, val := range conditionValues(cond) {
				if !IsWildcard(val) {
					return true
				}
			}
		}
	}
	return false
}

func conditionValues(cond ConstraintCondition) []json.RawMessage {
	switch cond.Op {
	case OpAnyOf, OpNoneOf:
		return cond.Values
	case OpMatches, OpDoesNotMatch:
		if len(cond.Value) > 0 {
			return []json.RawMessage{cond.Value}
		}
	}
	return nil
}

func isAllowOp(op string) bool {
	return op == OpAnyOf || op == OpMatches
}

func isDenyOp(op string) bool {
	return op == OpNoneOf || op == OpDoesNotMatch
}

// ValidateParametersAgainstStructuredConfig evaluates structured constraints against exec params.
func ValidateParametersAgainstStructuredConfig(sc StructuredConstraints, execParams, resolvedMeta json.RawMessage) error {
	var exec map[string]json.RawMessage
	if len(execParams) == 0 || string(execParams) == "null" || string(execParams) == "{}" {
		exec = map[string]json.RawMessage{}
	} else if err := json.Unmarshal(execParams, &exec); err != nil {
		return fmt.Errorf("invalid execution parameters: %w", err)
	}

	var meta map[string]json.RawMessage
	hasMeta := len(resolvedMeta) > 0 && string(resolvedMeta) != "null"
	if hasMeta {
		if err := json.Unmarshal(resolvedMeta, &meta); err != nil {
			return fmt.Errorf("invalid resolved metadata: %w", err)
		}
	}

	var lastErr error
	for _, group := range sc.Groups {
		matched, err := evaluateConstraintGroup(group, exec, meta, hasMeta)
		if err != nil {
			return err
		}
		if matched {
			return nil
		}
		if lastErr == nil {
			lastErr = &ConfigValidationError{
				Parameter: "constraints",
				Reason:    "no matching scenario",
			}
		}
	}
	if lastErr != nil {
		return lastErr
	}
	return &ConfigValidationError{
		Parameter: "constraints",
		Reason:    "no matching scenario",
	}
}

func evaluateConstraintGroup(group ConstraintGroup, exec map[string]json.RawMessage, meta map[string]json.RawMessage, hasMeta bool) (matched bool, err error) {
	byField := groupConditionsByField(group.Conditions)

	for field, conds := range byField {
		if field == DataWindowNamespaceKey {
			continue
		}
		if strings.HasPrefix(field, MetaNamespaceKey+".") {
			if !hasMeta {
				return false, ErrMetadataUnresolved
			}
			metaKey := strings.TrimPrefix(field, MetaNamespaceKey+".")
			if err := evaluateFieldConditions(field, metaKey, conds, meta, true); err != nil {
				return false, nil
			}
			continue
		}
		if err := evaluateExecFieldConditions(field, conds, exec); err != nil {
			return false, nil
		}
	}

	// Reject extra exec params not declared in this group's param conditions.
	configured := make(map[string]struct{})
	for field := range byField {
		if field == DataWindowNamespaceKey || strings.HasPrefix(field, MetaNamespaceKey+".") {
			continue
		}
		configured[field] = struct{}{}
	}
	for key := range exec {
		if _, ok := configured[key]; !ok {
			return false, &ConfigValidationError{
				Parameter: key,
				Reason:    "parameter not defined in configuration",
			}
		}
	}

	return true, nil
}

func groupConditionsByField(conditions []ConstraintCondition) map[string][]ConstraintCondition {
	out := make(map[string][]ConstraintCondition)
	for _, cond := range conditions {
		out[cond.Field] = append(out[cond.Field], cond)
	}
	return out
}

func evaluateExecFieldConditions(field string, conds []ConstraintCondition, exec map[string]json.RawMessage) error {
	allow, deny := splitAllowDenyValues(conds)
	sourceValue, present := exec[field]

	if len(allow) == 0 && len(deny) == 0 {
		return nil
	}

	if hasWildcardAllow(allow) {
		return evaluateDenyOnly(field, deny, sourceValue, present, false)
	}

	if !present {
		if len(allow) > 0 {
			return &ConfigValidationError{
				Parameter: field,
				Reason:    "required parameter is missing",
			}
		}
		return evaluateDenyOnly(field, deny, sourceValue, present, false)
	}

	if len(allow) > 0 {
		if !valueMatchesAnyAllow(field, allow, sourceValue, false) {
			return &ConfigValidationError{
				Parameter: field,
				Reason:    "value does not match allowed constraints",
			}
		}
	}

	return evaluateDenyList(field, deny, sourceValue, false)
}

func evaluateFieldConditions(param, metaKey string, conds []ConstraintCondition, meta map[string]json.RawMessage, isMeta bool) error {
	allow, deny := splitAllowDenyValues(conds)

	if messagesRaw, ok := meta["messages"]; ok && len(messagesRaw) > 0 && string(messagesRaw) != "null" {
		var messages []map[string]json.RawMessage
		if err := json.Unmarshal(messagesRaw, &messages); err != nil {
			return fmt.Errorf("invalid resolved messages metadata: %w", err)
		}
		if len(messages) == 0 {
			return ErrMetadataUnresolved
		}
		for _, msg := range messages {
			sourceValue := perMessageMetaSourceValue(metaKey, msg)
			if err := evaluateMetaSource(param, metaKey, allow, deny, sourceValue, isMeta); err != nil {
				return err
			}
		}
		return nil
	}

	sourceValue := flatMetaSourceValue(metaKey, meta)
	return evaluateMetaSource(param, metaKey, allow, deny, sourceValue, isMeta)
}

func evaluateMetaSource(param, metaKey string, allow, deny []json.RawMessage, sourceValue json.RawMessage, isMeta bool) error {
	if hasWildcardAllow(allow) {
		return evaluateDenyOnly(param, deny, sourceValue, len(sourceValue) > 0, isMeta && isRecipientMetaKey(metaKey))
	}

	if len(allow) > 0 {
		if !valueMatchesAnyAllow(param, allow, sourceValue, isMeta && isRecipientMetaKey(metaKey)) {
			return &ConfigValidationError{
				Parameter: param,
				Reason:    "metadata value does not match allowed constraints",
			}
		}
	}

	return evaluateDenyList(param, deny, sourceValue, isMeta && isRecipientMetaKey(metaKey))
}

func splitAllowDenyValues(conds []ConstraintCondition) (allow, deny []json.RawMessage) {
	for _, cond := range conds {
		vals := conditionValues(cond)
		if isAllowOp(cond.Op) {
			allow = append(allow, vals...)
		} else if isDenyOp(cond.Op) {
			deny = append(deny, vals...)
		}
	}
	return allow, deny
}

func hasWildcardAllow(allow []json.RawMessage) bool {
	for _, val := range allow {
		if IsWildcard(val) {
			return true
		}
	}
	return false
}

func evaluateDenyOnly(param string, deny []json.RawMessage, sourceValue json.RawMessage, present, recipientMulti bool) error {
	if len(deny) == 0 {
		return nil
	}
	if !present {
		return nil
	}
	return evaluateDenyList(param, deny, sourceValue, recipientMulti)
}

func valueMatchesAnyAllow(param string, allow []json.RawMessage, sourceValue json.RawMessage, recipientMulti bool) bool {
	for _, allowVal := range allow {
		if IsWildcard(allowVal) {
			return true
		}
		if recipientMulti {
			if err := validateRecipientMetaConstraint(param, allowVal, sourceValue); err == nil {
				return true
			}
			continue
		}
		if err := validateConstraintValueAgainstSource(param, allowVal, sourceValue); err == nil {
			return true
		}
	}
	return false
}

func evaluateDenyList(param string, deny []json.RawMessage, sourceValue json.RawMessage, recipientMulti bool) error {
	for _, denyVal := range deny {
		if IsWildcard(denyVal) {
			continue
		}
		if recipientMulti {
			if err := validateRecipientMetaConstraint(param, denyVal, sourceValue); err == nil {
				return &ConfigValidationError{
					Parameter: param,
					Reason:    "metadata value matches a denied constraint",
				}
			}
			continue
		}
		if err := validateConstraintValueAgainstSource(param, denyVal, sourceValue); err == nil {
			return &ConfigValidationError{
				Parameter: param,
				Reason:    "value matches a denied constraint",
			}
		}
	}
	return nil
}

// NormalizeStructuredConstraints wraps bare *-strings as $pattern and returns canonical JSON bytes.
func NormalizeStructuredConstraints(raw json.RawMessage) ([]byte, error) {
	if !IsStructuredConstraintsV2(raw) {
		return raw, nil
	}
	var sc StructuredConstraints
	if err := json.Unmarshal(raw, &sc); err != nil {
		return nil, err
	}
	mutated := false
	for gi := range sc.Groups {
		for ci := range sc.Groups[gi].Conditions {
			cond := &sc.Groups[gi].Conditions[ci]
			switch cond.Op {
			case OpAnyOf, OpNoneOf:
				for vi := range cond.Values {
					normalized, changed, err := normalizeConstraintValue(cond.Values[vi])
					if err != nil {
						return nil, err
					}
					if changed {
						cond.Values[vi] = normalized
						mutated = true
					}
				}
			case OpMatches, OpDoesNotMatch:
				normalized, changed, err := normalizeConstraintValue(cond.Value)
				if err != nil {
					return nil, err
				}
				if changed {
					cond.Value = normalized
					mutated = true
				}
			}
		}
	}
	if !mutated {
		return raw, nil
	}
	return json.Marshal(sc)
}

func normalizeConstraintValue(val json.RawMessage) (json.RawMessage, bool, error) {
	if IsWildcard(val) {
		return val, false, nil
	}
	if _, ok := extractPattern(val); ok {
		return val, false, nil
	}
	var s string
	if json.Unmarshal(val, &s) != nil {
		return val, false, nil
	}
	if s == WildcardValue {
		return val, false, nil
	}
	if strings.Contains(s, "*") {
		wrapped, err := json.Marshal(map[string]string{PatternKey: s})
		if err != nil {
			return nil, false, err
		}
		return wrapped, true, nil
	}
	return val, false, nil
}

// MarshalStructuredConstraints serializes structured constraints to JSON.
func MarshalStructuredConstraints(sc StructuredConstraints) ([]byte, error) {
	sc.Version = ConstraintVersionV2
	if sc.Match == "" {
		sc.Match = GroupMatchAny
	}
	for i := range sc.Groups {
		if sc.Groups[i].Match == "" {
			sc.Groups[i].Match = GroupMatchAll
		}
	}
	return json.Marshal(sc)
}

// StructuredConstraintsConfiguredParamFields returns schema param fields referenced in constraints.
func StructuredConstraintsConfiguredParamFields(sc StructuredConstraints) map[string]struct{} {
	out := make(map[string]struct{})
	for _, group := range sc.Groups {
		for _, cond := range group.Conditions {
			if cond.Field == DataWindowNamespaceKey || strings.HasPrefix(cond.Field, MetaNamespaceKey+".") {
				continue
			}
			out[cond.Field] = struct{}{}
		}
	}
	return out
}

// StructuredConstraintsHasMeta reports whether any condition references $meta fields.
func StructuredConstraintsHasMeta(sc StructuredConstraints) bool {
	for _, group := range sc.Groups {
		for _, cond := range group.Conditions {
			if strings.HasPrefix(cond.Field, MetaNamespaceKey+".") {
				return true
			}
		}
	}
	return false
}

// AddMissingSchemaFieldsToStructured adds wildcard matches conditions for absent schema keys.
func AddMissingSchemaFieldsToStructured(sc StructuredConstraints, schemaKeys map[string]struct{}) StructuredConstraints {
	if len(schemaKeys) == 0 || len(sc.Groups) == 0 {
		return sc
	}
	configured := StructuredConstraintsConfiguredParamFields(sc)
	wildcardBytes := json.RawMessage(`"*"`)
	for gi := range sc.Groups {
		for key := range schemaKeys {
			if _, ok := configured[key]; ok {
				continue
			}
			sc.Groups[gi].Conditions = append(sc.Groups[gi].Conditions, ConstraintCondition{
				Field: key,
				Op:    OpMatches,
				Value: wildcardBytes,
			})
		}
	}
	return sc
}
