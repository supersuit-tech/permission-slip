package connectors

import (
	"strconv"
	"testing"
	"time"
)

func TestParseUnixTimestampOrRFC3339(t *testing.T) {
	t.Parallel()
	mar1 := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC).Unix()
	cases := []struct {
		in   string
		want int64
	}{
		{"", 0},
		{"1709251200", 1709251200},
		{"1709251200000", 1709251200},
		{"2024-03-01T00:00:00Z", mar1},
		{"2024-03-01T00:00:00.123456789Z", mar1}, // fractional seconds
	}
	for _, tc := range cases {
		got, err := ParseUnixTimestampOrRFC3339(tc.in)
		if err != nil {
			t.Errorf("ParseUnixTimestampOrRFC3339(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseUnixTimestampOrRFC3339(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestParseUnixTimestampOrRFC3339_Errors(t *testing.T) {
	t.Parallel()
	badInputs := []string{
		"not-a-date",
		"2024-13-01",
		"yesterday",
	}
	for _, in := range badInputs {
		_, err := ParseUnixTimestampOrRFC3339(in)
		if err == nil {
			t.Errorf("ParseUnixTimestampOrRFC3339(%q) expected error, got nil", in)
		}
	}
}

func TestNormalizeHubSpotAnalyticsTimeParam(t *testing.T) {
	t.Parallel()
	got, err := NormalizeHubSpotAnalyticsTimeParam("2026-01-01")
	if err != nil || got != "2026-01-01" {
		t.Errorf("date-only: got %q err %v", got, err)
	}
	got, err = NormalizeHubSpotAnalyticsTimeParam("2026-01-01T12:00:00Z")
	if err != nil || got != "2026-01-01" {
		t.Errorf("rfc3339: got %q err %v", got, err)
	}
	ms := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	got, err = NormalizeHubSpotAnalyticsTimeParam(strconv.FormatInt(ms, 10))
	if err != nil || got != "2025-02-01" {
		t.Errorf("epoch ms: got %q err %v", got, err)
	}
}

func TestNormalizeHubSpotAnalyticsTimeParam_Errors(t *testing.T) {
	t.Parallel()
	badInputs := []string{"not-a-date", "yesterday"}
	for _, in := range badInputs {
		_, err := NormalizeHubSpotAnalyticsTimeParam(in)
		if err == nil {
			t.Errorf("NormalizeHubSpotAnalyticsTimeParam(%q) expected error, got nil", in)
		}
	}
}

func TestNormalizePlaidDateParam(t *testing.T) {
	t.Parallel()
	got, err := NormalizePlaidDateParam("2025-06-15")
	if err != nil || got != "2025-06-15" {
		t.Errorf("date: got %q err %v", got, err)
	}
	got, err = NormalizePlaidDateParam("2025-06-15T23:59:59Z")
	if err != nil || got != "2025-06-15" {
		t.Errorf("rfc3339: got %q err %v", got, err)
	}
}

func TestNormalizePlaidDateParam_Errors(t *testing.T) {
	t.Parallel()
	badInputs := []string{"", "not-a-date", "yesterday"}
	for _, in := range badInputs {
		_, err := NormalizePlaidDateParam(in)
		if err == nil {
			t.Errorf("NormalizePlaidDateParam(%q) expected error, got nil", in)
		}
	}
}
