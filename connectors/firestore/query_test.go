package firestore

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestQuery_EmptyDocumentsIsJSONArray(t *testing.T) {
	t.Parallel()
	mock := newMockRunner()
	mock.pushQueryBatch([]map[string]interface{}{})
	conn := newForTest(mock)
	action := conn.Actions()["firestore.query"]

	res, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "firestore.query",
		Parameters: json.RawMessage(`{
			"collection_path":"posts",
			"allowed_paths":["posts"]
		}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := string(res.Data)
	if raw == "" {
		t.Fatal("empty result JSON")
	}
	if !json.Valid(res.Data) {
		t.Fatalf("invalid JSON: %s", raw)
	}
	var payload struct {
		Documents []any `json:"documents"`
		Count     int   `json:"count"`
	}
	if err := json.Unmarshal(res.Data, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Documents == nil {
		t.Fatalf("documents must be non-nil empty slice, got null in JSON: %s", raw)
	}
	if len(payload.Documents) != 0 {
		t.Fatalf("want 0 documents, got %d", len(payload.Documents))
	}
	if payload.Count != 0 {
		t.Fatalf("count = %d, want 0", payload.Count)
	}
}
