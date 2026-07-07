package db

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Relative date tokens resolve at standing-approval execution time so date
// constraints stay fresh without daily rule updates.
const (
	RelativeDateToday     = "@today"
	RelativeDateYesterday = "@yesterday"
	RelativeDateNow       = "@now"
)

var relativeDateOffsetPattern = regexp.MustCompile(`^-(\d+)d$`)

// IsRelativeDateToken reports whether s is a supported relative date token.
func IsRelativeDateToken(s string) bool {
	return ValidateRelativeDateToken(s) == nil
}

// ValidateRelativeDateToken checks token syntax at constraint write time.
func ValidateRelativeDateToken(s string) error {
	s = strings.TrimSpace(s)
	switch s {
	case RelativeDateToday, RelativeDateYesterday, RelativeDateNow:
		return nil
	}
	if relativeDateOffsetPattern.MatchString(s) {
		m := relativeDateOffsetPattern.FindStringSubmatch(s)
		if m != nil {
			days, err := strconv.Atoi(m[1])
			if err != nil || days < 1 {
				return fmt.Errorf("invalid relative date token %q", s)
			}
			return nil
		}
	}
	return fmt.Errorf("invalid relative date token %q", s)
}

// ExtractRelativeDateToken returns the token string when raw is a JSON string
// containing a supported relative date token.
func ExtractRelativeDateToken(raw json.RawMessage) (string, bool) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", false
	}
	s = strings.TrimSpace(s)
	if ValidateRelativeDateToken(s) != nil {
		return "", false
	}
	return s, true
}

// ResolveRelativeDateToken converts a token to an absolute instant in loc.
func ResolveRelativeDateToken(token string, now time.Time, loc *time.Location) (time.Time, error) {
	if loc == nil {
		loc = time.UTC
	}
	token = strings.TrimSpace(token)
	if err := ValidateRelativeDateToken(token); err != nil {
		return time.Time{}, err
	}

	localNow := now.In(loc)
	switch token {
	case RelativeDateToday:
		y, m, d := localNow.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, loc), nil
	case RelativeDateYesterday:
		y, m, d := localNow.AddDate(0, 0, -1).Date()
		return time.Date(y, m, d, 0, 0, 0, 0, loc), nil
	case RelativeDateNow:
		return now.In(loc), nil
	}

	m := relativeDateOffsetPattern.FindStringSubmatch(token)
	if m == nil {
		return time.Time{}, fmt.Errorf("invalid relative date token %q", token)
	}
	days, _ := strconv.Atoi(m[1])
	return now.In(loc).AddDate(0, 0, -days), nil
}

// RelativeDateBoundRole describes how a datetime constraint field should be
// compared against a resolved token.
type RelativeDateBoundRole string

const (
	RelativeDateBoundLower RelativeDateBoundRole = "lower"
	RelativeDateBoundUpper RelativeDateBoundRole = "upper"
	RelativeDateBoundExact RelativeDateBoundRole = "exact"
)

// RelativeDateBoundRoleForToken picks a comparison role from token semantics
// and optional schema metadata (datetime_range_role).
func RelativeDateBoundRoleForToken(token string, schemaRole string) RelativeDateBoundRole {
	switch strings.TrimSpace(schemaRole) {
	case "lower":
		return RelativeDateBoundLower
	case "upper":
		return RelativeDateBoundUpper
	}
	switch strings.TrimSpace(token) {
	case RelativeDateNow:
		return RelativeDateBoundUpper
	default:
		return RelativeDateBoundLower
	}
}

// ExecValueWithinRelativeDateBound checks whether an execution parameter
// satisfies a relative date constraint.
func ExecValueWithinRelativeDateBound(
	param string,
	token string,
	execValue json.RawMessage,
	role RelativeDateBoundRole,
	now time.Time,
	loc *time.Location,
) error {
	if len(execValue) == 0 {
		return nil
	}
	var execStr string
	if err := json.Unmarshal(execValue, &execStr); err != nil {
		return &ConfigValidationError{
			Parameter: param,
			Reason:    "value must be a string timestamp for relative date constraint",
		}
	}
	if execStr == "" {
		return nil
	}

	bound, err := ResolveRelativeDateToken(token, now, loc)
	if err != nil {
		return err
	}
	execTime, err := parseFlexibleDateTime(execStr)
	if err != nil {
		return &ConfigValidationError{
			Parameter: param,
			Reason:    fmt.Sprintf("value must be a valid timestamp for relative date constraint %q", token),
		}
	}

	switch role {
	case RelativeDateBoundUpper:
		if execTime.After(bound) {
			return &ConfigValidationError{
				Parameter: param,
				Reason:    fmt.Sprintf("value must be on or before relative date bound %q", token),
			}
		}
	case RelativeDateBoundExact:
		if !execTime.Equal(bound) {
			return &ConfigValidationError{
				Parameter: param,
				Reason:    fmt.Sprintf("value must match relative date bound %q", token),
			}
		}
	default:
		if execTime.Before(bound) {
			return &ConfigValidationError{
				Parameter: param,
				Reason:    fmt.Sprintf("value must be on or after relative date bound %q", token),
			}
		}
	}
	return nil
}

func parseFlexibleDateTime(s string) (time.Time, error) {
	if t, err := parseRFC3339(s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("unrecognized datetime format")
}

func marshalRelativeDateValue(t time.Time, format string) (json.RawMessage, error) {
	switch format {
	case "date":
		s, err := json.Marshal(t.Format("2006-01-02"))
		if err != nil {
			return nil, err
		}
		return s, nil
	default:
		return marshalRFC3339(t)
	}
}
