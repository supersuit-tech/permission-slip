package connectors

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var yyyymmddPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// ParseUnixTimestampOrRFC3339 parses a value that may be a Unix timestamp in seconds
// (optionally fractional), epoch milliseconds, or an RFC 3339 datetime. Empty string returns (0, nil).
func ParseUnixTimestampOrRFC3339(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return unixSecondsFromNumeric(f), nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return 0, fmt.Errorf("expected Unix timestamp or RFC 3339 datetime, got %q", s)
	}
	return t.Unix(), nil
}

// NormalizeHubSpotAnalyticsTimeParam converts start/end query values for HubSpot analytics.
// Accepts YYYY-MM-DD, RFC 3339, Unix seconds, or epoch milliseconds. Returns YYYY-MM-DD in UTC.
func NormalizeHubSpotAnalyticsTimeParam(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if yyyymmddPattern.MatchString(s) {
		return s, nil
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		sec := unixSecondsFromNumeric(f)
		t := time.Unix(sec, 0).UTC()
		return t.Format("2006-01-02"), nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return "", fmt.Errorf("expected YYYY-MM-DD, RFC 3339, or Unix timestamp, got %q", s)
	}
	return t.UTC().Format("2006-01-02"), nil
}

// NormalizePlaidDateParam accepts YYYY-MM-DD or RFC 3339 / Unix timestamps and returns YYYY-MM-DD (UTC calendar date).
func NormalizePlaidDateParam(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("empty date")
	}
	if yyyymmddPattern.MatchString(s) {
		return s, nil
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		sec := unixSecondsFromNumeric(f)
		t := time.Unix(sec, 0).UTC()
		return t.Format("2006-01-02"), nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return "", fmt.Errorf("expected YYYY-MM-DD, RFC 3339, or Unix timestamp, got %q", s)
	}
	return t.UTC().Format("2006-01-02"), nil
}

// unixSecondsFromNumeric interprets a JSON-unmarshaled time as Unix seconds or epoch millis.
// Values >= 1e11 are treated as milliseconds (covers epoch-ms through year ~5138 in seconds).
func unixSecondsFromNumeric(f float64) int64 {
	if f >= 1e11 {
		return time.UnixMilli(int64(f)).Unix()
	}
	return int64(f)
}
