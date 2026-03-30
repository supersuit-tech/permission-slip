package figma

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetVariables_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/files/abc123DEF/variables/local" {
			t.Errorf("expected path /files/abc123DEF/variables/local, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(figmaVariablesResponse{
			Meta: figmaVariablesMeta{
				Variables: map[string]figmaVariable{
					"var-1": {
						ID:                   "var-1",
						Name:                 "color/primary",
						ResolvedType:         "COLOR",
						VariableCollectionID: "col-1",
					},
					"var-2": {
						ID:                   "var-2",
						Name:                 "spacing/md",
						ResolvedType:         "FLOAT",
						VariableCollectionID: "col-1",
					},
				},
				VariableCollections: map[string]figmaVariableCollection{
					"col-1": {
						ID:   "col-1",
						Name: "Design Tokens",
						Modes: []figmaVariableMode{
							{ModeID: "mode-1", Name: "Light"},
							{ModeID: "mode-2", Name: "Dark"},
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getVariablesAction{conn: conn}

	params, _ := json.Marshal(getVariablesParams{FileKey: "abc123DEF"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.get_variables",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		VariableCount    int `json:"variable_count"`
		CollectionCount  int `json:"collection_count"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.VariableCount != 2 {
		t.Errorf("expected variable_count 2, got %d", data.VariableCount)
	}
	if data.CollectionCount != 1 {
		t.Errorf("expected collection_count 1, got %d", data.CollectionCount)
	}
}

func TestGetVariables_MissingFileKey(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getVariablesAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.get_variables",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestGetVariables_URLExtraction(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/files/abc123DEF/variables/local" {
			t.Errorf("expected path /files/abc123DEF/variables/local, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(figmaVariablesResponse{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getVariablesAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"file_key": "https://www.figma.com/design/abc123DEF/My-Design",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.get_variables",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
