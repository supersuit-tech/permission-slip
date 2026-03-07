package docusign

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListTemplates_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.URL.Path; got != "/accounts/test-account-id-456/templates" {
			t.Errorf("expected path /accounts/test-account-id-456/templates, got %s", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listTemplatesResponse{
			EnvelopeTemplates: []templateSummary{
				{
					TemplateID:  "tpl-1",
					Name:        "NDA Template",
					Description: "Non-disclosure agreement",
				},
				{
					TemplateID:  "tpl-2",
					Name:        "Employment Contract",
					Description: "Standard employment agreement",
				},
			},
			ResultSetSize: "2",
			TotalSetSize:  "2",
			StartPosition: "0",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listTemplatesAction{conn: conn}

	params, _ := json.Marshal(listTemplatesParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.list_templates",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	var templates []map[string]string
	if err := json.Unmarshal(data["templates"], &templates); err != nil {
		t.Fatalf("failed to unmarshal templates: %v", err)
	}
	if len(templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(templates))
	}
	if templates[0]["template_id"] != "tpl-1" {
		t.Errorf("expected template_id tpl-1, got %q", templates[0]["template_id"])
	}
	if templates[0]["name"] != "NDA Template" {
		t.Errorf("expected name NDA Template, got %q", templates[0]["name"])
	}
}

func TestListTemplates_WithSearchText(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("search_text"); got != "NDA" {
			t.Errorf("expected search_text=NDA, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listTemplatesResponse{
			EnvelopeTemplates: []templateSummary{},
			ResultSetSize:     "0",
			TotalSetSize:      "0",
			StartPosition:     "0",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listTemplatesAction{conn: conn}

	params, _ := json.Marshal(listTemplatesParams{SearchText: "NDA"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.list_templates",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListTemplates_InvalidCount(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listTemplatesAction{conn: conn}

	params, _ := json.Marshal(listTemplatesParams{Count: 200})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.list_templates",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid count")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
