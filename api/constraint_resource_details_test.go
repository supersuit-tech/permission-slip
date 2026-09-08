package api

import (
	"encoding/json"
	"testing"
)

func TestCollectFixedConstraintIDs_LegacyFlat(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"spreadsheet_id":"s123","range":"A1:B2","to":"*"}`)
	got, err := collectFixedConstraintIDs(raw)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(got["spreadsheet_id"]) != 1 || got["spreadsheet_id"][0] != "s123" {
		t.Errorf("spreadsheet_id: %#v", got["spreadsheet_id"])
	}
	if len(got["range"]) != 1 || got["range"][0] != "A1:B2" {
		t.Errorf("range: %#v", got["range"])
	}
	if _, ok := got["to"]; ok {
		t.Errorf("wildcard to should be skipped, got %#v", got["to"])
	}
}

func TestCollectFixedConstraintIDs_SkipsPatternsAndMeta(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"$version":2,
		"match":"any",
		"groups":[{
			"match":"all",
			"conditions":[
				{"field":"spreadsheet_id","op":"any_of","values":["abc","def"]},
				{"field":"folder_id","op":"matches","value":{"$pattern":"foo*"}},
				{"field":"$meta.from","op":"matches","value":"a@b.com"},
				{"field":"count","op":"lte","value":10},
				{"field":"issue_number","op":"matches","value":42}
			]
		}]
	}`)
	got, err := collectFixedConstraintIDs(raw)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(got["spreadsheet_id"]) != 2 {
		t.Errorf("expected two spreadsheet ids, got %#v", got["spreadsheet_id"])
	}
	if _, ok := got["folder_id"]; ok {
		t.Errorf("pattern folder_id should be skipped, got %#v", got["folder_id"])
	}
	if _, ok := got["$meta.from"]; ok {
		t.Errorf("$meta.from should be skipped")
	}
	if _, ok := got["from"]; ok {
		t.Errorf("email $meta.from should not become a from lookup, got %#v", got["from"])
	}
	if _, ok := got["count"]; ok {
		t.Errorf("comparison op should be skipped")
	}
	if len(got["issue_number"]) != 1 || got["issue_number"][0] != "42" {
		t.Errorf("issue_number: %#v", got["issue_number"])
	}
}

func TestCollectFixedConstraintIDs_CollectsMetaResourceIDs(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"event_id":"*",
		"calendar_id":"*",
		"$meta":{"calendar_id":"c_abc@group.calendar.google.com","from":"skip@example.com"}
	}`)
	got, err := collectFixedConstraintIDs(raw)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(got["calendar_id"]) != 1 || got["calendar_id"][0] != "c_abc@group.calendar.google.com" {
		t.Errorf("calendar_id: %#v", got["calendar_id"])
	}
	if _, ok := got["from"]; ok {
		t.Errorf("email $meta.from should be skipped, got %#v", got["from"])
	}
	if _, ok := got["event_id"]; ok {
		t.Errorf("wildcard event_id should be skipped, got %#v", got["event_id"])
	}

	driveRaw := []byte(`{"folder_id":"*","$meta":{"drive_id":"0AKbIIKZ8knmBUk9PVA"}}`)
	driveGot, err := collectFixedConstraintIDs(driveRaw)
	if err != nil {
		t.Fatalf("collect drive: %v", err)
	}
	if len(driveGot["drive_id"]) != 1 || driveGot["drive_id"][0] != "0AKbIIKZ8knmBUk9PVA" {
		t.Errorf("drive_id: %#v", driveGot["drive_id"])
	}
}

func TestDecodeFixedConstraintID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		raw  string
		want string
		ok   bool
	}{
		{`"abc"`, "abc", true},
		{`"*"`, "", false},
		{`""`, "", false},
		{`42`, "42", true},
		{`{"$pattern":"x*"}`, "", false},
		{`null`, "", false},
	}
	for _, tc := range cases {
		got, ok := decodeFixedConstraintID(json.RawMessage(tc.raw))
		if ok != tc.ok || got != tc.want {
			t.Errorf("%s: got (%q,%v) want (%q,%v)", tc.raw, got, ok, tc.want, tc.ok)
		}
	}
}
