package db

import (
	"encoding/json"
	"testing"
	"time"
)

func TestValidateRelativeDateToken(t *testing.T) {
	t.Parallel()
	valid := []string{"@today", "@yesterday", "@now", "-7d", "-30d"}
	for _, token := range valid {
		if err := ValidateRelativeDateToken(token); err != nil {
			t.Errorf("expected valid token %q, got %v", token, err)
		}
	}
	invalid := []string{"", "today", "@tomorrow", "-0d", "-7days", "2026-01-01"}
	for _, token := range invalid {
		if err := ValidateRelativeDateToken(token); err == nil {
			t.Errorf("expected invalid token %q", token)
		}
	}
}

func TestResolveRelativeDateToken_Today(t *testing.T) {
	t.Parallel()
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	now := time.Date(2026, 7, 7, 15, 30, 0, 0, loc)
	got, err := ResolveRelativeDateToken("@today", now, loc)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	want := time.Date(2026, 7, 7, 0, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestResolveRelativeDateToken_Yesterday(t *testing.T) {
	t.Parallel()
	loc := time.UTC
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, loc)
	got, err := ResolveRelativeDateToken("@yesterday", now, loc)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	want := time.Date(2026, 7, 6, 0, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestResolveRelativeDateToken_RollingDays(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	got, err := ResolveRelativeDateToken("-7d", now, time.UTC)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	want := now.AddDate(0, 0, -7)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestExecValueWithinRelativeDateBound_Lower(t *testing.T) {
	t.Parallel()
	loc := time.UTC
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, loc)

	if err := ExecValueWithinRelativeDateBound(
		"since", "@today", nil, RelativeDateBoundLower, now, loc,
	); err != nil {
		t.Fatalf("empty value should pass: %v", err)
	}

	afterToday, _ := jsonMarshalString("2026-07-07T13:00:00Z")
	if err := ExecValueWithinRelativeDateBound(
		"since", "@today", afterToday, RelativeDateBoundLower, now, loc,
	); err != nil {
		t.Fatalf("after bound should pass: %v", err)
	}

	beforeToday, _ := jsonMarshalString("2026-07-06T23:00:00Z")
	if err := ExecValueWithinRelativeDateBound(
		"since", "@today", beforeToday, RelativeDateBoundLower, now, loc,
	); err == nil {
		t.Fatal("expected rejection for value before @today")
	}
}

func TestApplyRelativeDateConstraintsToParams_InjectsSince(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	constraints := json.RawMessage(`{"limit":"*","since":"@today"}`)
	fields := map[string]DateTimeFieldInfo{
		"since": {Format: "date-time", Role: "lower"},
	}
	out, err := ApplyRelativeDateConstraintsToParams(
		json.RawMessage(`{"limit":20}`),
		constraints,
		fields,
		now,
		time.UTC,
	)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	var recorded map[string]any
	if err := json.Unmarshal(out, &recorded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	sinceStr, ok := recorded["since"].(string)
	if !ok || sinceStr == "" {
		t.Fatalf("expected injected since, got %v", recorded["since"])
	}
	if sinceStr != "2026-07-07T00:00:00Z" {
		t.Fatalf("since = %q, want start of day UTC", sinceStr)
	}
}

func TestApplyRelativeDateConstraintsToParams_ClampOlderStart(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	constraints := json.RawMessage(`{"since":"@today"}`)
	fields := map[string]DateTimeFieldInfo{"since": {Format: "date-time", Role: "lower"}}
	out, err := ApplyRelativeDateConstraintsToParams(
		json.RawMessage(`{"since":"2026-01-01T00:00:00Z"}`),
		constraints,
		fields,
		now,
		time.UTC,
	)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	var recorded map[string]string
	if err := json.Unmarshal(out, &recorded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if recorded["since"] != "2026-07-07T00:00:00Z" {
		t.Fatalf("since = %q, want clamped to @today", recorded["since"])
	}
}

func TestApplyRelativeDateConstraintsToParams_DateFormat(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	constraints := json.RawMessage(`{"since":"@today"}`)
	fields := map[string]DateTimeFieldInfo{"since": {Format: "date", Role: "lower"}}
	out, err := ApplyRelativeDateConstraintsToParams(
		json.RawMessage(`{}`),
		constraints,
		fields,
		now,
		time.UTC,
	)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	var recorded map[string]string
	if err := json.Unmarshal(out, &recorded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if recorded["since"] != "2026-07-07" {
		t.Fatalf("since = %q, want YYYY-MM-DD", recorded["since"])
	}
}

func TestParseActionSchemaDateTimeFields(t *testing.T) {
	t.Parallel()
	schema := []byte(`{
		"type":"object",
		"properties":{
			"since":{"type":"string","format":"date-time","x-ui":{"datetime_range_role":"lower"}},
			"before":{"type":"string","format":"date-time","x-ui":{"datetime_range_role":"upper"}},
			"limit":{"type":"integer"}
		}
	}`)
	fields, err := ParseActionSchemaDateTimeFields(schema)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fields["since"].Format != "date-time" || fields["since"].Role != "lower" {
		t.Fatalf("since field: %+v", fields["since"])
	}
	if fields["before"].Role != "upper" {
		t.Fatalf("before field: %+v", fields["before"])
	}
	if _, ok := fields["limit"]; ok {
		t.Fatal("limit should not be datetime field")
	}
}

func jsonMarshalString(s string) (json.RawMessage, error) {
	b, err := json.Marshal(s)
	return b, err
}
