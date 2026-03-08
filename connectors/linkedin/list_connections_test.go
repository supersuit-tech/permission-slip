package linkedin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListConnections_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/connections") {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.URL.Query().Get("q"); got != "viewer" {
			t.Errorf("expected q=viewer, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(connectionsResponse{
			Elements: []connectionElement{
				{ID: "person1", FirstName: "Alice", LastName: "Smith", Headline: "CEO at Startup"},
				{ID: "person2", FirstName: "Bob", LastName: "Jones", Headline: "Engineer"},
			},
			Paging: searchPaging{Total: 2, Start: 0, Count: 20},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &listConnectionsAction{conn: conn}

	params, _ := json.Marshal(listConnectionsParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.list_connections",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	connections, ok := data["connections"].([]any)
	if !ok || len(connections) != 2 {
		t.Fatalf("expected 2 connections, got: %v", data["connections"])
	}
	first := connections[0].(map[string]any)
	if first["id"] != "person1" {
		t.Errorf("expected id 'person1', got %v", first["id"])
	}
	if first["first_name"] != "Alice" {
		t.Errorf("expected first_name 'Alice', got %v", first["first_name"])
	}
	if first["person_urn"] != "urn:li:person:person1" {
		t.Errorf("expected person_urn 'urn:li:person:person1', got %v", first["person_urn"])
	}
	// next_start should equal start (0) + 2 returned = 2
	if ns, ok := data["next_start"].(float64); !ok || ns != 2 {
		t.Errorf("expected next_start 2, got %v", data["next_start"])
	}
}

func TestListConnections_CountTooLarge(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listConnectionsAction{conn: conn}

	params, _ := json.Marshal(listConnectionsParams{Count: maxConnectionsCount + 1})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.list_connections",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for count too large")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListConnections_NegativeStart(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listConnectionsAction{conn: conn}

	params, _ := json.Marshal(listConnectionsParams{Start: -1})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.list_connections",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for negative start")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListConnections_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listConnectionsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.list_connections",
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
