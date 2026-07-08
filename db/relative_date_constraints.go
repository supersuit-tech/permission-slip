package db

import (
	"encoding/json"
	"fmt"
	"time"
)

// CollectRelativeDateConstraintTokens returns parameter fields whose constraint
// values are relative date tokens.
func CollectRelativeDateConstraintTokens(constraints json.RawMessage) map[string]string {
	if len(constraints) == 0 || string(constraints) == "null" {
		return nil
	}
	if IsStructuredConstraintsV2(constraints) {
		sc, err := ParseStructuredConstraints(constraints)
		if err != nil {
			return nil
		}
		out := make(map[string]string)
		for _, group := range sc.Groups {
			for _, cond := range group.Conditions {
				if cond.Field == MetaNamespaceKey || cond.Field == DataWindowNamespaceKey ||
					stringsHasPrefix(cond.Field, MetaNamespaceKey+".") {
					continue
				}
				for _, val := range conditionValues(cond) {
					if token, ok := ExtractRelativeDateToken(val); ok {
						if isAllowOp(cond.Op) {
							out[cond.Field] = token
						}
					}
				}
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(constraints, &obj); err != nil {
		return nil
	}
	out := make(map[string]string)
	for key, val := range obj {
		if key == MetaNamespaceKey || key == DataWindowNamespaceKey {
			continue
		}
		if token, ok := ExtractRelativeDateToken(val); ok {
			out[key] = token
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// ApplyRelativeDateConstraintsToParams injects or clamps execution parameters
// for fields constrained by relative date tokens.
func ApplyRelativeDateConstraintsToParams(
	params json.RawMessage,
	constraints json.RawMessage,
	fields map[string]DateTimeFieldInfo,
	now time.Time,
	loc *time.Location,
) (json.RawMessage, error) {
	tokens := CollectRelativeDateConstraintTokens(constraints)
	if len(tokens) == 0 {
		return params, nil
	}

	var m map[string]json.RawMessage
	if len(params) == 0 || string(params) == "{}" {
		m = map[string]json.RawMessage{}
	} else if err := json.Unmarshal(params, &m); err != nil {
		return nil, fmt.Errorf("invalid execution parameters: %w", err)
	}

	for field, token := range tokens {
		info := fields[field]
		format := info.Format
		if format == "" {
			format = "date-time"
		}
		role := RelativeDateBoundRoleForToken(token, info.Role)

		bound, err := ResolveRelativeDateToken(token, now, loc)
		if err != nil {
			return nil, err
		}

		effective := bound
		if raw, ok := m[field]; ok {
			if execTime, ok, parseErr := parseTimeParamFlexible(raw, format); parseErr != nil {
				return nil, parseErr
			} else if ok {
				switch role {
				case RelativeDateBoundUpper:
					if execTime.Before(effective) {
						effective = execTime
					}
				case RelativeDateBoundExact:
					effective = bound
				default:
					if execTime.After(effective) {
						effective = execTime
					}
				}
			}
		}

		encoded, err := marshalRelativeDateValue(effective, format)
		if err != nil {
			return nil, err
		}
		m[field] = encoded
	}

	out, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func parseTimeParamFlexible(raw json.RawMessage, format string) (time.Time, bool, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return time.Time{}, false, &ConfigValidationError{
			Parameter: "datetime",
			Reason:    "relative date parameter must be a timestamp string",
		}
	}
	if s == "" {
		return time.Time{}, false, nil
	}
	if format == "date" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return time.Time{}, false, &ConfigValidationError{
				Parameter: "datetime",
				Reason:    "relative date parameter must be a YYYY-MM-DD date string",
			}
		}
		return t.UTC(), true, nil
	}
	t, err := parseRFC3339(s)
	if err != nil {
		return time.Time{}, false, &ConfigValidationError{
			Parameter: "datetime",
			Reason:    "relative date parameter must be an RFC3339 timestamp string",
		}
	}
	return t, true, nil
}
