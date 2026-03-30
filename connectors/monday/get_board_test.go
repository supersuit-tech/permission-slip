package monday

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetBoard_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var gqlReq map[string]any
		json.Unmarshal(body, &gqlReq)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"boards": []map[string]any{
					{
						"id":         "12345",
						"name":       "Sprint Board",
						"state":      "active",
						"board_kind": "public",
						"columns": []map[string]any{
							{"id": "name", "title": "Name", "type": "name"},
						},
						"groups": []map[string]any{
							{"id": "new_group", "title": "To Do", "color": "#ff0000"},
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["monday.get_board"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.get_board",
		Parameters:  json.RawMessage(`{"board_id":"12345"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["name"] != "Sprint Board" {
		t.Errorf("name = %v, want Sprint Board", data["name"])
	}
}

func TestGetBoard_MissingBoardID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["monday.get_board"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.get_board",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestGetBoard_InvalidBoardID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["monday.get_board"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.get_board",
		Parameters:  json.RawMessage(`{"board_id":"not-numeric"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestGetBoard_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"boards": []any{},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["monday.get_board"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.get_board",
		Parameters:  json.RawMessage(`{"board_id":"99999"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}
