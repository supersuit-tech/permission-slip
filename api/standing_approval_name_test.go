package api

import (
	"encoding/json"
	"testing"
)

func TestDeriveStandingApprovalNameFromRequest_MetaFrom(t *testing.T) {
	t.Parallel()
	name := deriveStandingApprovalNameFromRequest("Read Email", json.RawMessage(`{
		"message_id":"*",
		"folder":"*",
		"$meta":{"from":"automated@airbnb.com"}
	}`), nil)
	if name != "Read Email — from automated@airbnb.com" {
		t.Fatalf("name = %q", name)
	}
}

func TestDeriveStandingApprovalNameFromRequest_MetaDriveID(t *testing.T) {
	t.Parallel()
	name := deriveStandingApprovalNameFromRequest("Upload Drive File", json.RawMessage(`{
		"folder_id":"*",
		"$meta":{"drive_id":"0AKbIIKZ8knmBUk9PVA"}
	}`), nil)
	if name != "Upload Drive File — in Shared Drive 0AKbIIKZ8knmBUk9PVA" {
		t.Fatalf("name = %q", name)
	}
}

func TestDeriveStandingApprovalNameFromRequest_MetaCalendarID(t *testing.T) {
	t.Parallel()
	name := deriveStandingApprovalNameFromRequest("Create Calendar Event", json.RawMessage(`{
		"calendar_id":"*",
		"$meta":{"calendar_id":"work@example.com"}
	}`), nil)
	if name != "Create Calendar Event — on calendar work@example.com" {
		t.Fatalf("name = %q", name)
	}
}

func TestDeriveStandingApprovalNameFromRequest_OverlaysResourceNames(t *testing.T) {
	t.Parallel()
	details := json.RawMessage(`{
		"resources":{
			"calendar_id":{"c_abc@group.calendar.google.com":{"name":"Team Calendar"}},
			"drive_id":{"0AKbIIKZ8knmBUk9PVA":{"name":"Finance Shared Drive"}}
		}
	}`)
	calendarName := deriveStandingApprovalNameFromRequest("Create Calendar Event", json.RawMessage(`{
		"calendar_id":"*",
		"$meta":{"calendar_id":"c_abc@group.calendar.google.com"}
	}`), details)
	if calendarName != "Create Calendar Event — on calendar Team Calendar" {
		t.Fatalf("calendar name = %q", calendarName)
	}
	driveName := deriveStandingApprovalNameFromRequest("Upload Drive File", json.RawMessage(`{
		"folder_id":"*",
		"$meta":{"drive_id":"0AKbIIKZ8knmBUk9PVA"}
	}`), details)
	if driveName != "Upload Drive File — in Shared Drive Finance Shared Drive" {
		t.Fatalf("drive name = %q", driveName)
	}
}

func TestDeriveStandingApprovalNameFromRequest_ParamConstraint(t *testing.T) {
	t.Parallel()
	name := deriveStandingApprovalNameFromRequest("Send Email", json.RawMessage(`{"to":"*@example.com"}`), nil)
	if name != "Send Email — to: *@example.com" {
		t.Fatalf("name = %q", name)
	}
}

func TestDeriveStandingApprovalNameFromRequest_AllWildcard(t *testing.T) {
	t.Parallel()
	name := deriveStandingApprovalNameFromRequest("Read Email", json.RawMessage(`{"message_id":"*","folder":"*"}`), nil)
	if name != "Read Email auto-approve" {
		t.Fatalf("name = %q", name)
	}
}

func TestDeriveStandingApprovalNameFromRequest_TruncatesLongName(t *testing.T) {
	t.Parallel()
	long := make([]byte, 300)
	for i := range long {
		long[i] = 'a'
	}
	name := deriveStandingApprovalNameFromRequest("Read Email", json.RawMessage(`{"to":"`+string(long)+`"}`), nil)
	if len(name) > maxStandingApprovalNameLength {
		t.Fatalf("name length = %d, want <= %d", len(name), maxStandingApprovalNameLength)
	}
}
