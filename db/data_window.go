package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"
)

// DataWindowNamespaceKey is the reserved standing-approval constraint key for
// time-scoped read grants. Nested fields: starts_at, ends_at (RFC3339), last_days.
const DataWindowNamespaceKey = "$data_window"

// DataWindowParams names connector execution parameters used as the window pair.
type DataWindowParams struct {
	StartParam string `json:"start_param"`
	EndParam   string `json:"end_param"`
}

// ResolvedDataWindow is an absolute time range derived from a $data_window constraint.
type ResolvedDataWindow struct {
	StartsAt *time.Time
	EndsAt   *time.Time
}

// ErrDataWindowUnsupported indicates $data_window is present but the action
// does not declare a data window parameter pair.
var ErrDataWindowUnsupported = errors.New("data window not supported for action")

// ValidateDataWindowConstraintShape checks $data_window object shape at write time.
func ValidateDataWindowConstraintShape(raw json.RawMessage) error {
	_, err := parseDataWindowConstraint(raw)
	return err
}

func parseDataWindowConstraint(raw json.RawMessage) (map[string]json.RawMessage, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, &ConfigValidationError{
			Parameter: DataWindowNamespaceKey,
			Reason:    "must be a JSON object",
		}
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, &ConfigValidationError{
			Parameter: DataWindowNamespaceKey,
			Reason:    "must be a JSON object",
		}
	}
	if len(obj) == 0 {
		return nil, &ConfigValidationError{
			Parameter: DataWindowNamespaceKey,
			Reason:    "must contain at least one of starts_at, ends_at, or last_days",
		}
	}

	var hasStartsAt, hasEndsAt, hasLastDays bool
	var startsAt, endsAt time.Time

	for key, val := range obj {
		switch key {
		case "starts_at":
			hasStartsAt = true
			var s string
			if err := json.Unmarshal(val, &s); err != nil || s == "" {
				return nil, &ConfigValidationError{
					Parameter: DataWindowNamespaceKey + ".starts_at",
					Reason:    "must be an RFC3339 timestamp string",
				}
			}
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				if t, err = time.Parse(time.RFC3339Nano, s); err != nil {
					return nil, &ConfigValidationError{
						Parameter: DataWindowNamespaceKey + ".starts_at",
						Reason:    "must be an RFC3339 timestamp string",
					}
				}
			}
			startsAt = t.UTC()
		case "ends_at":
			hasEndsAt = true
			var s string
			if err := json.Unmarshal(val, &s); err != nil || s == "" {
				return nil, &ConfigValidationError{
					Parameter: DataWindowNamespaceKey + ".ends_at",
					Reason:    "must be an RFC3339 timestamp string",
				}
			}
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				if t, err = time.Parse(time.RFC3339Nano, s); err != nil {
					return nil, &ConfigValidationError{
						Parameter: DataWindowNamespaceKey + ".ends_at",
						Reason:    "must be an RFC3339 timestamp string",
					}
				}
			}
			endsAt = t.UTC()
		case "last_days":
			hasLastDays = true
			var n float64
			if err := json.Unmarshal(val, &n); err != nil || n != math.Trunc(n) {
				return nil, &ConfigValidationError{
					Parameter: DataWindowNamespaceKey + ".last_days",
					Reason:    "must be an integer >= 1",
				}
			}
			days := int(n)
			if days < 1 {
				return nil, &ConfigValidationError{
					Parameter: DataWindowNamespaceKey + ".last_days",
					Reason:    "must be an integer >= 1",
				}
			}
		default:
			return nil, &ConfigValidationError{
				Parameter: DataWindowNamespaceKey + "." + key,
				Reason:    "unknown field",
			}
		}
	}

	if !hasStartsAt && !hasEndsAt && !hasLastDays {
		return nil, &ConfigValidationError{
			Parameter: DataWindowNamespaceKey,
			Reason:    "must contain at least one of starts_at, ends_at, or last_days",
		}
	}
	if hasLastDays && hasStartsAt {
		return nil, &ConfigValidationError{
			Parameter: DataWindowNamespaceKey,
			Reason:    "last_days is mutually exclusive with starts_at",
		}
	}
	if hasStartsAt && hasEndsAt && !endsAt.After(startsAt) {
		return nil, &ConfigValidationError{
			Parameter: DataWindowNamespaceKey,
			Reason:    "ends_at must be after starts_at",
		}
	}

	return obj, nil
}

// ResolveDataWindowConstraint converts a validated $data_window object into an
// absolute window relative to now (for last_days).
func ResolveDataWindowConstraint(raw json.RawMessage, now time.Time) (ResolvedDataWindow, error) {
	obj, err := parseDataWindowConstraint(raw)
	if err != nil {
		return ResolvedDataWindow{}, err
	}

	var out ResolvedDataWindow
	now = now.UTC()

	if lastDaysRaw, ok := obj["last_days"]; ok {
		var n float64
		_ = json.Unmarshal(lastDaysRaw, &n)
		days := int(n)
		start := now.AddDate(0, 0, -days)
		out.StartsAt = &start
	}

	if startsRaw, ok := obj["starts_at"]; ok {
		var s string
		_ = json.Unmarshal(startsRaw, &s)
		t, _ := parseRFC3339(s)
		out.StartsAt = &t
	}
	if endsRaw, ok := obj["ends_at"]; ok {
		var s string
		_ = json.Unmarshal(endsRaw, &s)
		t, _ := parseRFC3339(s)
		out.EndsAt = &t
	}

	return out, nil
}

// ApplyDataWindowToParams injects or clamps the declared window parameters on
// execution params. Never widens beyond the resolved approval window.
func ApplyDataWindowToParams(
	params json.RawMessage,
	window ResolvedDataWindow,
	startParam, endParam string,
) (json.RawMessage, error) {
	if startParam == "" || endParam == "" {
		return nil, ErrDataWindowUnsupported
	}

	var m map[string]json.RawMessage
	if len(params) == 0 || string(params) == "{}" {
		m = map[string]json.RawMessage{}
	} else if err := json.Unmarshal(params, &m); err != nil {
		return nil, fmt.Errorf("invalid execution parameters: %w", err)
	}

	if window.StartsAt != nil {
		effective := *window.StartsAt
		if raw, ok := m[startParam]; ok {
			if t, ok, err := parseTimeParam(raw); err != nil {
				return nil, err
			} else if ok {
				if t.After(effective) {
					effective = t
				}
			}
		}
		if window.EndsAt != nil && !effective.Before(*window.EndsAt) {
			effective = window.EndsAt.Add(-time.Nanosecond)
		}
		encoded, err := marshalRFC3339(effective)
		if err != nil {
			return nil, err
		}
		m[startParam] = encoded
	}

	if window.EndsAt != nil {
		effective := *window.EndsAt
		if raw, ok := m[endParam]; ok {
			if t, ok, err := parseTimeParam(raw); err != nil {
				return nil, err
			} else if ok {
				if t.Before(effective) {
					effective = t
				}
			}
		}
		if window.StartsAt != nil && !effective.After(*window.StartsAt) {
			effective = window.StartsAt.Add(time.Nanosecond)
		}
		encoded, err := marshalRFC3339(effective)
		if err != nil {
			return nil, err
		}
		m[endParam] = encoded
	}

	out, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ConstraintsHaveDataWindow reports whether constraints include $data_window.
func ConstraintsHaveDataWindow(constraints json.RawMessage) bool {
	if len(constraints) == 0 || string(constraints) == "null" {
		return false
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(constraints, &obj); err != nil {
		return false
	}
	raw, ok := obj[DataWindowNamespaceKey]
	return ok && len(raw) > 0 && string(raw) != "null" && string(raw) != "{}"
}

// ExtractDataWindowConstraint returns the raw $data_window value from constraints.
func ExtractDataWindowConstraint(constraints json.RawMessage) (json.RawMessage, bool) {
	if len(constraints) == 0 {
		return nil, false
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(constraints, &obj); err != nil {
		return nil, false
	}
	raw, ok := obj[DataWindowNamespaceKey]
	return raw, ok && len(raw) > 0 && string(raw) != "null"
}

func parseTimeParam(raw json.RawMessage) (time.Time, bool, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return time.Time{}, false, &ConfigValidationError{
			Parameter: "datetime",
			Reason:    "window parameter must be an RFC3339 timestamp string",
		}
	}
	if s == "" {
		return time.Time{}, false, nil
	}
	t, err := parseRFC3339(s)
	if err != nil {
		return time.Time{}, false, &ConfigValidationError{
			Parameter: "datetime",
			Reason:    "window parameter must be an RFC3339 timestamp string",
		}
	}
	return t, true, nil
}

func parseRFC3339(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Parse(time.RFC3339Nano, s)
	}
	return t.UTC(), nil
}

func marshalRFC3339(t time.Time) (json.RawMessage, error) {
	s, err := json.Marshal(t.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	return s, nil
}
