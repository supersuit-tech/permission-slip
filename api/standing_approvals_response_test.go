package api

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
)

func TestToStandingApprovalResponse_NullConstraints(t *testing.T) {
	t.Parallel()

	resp := toStandingApprovalResponse(db.StandingApproval{
		StandingApprovalID: "sa_test",
		AgentID:            1,
		UserID:             "user_1",
		ActionType:         "github.create_issue",
		Status:             "active",
		StartsAt:           time.Now().UTC(),
		CreatedAt:          time.Now().UTC(),
	})

	if resp.Constraints == nil {
		t.Fatal("expected constraints to be serialized as empty object, got nil")
	}
	obj, ok := resp.Constraints.(map[string]any)
	if !ok {
		t.Fatalf("expected constraints to be map[string]any, got %T", resp.Constraints)
	}
	if len(obj) != 0 {
		t.Fatalf("expected empty constraints object, got %#v", obj)
	}
}

func TestToStandingApprovalResponse_PreservesConstraints(t *testing.T) {
	t.Parallel()

	resp := toStandingApprovalResponse(db.StandingApproval{
		StandingApprovalID: "sa_test",
		AgentID:            1,
		UserID:             "user_1",
		ActionType:         "github.create_issue",
		Status:             "active",
		Constraints:        []byte(`{"repo":"myorg/*"}`),
		StartsAt:           time.Now().UTC(),
		CreatedAt:          time.Now().UTC(),
	})

	obj, ok := resp.Constraints.(map[string]any)
	if !ok {
		t.Fatalf("expected constraints to be map[string]any, got %T", resp.Constraints)
	}
	if obj["repo"] != "myorg/*" {
		t.Fatalf("expected repo constraint, got %#v", obj)
	}
}

func TestBuildStandingApprovalConstraintsFromTemplate_AllWildcard(t *testing.T) {
	t.Parallel()

	got, err := buildStandingApprovalConstraintsFromTemplate(context.Background(), nil, nil, "email.send", []byte(`{"to":"*","subject":"*","body":"*"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "{}" {
		t.Fatalf("expected {}, got %q", string(got))
	}
}

func TestBuildStandingApprovalConstraintsFromTemplate_EmptyObject(t *testing.T) {
	t.Parallel()

	got, err := buildStandingApprovalConstraintsFromTemplate(context.Background(), nil, nil, "email.send", []byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "{}" {
		t.Fatalf("expected {}, got %q", string(got))
	}
}

func TestToStandingApprovalResponse_JSONOmitsNullConstraints(t *testing.T) {
	t.Parallel()

	resp := toStandingApprovalResponse(db.StandingApproval{
		StandingApprovalID: "sa_test",
		AgentID:            1,
		UserID:             "user_1",
		ActionType:         "github.create_issue",
		Status:             "active",
		StartsAt:           time.Now().UTC(),
		CreatedAt:          time.Now().UTC(),
	})

	body, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(body) == "" {
		t.Fatal("expected non-empty JSON")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	constraints, ok := raw["constraints"]
	if !ok {
		t.Fatal("expected constraints key in JSON response")
	}
	if string(constraints) == "null" {
		t.Fatalf("expected constraints to serialize as {}, got null")
	}
	if string(constraints) != "{}" {
		t.Fatalf("expected constraints {}, got %s", string(constraints))
	}
}
