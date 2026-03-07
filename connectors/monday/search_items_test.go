package monday

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSearchItems_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"boards": []any{
					map[string]any{
						"items_page": map[string]any{
							"items": []any{
								map[string]any{
									"id":   "111",
									"name": "Task A",
									"column_values": []any{
										map[string]any{
											"id":   "status",
											"title": map[string]string{"title": "Status"},
											"text": "Working on it",
										},
									},
								},
								map[string]any{
									"id":   "222",
									"name": "Task B",
									"column_values": []any{},
								},
							},
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &searchItemsAction{conn: conn}

	params, _ := json.Marshal(searchItemsParams{
		BoardID: "9876",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.search_items",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Items []searchItemResult `json:"items"`
		Count int                `json:"count"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.Count != 2 {
		t.Errorf("expected count 2, got %d", data.Count)
	}
	if len(data.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(data.Items))
	}
	if data.Items[0].ID != "111" {
		t.Errorf("expected first item id '111', got %q", data.Items[0].ID)
	}
	if data.Items[0].Name != "Task A" {
		t.Errorf("expected first item name 'Task A', got %q", data.Items[0].Name)
	}
}

func TestSearchItems_WithQuery(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		// Verify query variable is passed.
		if _, ok := body.Variables["query"]; !ok {
			t.Error("expected query in variables for text search")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"boards": []any{
					map[string]any{
						"items_page": map[string]any{
							"items": []any{
								map[string]any{
									"id":            "333",
									"name":          "Matching Task",
									"column_values": []any{},
								},
							},
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &searchItemsAction{conn: conn}

	params, _ := json.Marshal(searchItemsParams{
		BoardID: "9876",
		Query:   "Matching",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.search_items",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Items []searchItemResult `json:"items"`
		Count int                `json:"count"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.Count != 1 {
		t.Errorf("expected count 1, got %d", data.Count)
	}
}

func TestSearchItems_WithColumnFilter(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"boards": []any{
					map[string]any{
						"items_page": map[string]any{
							"items": []any{},
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &searchItemsAction{conn: conn}

	params, _ := json.Marshal(searchItemsParams{
		BoardID:     "9876",
		ColumnID:    "status",
		ColumnValue: "Done",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.search_items",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Items []searchItemResult `json:"items"`
		Count int                `json:"count"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.Count != 0 {
		t.Errorf("expected count 0, got %d", data.Count)
	}
}

func TestSearchItems_MissingBoardID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchItemsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.search_items",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing board_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchItems_PartialColumnFilter(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchItemsAction{conn: conn}

	// Provide column_id without column_value.
	params, _ := json.Marshal(map[string]string{
		"board_id":  "9876",
		"column_id": "status",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.search_items",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error when column_id is set without column_value")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}

	// Provide column_value without column_id.
	params2, _ := json.Marshal(map[string]string{
		"board_id":     "9876",
		"column_value": "Done",
	})

	_, err = action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.search_items",
		Parameters:  params2,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error when column_value is set without column_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchItems_NonNumericBoardID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchItemsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"board_id": "abc-123",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.search_items",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for non-numeric board_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchItems_EmptyBoard(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"boards": []any{},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &searchItemsAction{conn: conn}

	params, _ := json.Marshal(searchItemsParams{
		BoardID: "9876",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.search_items",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.Count != 0 {
		t.Errorf("expected count 0, got %d", data.Count)
	}
}

func TestSearchItems_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchItemsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.search_items",
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
