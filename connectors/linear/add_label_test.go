package linear

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestAddLabel_Success(t *testing.T) {
	t.Parallel()

	// The add_label action makes two GraphQL calls: first to fetch current
	// labels, then to update with the new label added.
	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				// First call returns current labels; second call returns the update result.
				// Since we use the same handler for both, the response must satisfy both.
				"issue": map[string]any{
					"labels": map[string]any{
						"nodes": []map[string]string{
							{"id": "label-existing"},
						},
					},
				},
				"issueUpdate": map[string]any{
					"success": true,
					"issue": map[string]any{
						"id":         "issue-1",
						"identifier": "ENG-1",
						"labels": map[string]any{
							"nodes": []map[string]string{
								{"id": "label-existing", "name": "existing"},
								{"id": "label-new", "name": "new-label"},
							},
						},
					},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &addLabelAction{conn: conn}

	params, _ := json.Marshal(addLabelParams{IssueID: "issue-1", LabelID: "label-new"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.add_label",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]interface{}
	json.Unmarshal(result.Data, &data)
	labels, ok := data["labels"].([]interface{})
	if !ok {
		t.Fatal("expected labels array")
	}
	if len(labels) != 2 {
		t.Errorf("got %d labels, want 2", len(labels))
	}
}

func TestAddLabel_MissingIssueID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addLabelAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"label_id": "label-1"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.add_label",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing issue_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAddLabel_MissingLabelID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addLabelAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"issue_id": "issue-1"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.add_label",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing label_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
