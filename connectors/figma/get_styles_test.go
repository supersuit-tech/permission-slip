package figma

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetStyles_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/files/abc123DEF/styles" {
			t.Errorf("expected path /files/abc123DEF/styles, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(getStylesResponse{
			Meta: struct {
				Styles []figmaStyle `json:"styles"`
			}{
				Styles: []figmaStyle{
					{
						Key:       "style-key-1",
						FileKey:   "abc123DEF",
						NodeID:    "1:5",
						StyleType: "FILL",
						Name:      "Primary Blue",
					},
					{
						Key:       "style-key-2",
						FileKey:   "abc123DEF",
						NodeID:    "1:6",
						StyleType: "TEXT",
						Name:      "Heading 1",
					},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getStylesAction{conn: conn}

	params, _ := json.Marshal(getStylesParams{FileKey: "abc123DEF"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.get_styles",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Styles []figmaStyle `json:"styles"`
		Count  int          `json:"count"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.Count != 2 {
		t.Errorf("expected count 2, got %d", data.Count)
	}
	if len(data.Styles) != 2 {
		t.Fatalf("expected 2 styles, got %d", len(data.Styles))
	}
	if data.Styles[0].Name != "Primary Blue" {
		t.Errorf("expected first style 'Primary Blue', got %q", data.Styles[0].Name)
	}
}

func TestGetStyles_URLExtraction(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/files/abc123DEF/styles" {
			t.Errorf("expected path /files/abc123DEF/styles, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(getStylesResponse{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getStylesAction{conn: conn}

	// Pass a full Figma URL — the key should be extracted.
	params, _ := json.Marshal(map[string]string{
		"file_key": "https://www.figma.com/design/abc123DEF/My-Design",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.get_styles",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetStyles_MissingFileKey(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getStylesAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.get_styles",
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
