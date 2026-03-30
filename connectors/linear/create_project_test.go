package linear

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateProject_Success(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				"projectCreate": map[string]any{
					"success": true,
					"project": map[string]string{
						"id":   "project-uuid-1",
						"name": "Q1 Roadmap",
						"url":  "https://linear.app/team/project/q1-roadmap",
					},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &createProjectAction{conn: conn}

	params, _ := json.Marshal(createProjectParams{
		TeamIDs: []string{"team-1"},
		Name:    "Q1 Roadmap",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.create_project",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["id"] != "project-uuid-1" {
		t.Errorf("id = %q, want %q", data["id"], "project-uuid-1")
	}
	if data["name"] != "Q1 Roadmap" {
		t.Errorf("name = %q, want %q", data["name"], "Q1 Roadmap")
	}
}

func TestCreateProject_WithOptionalFields(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				"projectCreate": map[string]any{
					"success": true,
					"project": map[string]string{
						"id":   "project-uuid-2",
						"name": "Feature Work",
						"url":  "https://linear.app/team/project/feature-work",
					},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &createProjectAction{conn: conn}

	params, _ := json.Marshal(createProjectParams{
		TeamIDs:     []string{"team-1", "team-2"},
		Name:        "Feature Work",
		Description: "Cross-team feature project",
		State:       "started",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.create_project",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["name"] != "Feature Work" {
		t.Errorf("name = %q, want %q", data["name"], "Feature Work")
	}
}

func TestCreateProject_MissingTeamIDs(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createProjectAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"name": "Test"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.create_project",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing team_ids")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateProject_MissingName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createProjectAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"team_ids": []string{"team-1"}})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.create_project",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateProject_InvalidState(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createProjectAction{conn: conn}

	params, _ := json.Marshal(createProjectParams{
		TeamIDs: []string{"team-1"},
		Name:    "Test",
		State:   "invalid_state",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.create_project",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid state")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateProject_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createProjectAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.create_project",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateProject_SuccessFalse(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				"projectCreate": map[string]any{
					"success": false,
					"project": nil,
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &createProjectAction{conn: conn}

	params, _ := json.Marshal(createProjectParams{
		TeamIDs: []string{"team-1"},
		Name:    "Test",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.create_project",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for success=false")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}
